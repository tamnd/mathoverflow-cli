package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/mathoverflow-cli/mathoverflow"
)

func (a *App) answersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "answers <question-id>",
		Short: "List answers for a question",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return codeError(exitUsage, err)
			}
			n := a.effectiveLimit(10)
			a.progressf("fetching answers for question %d...", id)
			ans, err := a.client.Answers(cmd.Context(), id, mathoverflow.AnswerOptions{
				Limit: n,
			})
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(ans, len(ans))
		},
	}
	return cmd
}
