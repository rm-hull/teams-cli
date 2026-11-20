package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.com/rm-hull/teams-cli/internal/graph"
)

func newChatDetailsCommand(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "chat-details CHAT_ID",
		Short: "Get chat metadata by ID",
		Long:  "Return the Microsoft Graph JSON object for one chat (not its messages).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			request, err := graph.ChatDetailsRequest(args[0])
			if err != nil {
				return fmt.Errorf("invalid CHAT_ID: %w", err)
			}
			return runGraphCommand(cmd, a, request)
		},
	}
}
