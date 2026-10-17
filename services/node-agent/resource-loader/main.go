package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sync"
	"sync/atomic"

	"orcn/services/node-agent/resources"
)

func main() {
	planPath := os.Getenv("ORCN_RESOURCE_PLAN")
	if planPath == "" {
		planPath = "/run/resource-plan.json"
	}
	data, err := os.ReadFile(planPath)
	if err != nil {
		log.Fatalf("read resource plan: %v", err)
	}
	var plan resources.Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		log.Fatalf("parse resource plan: %v", err)
	}
	if plan.Version == 0 {
		plan.Version = 1
	}
	registry := newInstallerRegistry()
	if err := registry.Register("http", installHTTP); err != nil {
		log.Fatalf("register HTTP resource installer: %v", err)
	}
	if err := registry.Register("https", installHTTP); err != nil {
		log.Fatalf("register HTTPS resource installer: %v", err)
	}
	if err := registry.Register("hf", installHF); err != nil {
		log.Fatalf("register HF resource installer: %v", err)
	}
	if err := registry.Register("file", installFile); err != nil {
		log.Fatalf("register file resource installer: %v", err)
	}
	if err := registry.Register("git", installGit); err != nil {
		log.Fatalf("register git resource installer: %v", err)
	}
	if err := registry.Register("s3", installS3); err != nil {
		log.Fatalf("register s3 resource installer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	var hasErr atomic.Bool

	for _, resource := range plan.Resources {
		res := resource
		id := res.ID
		if id == "" {
			id = res.Type
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Printf("resource installing id=%s type=%s destination=%s", id, res.Type, res.Destination)
			progress := func(done, total int64) {
				if total > 0 {
					log.Printf("resource progress id=%s downloaded_bytes=%d total_bytes=%d percentage=%.2f", id, done, total, float64(done)*100/float64(total))
					return
				}
				log.Printf("resource progress id=%s downloaded_bytes=%d", id, done)
			}
			if err := registry.Install(ctx, res, progress); err != nil {
				log.Printf("resource failed id=%s error=%v", id, err)
				hasErr.Store(true)
				cancel()
				return
			}
			log.Printf("resource installed id=%s", id)
		}()
	}
	wg.Wait()

	if hasErr.Load() {
		os.Exit(1)
	}
}
