package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.com/rm-hull/teams-cli/internal/graph"
)

func newListMessagesCommand(a *app) *cobra.Command {
	var limit int
	var nextLink string
	command := &cobra.Command{
		Use:   "list-messages CHAT_ID",
		Short: "List one page of messages in a chat",
		Long:  "Return one page of messages, newest first. Message bodies may contain HTML.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
			request, err := graph.ListMessagesRequest(args[0], pageSize, nextLink)
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
