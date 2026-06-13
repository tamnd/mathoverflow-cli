package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/mathoverflow-cli/mathoverflow"
)

func (a *App) questionsCmd() *cobra.Command {
	var (
		sort string
		tag  string
	)
	cmd := &cobra.Command{
		Use:   "questions",
		Short: "List MathOverflow questions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(10)
			a.progressf("fetching %d questions (sort=%s)...", n, sort)
			qs, err := a.client.Questions(cmd.Context(), mathoverflow.QuestionOptions{
				Sort:  sort,
				Tag:   tag,
				Limit: n,
			})
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(qs, len(qs))
		},
	}
	cmd.Flags().StringVar(&sort, "sort", "votes", "sort order: votes|activity|newest")
	cmd.Flags().StringVar(&tag, "tag", "", "filter by tag (e.g. nt.number-theory)")
	return cmd
}

func (a *App) questionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "question <id>",
		Short: "Fetch a single question by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return codeError(exitUsage, err)
			}
			a.progressf("fetching question %d...", id)
			q, err := a.client.Question(cmd.Context(), id)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.render([]mathoverflow.Question{q})
		},
	}
}
