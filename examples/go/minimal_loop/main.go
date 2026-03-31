package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/pluribus-ai/pluribus/sdk/go/pluribus"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client := pluribus.NewClient("http://127.0.0.1:8123", "")

	// 1) recall_context (pre-action)
	bundle, err := client.RecallContext(ctx, "Refactor auth middleware safely", pluribus.RecallContextOpts{
		Tags:          []string{"auth"},
		CorrelationID: "session-123",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 2) plan / reason (use mcp_context + recall_bundle in a real agent)
	_ = bundle

	// 3) act (your tools, edits, commits)
	fmt.Printf("recall: %d constraints, %d failures — act phase.\n",
		len(bundle.GoverningConstraints), len(bundle.KnownFailures))

	// 4) record_experience (post-action)
	episode, err := client.RecordExperience(ctx, "Fixed race in session refresh; added test coverage.", pluribus.RecordExperienceOpts{
		Tags:          []string{"auth", "incident"},
		CorrelationID: "session-123",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("recorded advisory episode %s (deduplicated=%v)\n", episode.ID, episode.Deduplicated)
}
