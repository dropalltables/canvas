package cmd

import (
	"os"

	"github.com/dropalltables/canvas/internal/api"
	"github.com/dropalltables/canvas/internal/config"
	"github.com/dropalltables/canvas/internal/tui"
	"github.com/dropalltables/canvas/internal/ui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "canvas",
	Short: "Canvas LMS CLI",
	Long:  "A CLI for managing Canvas LMS assignments",
	RunE:  runTUI,
}

func runTUI(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		if err == config.ErrNotConfigured {
			ui.Error("Not logged in. Run 'canvas auth login' to authenticate.")
			return err
		}
		return err
	}
	client := api.NewClient(cfg)
	return tui.Run(client)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(assignmentsCmd)
}
