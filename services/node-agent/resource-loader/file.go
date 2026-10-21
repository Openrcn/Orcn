package main

import (
	"context"
	"fmt"
	"path/filepath"
	"os"
	"strconv"

	"orcn/services/node-agent/resources"
)

func installFile(ctx context.Context, resource resources.Resource, progress progressFunc) error {
	content, ok := resource.Config["content"].(string)
	if !ok || content == "" {
		return fmt.Errorf("file resource requires 'content' in config")
	}

	perm := os.FileMode(0o644)
	if permStr, ok := resource.Config["permissions"].(string); ok && permStr != "" {
		if parsed, err := strconv.ParseUint(permStr, 8, 32); err == nil {
			perm = os.FileMode(parsed)
		}
	}

	// Destination is the full file path, create parent directory
	parentDir := filepath.Dir(resource.Destination)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return fmt.Errorf("create directory for file: %w", err)
	}

	if err := os.WriteFile(resource.Destination, []byte(content), perm); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	if progress != nil {
		progress(int64(len(content)), int64(len(content)))
	}

	return nil
}
