package auth

import (
	"errors"
	"fmt"
	"strings"
)

var cliAuthErrorCodes = map[string]struct{}{
	"cli_token_revoked": {},
	"invalid_cli_token": {},
	"expired_cli_token": {},
}

func IsCliAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for code := range cliAuthErrorCodes {
		if strings.Contains(msg, code) {
			return true
		}
	}
	return false
}

func HandleAuthRevocation(err error) error {
	if !IsCliAuthError(err) {
		return err
	}
	_ = ClearCredentials()
	return fmt.Errorf("login expired (your password may have been reset); run `mcpzero login` to sign in again: %w", err)
}

func IsCliTokenRevokedReason(reason string) bool {
	return strings.Contains(strings.ToLower(reason), "cli_token_revoked")
}

func HandleRevokedDisconnect(reason string) error {
	if !IsCliTokenRevokedReason(reason) {
		return errors.New(reason)
	}
	_ = ClearCredentials()
	return fmt.Errorf("login expired (your password may have been reset); run `mcpzero login` to sign in again")
}
