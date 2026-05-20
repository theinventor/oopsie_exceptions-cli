package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

//go:embed embedded/oopsie_skill.md
var oopsieSkill string

func newSkillCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "skill",
		Short: "Print bundled agent workflow docs",
	}
	c.AddCommand(newSkillGetCmd())
	return c
}

func newSkillGetCmd() *cobra.Command {
	var output string
	c := &cobra.Command{
		Use:   "get oopsie",
		Short: "Get the bundled Oopsie agent skill",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 || args[0] != "oopsie" {
				return usageError("usage: oopsie skill get oopsie")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if output == "" || output == "-" {
				_, err := fmt.Fprint(cmd.OutOrStdout(), oopsieSkill)
				return err
			}
			if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
				return err
			}
			return os.WriteFile(output, []byte(oopsieSkill), 0o644)
		},
	}
	c.Flags().StringVarP(&output, "output", "o", "", "write skill docs to this file instead of stdout")
	return c
}
