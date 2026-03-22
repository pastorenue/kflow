package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/pastorenue/kflow/cmd/kflow/uiassets"
	"github.com/spf13/cobra"
)

var uiCmd = &cobra.Command{
	Use:   "ui <port>",
	Short: "Serve the kflow dashboard locally and open it in the browser",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		port := args[0]

		token := loadSavedToken()
		if token == "" {
			key := resolveAPIKey(cmd)
			if key != "" {
				result, err := doJSONNoAuth("POST", "/api/v1/auth/token", map[string]string{"api_key": key})
				if err != nil {
					return fmt.Errorf("exchange api key for token: %w", err)
				}
				token, _ = result["token"].(string)
			}
		}

		targetURL := fmt.Sprintf("http://localhost:%s/", port)
		if token != "" {
			targetURL = fmt.Sprintf("http://localhost:%s/?token=%s", port, token)
		}

		mux := http.NewServeMux()
		mux.Handle("/", http.FileServer(http.FS(uiassets.FS)))

		go func() {
			if err := http.ListenAndServe(":"+port, mux); err != nil {
				fmt.Printf("ui server error: %v\n", err)
			}
		}()

		fmt.Printf("Dashboard: %s\n", targetURL)
		openBrowser(targetURL)

		select {}
	},
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	_ = exec.Command(cmd, args...).Start()
}
