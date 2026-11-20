package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"gitlab.com/rm-hull/teams-cli/internal/graph"
)

func newSearchCommand(a *app) *cobra.Command {
	var offset, limit int
	command := &cobra.Command{
		Use:   "search QUERY",
		Short: "Search Teams messages using Microsoft Graph beta",
		Long:  "Search chat messages visible to the signed-in user. Increase --offset by --limit when more results are available.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(args[0]) == "" {
				return graph.ErrBlankSearch
			}
			request, err := graph.SearchRequest(args[0], offset, limit)
			if err != nil {
				return err
			}
			return runGraphCommand(cmd, a, request)
		},
	}
	command.Flags().IntVar(&offset, "offset", 0, "Zero-based result offset (0 or greater)")
	command.Flags().IntVar(&limit, "limit", 20, "Requested result count (1-50)")
	return command
}
