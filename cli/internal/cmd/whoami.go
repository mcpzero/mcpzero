package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/mcpzero/mcpzero/cli/internal/auth"
	"github.com/mcpzero/mcpzero/cli/internal/cloud"
	"github.com/mcpzero/mcpzero/cli/internal/config"
)

func whoami(args []string) error {
	fs := flag.NewFlagSet("whoami", flag.ExitOnError)
	showLimits := fs.Bool("limits", false, "Show plan quotas and limits")
	if err := fs.Parse(args); err != nil {
		return err
	}

	creds, err := auth.LoadCredentials()
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "email: %s\n", creds.Email)
	fmt.Fprintf(os.Stdout, "user_id: %s\n", creds.UserID)
	fmt.Fprintf(os.Stdout, "gw_base: %s\n", creds.GWBase)
	fmt.Fprintf(os.Stdout, "web_base: %s\n", creds.WebBase)

	if !*showLimits {
		return nil
	}

	web := firstNonEmpty(creds.WebBase, config.DefaultWebBase)
	client := cloud.NewClient(web, creds.RefreshToken)
	me, err := client.Me(context.Background())
	if err != nil {
		return fmt.Errorf("fetch account limits: %w", err)
	}
	fmt.Fprintln(os.Stdout)
	printWhoamiLimits(me)
	return nil
}
