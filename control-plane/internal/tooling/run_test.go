package tooling

import (
	"context"
	"testing"
)

func TestRunBuild_success(t *testing.T) {
	ctx := context.Background()
	output, exitCode, err := RunBuild(ctx, ".", nil)
	if err != nil {
		t.Fatalf("RunBuild: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exit code = %d, output: %s", exitCode, output)
	}
}
