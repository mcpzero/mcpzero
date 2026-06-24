package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mcpzero/mcpzero/cli/internal/auth"
	"github.com/mcpzero/mcpzero/cli/internal/config"
)

func login(args []string) error {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	webBase := fs.String("web-base", config.DefaultWebBase, "MCPZERO web base URL")
	gwBase := fs.String("gw-base", config.DefaultGWBase, "MCPZERO gateway base URL")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	creds, err := auth.Login(ctx, auth.LoginOptions{
		WebBase: *webBase,
		GWBase:  *gwBase,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Login successful as %s (%s)\n", creds.Email, creds.UserID)
	fmt.Fprintf(os.Stdout, "Credentials saved. You can now run tunnel start without --token.\n")
	return nil
}

func logout(_ []string) error {
	if err := auth.ClearCredentials(); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, "Logged out.")
	return nil
}

func whoami(_ []string) error {
	creds, err := auth.LoadCredentials()
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "email: %s\n", creds.Email)
	fmt.Fprintf(os.Stdout, "user_id: %s\n", creds.UserID)
	fmt.Fprintf(os.Stdout, "gw_base: %s\n", creds.GWBase)
	fmt.Fprintf(os.Stdout, "web_base: %s\n", creds.WebBase)
	return nil
}
