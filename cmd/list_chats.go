package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.com/rm-hull/teams-cli/internal/graph"
)

func newListChatsCommand(a *app) *cobra.Command {
	var limit int
	var nextLink string
	command := &cobra.Command{
		Use:   "list-chats",
		Short: "List one page of your Teams chats",
		Long:  "Return a Microsoft Graph JSON object with value[] containing chats and their IDs.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			pageSize := limit
			if !cmd.Flags().Changed("limit") {
				pageSize = 20
			}
			if err := graph.ValidatePageSize(pageSize); err != nil {
				return err
			}
			hasNextLink := cmd.Flags().Changed("next-link")
			if hasNextLink && cmd.Flags().Changed("limit") {
				return fmt.Errorf("--limit cannot be combined with --next-link")
			}
			if hasNextLink && nextLink == "" {
				return graph.ValidateNextLink(nextLink)
			}
			request, err := graph.ListChatsRequest(pageSize, nextLink)
			if err != nil {
				return err
			}
			return runGraphCommand(cmd, a, request)
		},
	}
	command.Flags().IntVar(&limit, "limit", 20, "Requested first-page size (1-50)")
	command.Flags().StringVar(&nextLink, "next-link", "", "Fetch a previous response's @odata.nextLink")
	return command
}

func runGraphCommand(cmd *cobra.Command, a *app, request graph.Request) error {
	token, err := a.accessToken(cmd.Context(), true, cmd.ErrOrStderr())
	if err != nil {
		return err
	}
	result, err := graph.NewClient(token).Do(cmd.Context(), request)
	if err != nil {
		return err
	}
	return printJSON(cmd.OutOrStdout(), result)
}
