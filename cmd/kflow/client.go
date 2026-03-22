package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// tokenFilePath returns the path to the persisted session token file.
func tokenFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".kflow", "token")
}

// saveToken writes the token to ~/.kflow/token with mode 0600.
func saveToken(token string) error {
	dir := filepath.Dir(tokenFilePath())
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create token dir: %w", err)
	}
	return os.WriteFile(tokenFilePath(), []byte(token), 0600)
}

// loadSavedToken reads ~/.kflow/token, returning "" on any error.
func loadSavedToken() string {
	p := tokenFilePath()
	if p == "" {
		return ""
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return string(b)
}

// resolveBearer returns the bearer token to use: --api-key flag, then saved token.
func resolveBearer() string {
	if apiKeyFlag != "" {
		return apiKeyFlag
	}
	return loadSavedToken()
}

func doJSON(method, path string, body any) (map[string]any, error) {
	return doJSONWithBearer(method, path, body, resolveBearer())
}

// doJSONNoAuth sends a request without any Authorization header.
func doJSONNoAuth(method, path string, body any) (map[string]any, error) {
	return doJSONWithBearer(method, path, body, "")
}

func doJSONWithBearer(method, path string, body any, bearer string) (map[string]any, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, serverFlag+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("server error %d: %s", resp.StatusCode, string(raw))
	}

	var result map[string]any
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &result); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}
	}
	return result, nil
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
