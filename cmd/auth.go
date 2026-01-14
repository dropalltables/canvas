package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/dropalltables/canvas/internal/config"
	"github.com/dropalltables/canvas/internal/ui"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Canvas",
	RunE:  runLogin,
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored credentials",
	RunE:  runLogout,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE:  runStatus,
}

func init() {
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(statusCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Canvas URL (e.g., https://school.instructure.com): ")
	baseURL, _ := reader.ReadString('\n')
	baseURL = strings.TrimSpace(baseURL)

	if baseURL == "" {
		ui.Error("URL is required")
		return nil
	}

	fmt.Print("API Token: ")
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)

	if token == "" {
		ui.Error("Token is required")
		return nil
	}

	cfg := &config.Config{
		BaseURL: baseURL,
		Token:   token,
	}

	if err := config.Save(cfg); err != nil {
		ui.Error("Failed to save: %v", err)
		return err
	}

	fmt.Printf("Logged in to %s\n", baseURL)
	return nil
}

func runLogout(cmd *cobra.Command, args []string) error {
	path := config.Path()
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Not logged in")
			return nil
		}
		return err
	}
	fmt.Println("Logged out")
	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		if err == config.ErrNotConfigured {
			fmt.Println("Not logged in")
			return nil
		}
		ui.Error("Failed: %v", err)
		return err
	}

	fmt.Printf("Logged in: %s\n", cfg.BaseURL)

	if os.Getenv("CANVAS_BASE_URL") != "" && os.Getenv("CANVAS_TOKEN") != "" {
		fmt.Println("Source: environment")
	} else {
		fmt.Printf("Source: %s\n", config.Path())
	}

	return nil
}
