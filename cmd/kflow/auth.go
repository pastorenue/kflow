package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with the orchestrator",
}

var authTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Generate a 24-hour session token (prints to stdout)",
	RunE: func(cmd *cobra.Command, args []string) error {
		key := resolveAPIKey(cmd)
		if key == "" {
			return fmt.Errorf("--api-key flag or KFLOW_API_KEY env var required")
		}
		result, err := doJSONNoAuth("POST", "/api/v1/auth/token", map[string]string{"api_key": key})
		if err != nil {
			return err
		}
		token, _ := result["token"].(string)
		fmt.Println(token)
		return nil
	},
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate and save token to ~/.kflow/token",
	RunE: func(cmd *cobra.Command, args []string) error {
		key := resolveAPIKey(cmd)
		if key == "" {
			return fmt.Errorf("--api-key flag or KFLOW_API_KEY env var required")
		}
		result, err := doJSONNoAuth("POST", "/api/v1/auth/token", map[string]string{"api_key": key})
		if err != nil {
			return err
		}
		token, _ := result["token"].(string)
		if err := saveToken(token); err != nil {
			return err
		}
		fmt.Println("Logged in. Token saved to ~/.kflow/token")
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Delete the saved token (~/.kflow/token)",
	RunE: func(cmd *cobra.Command, args []string) error {
		p := tokenFilePath()
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove token: %w", err)
		}
		fmt.Println("Logged out.")
		return nil
	},
}

// resolveAPIKey returns the API key from the --api-key flag or KFLOW_API_KEY env var.
func resolveAPIKey(_ *cobra.Command) string {
	if apiKeyFlag != "" {
		return apiKeyFlag
	}
	return os.Getenv("KFLOW_API_KEY")
}

func init() {
	authCmd.AddCommand(authTokenCmd, authLoginCmd, authLogoutCmd)
}
