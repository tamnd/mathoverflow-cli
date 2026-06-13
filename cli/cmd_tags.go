package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/mathoverflow-cli/mathoverflow"
)

func (a *App) tagsCmd() *cobra.Command {
	var search string
	cmd := &cobra.Command{
		Use:   "tags",
		Short: "List popular MathOverflow tags",
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(20)
			a.progressf("fetching tags (limit=%d)...", n)
			tags, err := a.client.Tags(cmd.Context(), mathoverflow.TagOptions{
				Limit:  n,
				Search: search,
			})
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(tags, len(tags))
		},
	}
	cmd.Flags().StringVar(&search, "search", "", "filter tags by name substring")
	return cmd
}
