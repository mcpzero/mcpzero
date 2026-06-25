package auth

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/mcpzero/mcpzero/cli/internal/config"
)

type tokenRequest struct {
	Code       string `json:"code"`
	State      string `json:"state"`
	DeviceName string `json:"device_name"`
}

type tokenResponse struct {
	RefreshToken string `json:"refresh_token"`
	UserID       string `json:"user_id"`
	Email        string `json:"email"`
	GWBase       string `json:"gw_base"`
	WebBase      string `json:"web_base"`
}

type callbackPayload struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

type LoginOptions struct {
	WebBase string
	GWBase  string
	Timeout time.Duration
	// NoBrowser disables the automatic browser launch and instead prints
	// step-by-step instructions, then waits for the user to paste the callback
	// URL (or code) on stdin. Useful in containers, remote shells, or any
	// environment without a usable local browser.
	NoBrowser bool
}

func Login(ctx context.Context, opts LoginOptions) (*Credentials, error) {
	webBase := opts.WebBase
	if webBase == "" {
		webBase = config.DefaultWebBase
	}
	gwBase := opts.GWBase
	if gwBase == "" {
		gwBase = config.DefaultGWBase
	}
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	state, err := randomState()
	if err != nil {
		return nil, err
	}

	resultCh := make(chan callbackPayload, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	server := &http.Server{Handler: mux}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		gotState := r.URL.Query().Get("state")
		if code == "" || gotState == "" {
			http.Error(w, "missing code or state", http.StatusBadRequest)
			errCh <- fmt.Errorf("callback missing code or state")
			return
		}
		if gotState != state {
			http.Error(w, "invalid state", http.StatusBadRequest)
			errCh <- fmt.Errorf("callback state mismatch")
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, `<!doctype html><html><body style="font-family:system-ui;padding:2rem"><h1>Login successful</h1><p>You can close this window and return to the terminal.</p></body></html>`)
		resultCh <- callbackPayload{Code: code, State: gotState}
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen localhost: %w", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port

	go func() {
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("callback server: %w", err)
		}
	}()

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	loginURL, err := url.Parse(webBase)
	if err != nil {
		return nil, fmt.Errorf("parse web base: %w", err)
	}
	loginURL.Path = "/app/cli-auth"
	q := loginURL.Query()
	q.Set("state", state)
	q.Set("port", fmt.Sprintf("%d", port))
	loginURL.RawQuery = q.Encode()

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// In no-browser mode we still keep the local callback server running (so the
	// flow auto-completes when the redirect is reachable, e.g. `docker run
	// --network host`), but we primarily guide the user to paste the code back.
	pasteCh := make(chan callbackPayload, 1)
	if opts.NoBrowser {
		printManualLoginInstructions(loginURL.String(), port)
		go readPastedCode(waitCtx, state, pasteCh, errCh)
	} else {
		fmt.Fprintf(os.Stderr, "Opening browser for login: %s\n", loginURL.String())
		if err := openBrowser(loginURL.String()); err != nil {
			fmt.Fprintf(os.Stderr, "Could not open browser automatically: %v\n", err)
			fmt.Fprintf(os.Stderr, "Open this URL manually, or re-run with --no-browser:\n%s\n", loginURL.String())
		}
	}

	var payload callbackPayload
	select {
	case <-waitCtx.Done():
		if opts.NoBrowser {
			return nil, fmt.Errorf("login timed out waiting for the pasted code (the code is valid for ~60s — try again)")
		}
		return nil, fmt.Errorf("login timed out waiting for browser callback")
	case err := <-errCh:
		return nil, err
	case payload = <-resultCh:
	case payload = <-pasteCh:
	}

	deviceName, _ := os.Hostname()
	tokenResp, err := exchangeCode(waitCtx, webBase, tokenRequest{
		Code:       payload.Code,
		State:      payload.State,
		DeviceName: deviceName,
	})
	if err != nil {
		return nil, err
	}

	creds := Credentials{
		RefreshToken: tokenResp.RefreshToken,
		UserID:       tokenResp.UserID,
		Email:        tokenResp.Email,
		GWBase:       firstNonEmpty(tokenResp.GWBase, gwBase),
		WebBase:      firstNonEmpty(tokenResp.WebBase, webBase),
	}

	if err := SaveCredentials(creds); err != nil {
		return nil, err
	}

	return &creds, nil
}

func exchangeCode(ctx context.Context, webBase string, req tokenRequest) (*tokenResponse, error) {
	endpoint, err := url.Parse(webBase)
	if err != nil {
		return nil, fmt.Errorf("parse web base: %w", err)
	}
	endpoint.Path = "/app/api/cli/token"

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		if isTLSTrustError(err) {
			return nil, fmt.Errorf("token exchange request: %w\n"+
				"  TLS verification failed — this machine is likely missing root CA certificates\n"+
				"  (common in minimal containers). Install them and retry, e.g.:\n"+
				"    Debian/Ubuntu: apt-get update && apt-get install -y ca-certificates\n"+
				"    Alpine:        apk add --no-cache ca-certificates\n"+
				"    RHEL/Fedora:   dnf install -y ca-certificates", err)
		}
		return nil, fmt.Errorf("token exchange request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		snippet := string(respBody)
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		return nil, fmt.Errorf("token exchange failed (HTTP %d): %s", resp.StatusCode, snippet)
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		snippet := string(respBody)
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		return nil, fmt.Errorf("parse token response (HTTP %d, expected JSON): %w — body: %s", resp.StatusCode, err, snippet)
	}
	if tokenResp.RefreshToken == "" {
		return nil, fmt.Errorf("token response missing refresh_token")
	}
	return &tokenResp, nil
}

func randomState() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func openBrowser(target string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", target).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", target).Start()
	default:
		return exec.Command("xdg-open", target).Start()
	}
}

// printManualLoginInstructions explains, step by step, exactly what the user
// must do to complete login when no local browser is available.
func printManualLoginInstructions(loginURL string, port int) {
	w := os.Stderr
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "── Manual login (--no-browser) ──────────────────────────────────────")
	fmt.Fprintln(w, "This machine has no usable browser (e.g. a Docker container or remote")
	fmt.Fprintln(w, "shell), so finish the login by hand:")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  1. Open this URL in a browser on ANY machine (e.g. your laptop):")
	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "       %s\n", loginURL)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  2. Sign in. Your browser will then try to redirect to a URL like:")
	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "       http://127.0.0.1:%d/callback?code=...&state=...\n", port)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "     That page will FAIL to load — this is expected here, don't worry.")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  3. Copy the FULL redirected URL from the address bar (or just the")
	fmt.Fprintln(w, "     value after code=), paste it below, and press Enter.")
	fmt.Fprintln(w, "     The code expires ~60 seconds after you sign in, so be quick.")
	fmt.Fprintln(w, "─────────────────────────────────────────────────────────────────────")
}

// readPastedCode prompts the user (repeatedly, on parse errors) for the pasted
// callback URL or raw code, and forwards the parsed result on out.
func readPastedCode(ctx context.Context, expectedState string, out chan<- callbackPayload, errCh chan<- error) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Fprint(os.Stderr, "\nPaste callback URL or code, then Enter:\n> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				if strings.TrimSpace(line) == "" {
					sendErr(errCh, fmt.Errorf("no input received on stdin — run `mcpzero login --no-browser` in an interactive terminal"))
					return
				}
			} else {
				sendErr(errCh, fmt.Errorf("read stdin: %w", err))
				return
			}
		}

		code, perr := parsePastedCode(line, expectedState)
		if perr != nil {
			fmt.Fprintf(os.Stderr, "  invalid input: %v — please try again.\n", perr)
			if err == io.EOF {
				sendErr(errCh, perr)
				return
			}
			continue
		}

		select {
		case out <- callbackPayload{Code: code, State: expectedState}:
		case <-ctx.Done():
		}
		return
	}
}

func sendErr(errCh chan<- error, err error) {
	select {
	case errCh <- err:
	default:
	}
}

// parsePastedCode accepts either a full callback URL
// (http://127.0.0.1:PORT/callback?code=...&state=...) or a bare code value and
// returns the code. When a state is present in a pasted URL it must match.
func parsePastedCode(input, expectedState string) (string, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", fmt.Errorf("empty input")
	}

	if strings.Contains(s, "code=") || strings.Contains(s, "/callback") || strings.Contains(s, "://") {
		raw := s
		if !strings.Contains(raw, "://") {
			raw = "http://" + strings.TrimPrefix(raw, "//")
		}
		if u, err := url.Parse(raw); err == nil {
			if code := u.Query().Get("code"); code != "" {
				if gotState := u.Query().Get("state"); gotState != "" && expectedState != "" && gotState != expectedState {
					return "", fmt.Errorf("state in pasted URL does not match this login session")
				}
				return code, nil
			}
		}
	}

	return s, nil
}

// isTLSTrustError reports whether err is a TLS certificate-trust failure,
// typically caused by a missing system CA bundle (common in minimal containers).
func isTLSTrustError(err error) bool {
	var unknownAuthority x509.UnknownAuthorityError
	var certInvalid x509.CertificateInvalidError
	var hostnameErr x509.HostnameError
	if errors.As(err, &unknownAuthority) || errors.As(err, &certInvalid) || errors.As(err, &hostnameErr) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "certificate signed by unknown authority") ||
		strings.Contains(msg, "failed to verify certificate")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
