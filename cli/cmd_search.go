package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/mathoverflow-cli/mathoverflow"
)

func (a *App) searchCmd() *cobra.Command {
	var sort string
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search MathOverflow questions by title keyword",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n := a.effectiveLimit(10)
			a.progressf("searching for %q (sort=%s)...", args[0], sort)
			qs, err := a.client.Search(cmd.Context(), mathoverflow.SearchOptions{
				Query: args[0],
				Sort:  sort,
				Limit: n,
			})
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(qs, len(qs))
		},
	}
	cmd.Flags().StringVar(&sort, "sort", "votes", "sort order: votes|activity|newest")
	return cmd
}
