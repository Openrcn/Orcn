package main

import (
	"context"
	"fmt"

	"orcn/services/node-agent/resources"
)

func installFile(ctx context.Context, resource resources.Resource, progress progressFunc) error {
	return fmt.Errorf("inline file resource installer not yet implemented")
}
