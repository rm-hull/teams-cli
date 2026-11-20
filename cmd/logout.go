package cmd

import (
	"github.com/spf13/cobra"
)

func newLogoutCommand(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove this app and tenant's local sign-in cache",
		Long:  "Remove cached credentials locally without revoking server-side sessions.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := a.logout(cmd.Context()); err != nil {
				return err
			}
			return printJSON(cmd.OutOrStdout(), map[string]string{"status": "signed_out"})
		},
	}
}
