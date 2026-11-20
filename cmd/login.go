package cmd

import (
	"github.com/spf13/cobra"
)

func newLoginCommand(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Sign in with a device code",
		Long:  "Print device-code instructions to stderr, wait for browser sign-in, and cache credentials locally.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if _, err := a.authenticate(cmd.Context(), true, cmd.ErrOrStderr()); err != nil {
				return err
			}
			return printJSON(cmd.OutOrStdout(), map[string]string{"status": "signed_in"})
		},
	}
}
