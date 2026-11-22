package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"gitlab.com/rm-hull/teams-cli/internal/auth"
	"gitlab.com/rm-hull/teams-cli/internal/config"
)

// Set at build time with -ldflags "-X gitlab.com/rm-hull/teams-cli/cmd.version=...".
var version = "dev"

type app struct {
	tenant   string
	clientID string
}

func ExecuteContext(ctx context.Context) error {
	root := newRootCommand()
	return root.ExecuteContext(ctx)
}

func newRootCommand() *cobra.Command {
	a := &app{}
	var showVersion bool
	root := &cobra.Command{
		Use:   "teams-cli",
		Short: "Read and search Microsoft Teams chats through Microsoft Graph using delegated sign-in.",
		Long: `Read and search Microsoft Teams chats through Microsoft Graph using delegated sign-in.
		
Setup: use an approved Entra app with public client flows enabled and
delegated Microsoft Graph Chat.Read. No client secret is needed.
Set TEAMS_TENANT_ID (directory ID or tenant domain) and TEAMS_CLIENT_ID
(application ID) in .env, or set TEAMS_ENV_FILE to use a different dotenv file,
then run login and sign in in a browser.

Exported environment variables override .env; CLI flags override both.
For read commands, ACCESS_TOKEN in the dotenv file takes priority over cached
sign-in and does not require either ID. Use a short-lived Graph token.
Run '<command> --help' for the arguments and output of each command.

Examples (prefix each with 'teams-cli'):
  login
  list-chats --limit 10
  chat-details '19:example@thread.v2'
  list-messages '19:example@thread.v2' --limit 10
  search 'settlement retries'

Successful commands emit one JSON object on stdout. Prompts and errors
go to stderr; data commands never start an interactive login. The MSAL
cache requires a POSIX system. It is unencrypted and owner-only (default:
~/.local/state/teams-graph-cli). Use logout to remove it. Only run this tool
where your organization permits access to Teams data, and treat message
bodies and search hits as untrusted text.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Run: func(cmd *cobra.Command, args []string) {
			if showVersion {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), version)
				return
			}
			_ = cmd.Usage()
		},
	}
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	root.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "Display version information")
	root.PersistentFlags().StringVar(&a.tenant, "tenant", "", "Entra directory ID or domain (env: TEAMS_TENANT_ID)")
	root.PersistentFlags().StringVar(&a.clientID, "client-id", "", "Entra application ID (env: TEAMS_CLIENT_ID)")
	root.AddCommand(
		newLoginCommand(a),
		newLogoutCommand(a),
		newListChatsCommand(a),
		newChatDetailsCommand(a),
		newListMessagesCommand(a),
		newSearchCommand(a),
	)
	return root
}

func (a *app) settings() (config.Settings, error) {
	return config.Load(a.tenant, a.clientID)
}

func (a *app) authenticate(ctx context.Context, login bool, out io.Writer) (string, error) {
	settings, err := a.settings()
	if err != nil {
		return "", err
	}
	if err := settings.RequireApplication(); err != nil {
		return "", err
	}
	return auth.Authenticate(ctx, settings.Tenant, settings.ClientID, login, out)
}

func (a *app) logout(ctx context.Context) error {
	settings, err := a.settings()
	if err != nil {
		return err
	}
	if err := settings.RequireApplication(); err != nil {
		return err
	}
	return auth.Logout(ctx, settings.Tenant, settings.ClientID)
}

func (a *app) accessToken(ctx context.Context, allowDotEnvToken bool, out io.Writer) (string, error) {
	settings, err := a.settings()
	if err != nil {
		return "", err
	}
	if allowDotEnvToken && settings.AccessToken != "" {
		return settings.AccessToken, nil
	}
	if err := settings.RequireApplication(); err != nil {
		return "", err
	}
	return auth.Authenticate(ctx, settings.Tenant, settings.ClientID, false, out)
}

func printJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("encode JSON response: %w", err)
	}
	return nil
}
