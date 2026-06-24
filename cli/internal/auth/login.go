package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
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

	fmt.Fprintf(os.Stderr, "Opening browser for login: %s\n", loginURL.String())
	if err := openBrowser(loginURL.String()); err != nil {
		fmt.Fprintf(os.Stderr, "Could not open browser automatically: %v\n", err)
		fmt.Fprintf(os.Stderr, "Open this URL manually:\n%s\n", loginURL.String())
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var payload callbackPayload
	select {
	case <-waitCtx.Done():
		return nil, fmt.Errorf("login timed out waiting for browser callback")
	case err := <-errCh:
		return nil, err
	case payload = <-resultCh:
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

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
