package main

import (
	"context"
	"fmt"

	"orcn/services/node-agent/resources"
)

func installGit(ctx context.Context, resource resources.Resource, progress progressFunc) error {
	return fmt.Errorf("git resource installer not yet implemented")
}
