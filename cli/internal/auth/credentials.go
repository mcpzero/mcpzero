package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Credentials struct {
	RefreshToken string `json:"refresh_token"`
	UserID       string `json:"user_id"`
	Email        string `json:"email"`
	GWBase       string `json:"gw_base"`
	WebBase      string `json:"web_base"`
}

func CredentialsFilePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config dir: %w", err)
	}
	return filepath.Join(dir, "mcpzero", "credentials.json"), nil
}

func LoadCredentials() (*Credentials, error) {
	path, err := CredentialsFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not logged in (run mcpzero login)")
		}
		return nil, fmt.Errorf("read credentials: %w", err)
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("parse credentials: %w", err)
	}
	if creds.RefreshToken == "" {
		return nil, fmt.Errorf("credentials missing refresh_token")
	}
	return &creds, nil
}

func SaveCredentials(creds Credentials) error {
	path, err := CredentialsFilePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write credentials: %w", err)
	}
	return nil
}

func ClearCredentials() error {
	path, err := CredentialsFilePath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove credentials: %w", err)
	}
	return nil
}
