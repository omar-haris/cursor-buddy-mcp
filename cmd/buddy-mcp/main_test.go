package main

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunServer_Success(t *testing.T) {
	tempDir := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test that runServer initializes successfully
	err := runServer(ctx, tempDir)
	require.NoError(t, err)

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Cancel context to stop server
	cancel()
}

func TestRunServer_InvalidBuddyPath(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Test with an invalid path that should cause initialization to fail
	invalidPath := "/root/invalid/path/that/should/not/exist"

	err := runServer(ctx, invalidPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize buddy handlers")
}

func TestRunServer_EmptyBuddyPath(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Test with empty path
	err := runServer(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize buddy handlers")
}

func TestRunServer_ContextCancellation(t *testing.T) {
	tempDir := t.TempDir()

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	// Server should still initialize successfully even with cancelled context
	err := runServer(ctx, tempDir)
	require.NoError(t, err)
}

func TestRunServer_WithRealBuddyPath(t *testing.T) {
	tempDir := t.TempDir()

	// Create a real .buddy directory structure
	buddyPath := filepath.Join(tempDir, ".buddy")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test that runServer works with a real buddy path
	err := runServer(ctx, buddyPath)
	require.NoError(t, err)

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Cancel context to stop server
	cancel()
}

func TestRunServer_WithEnvironmentPath(t *testing.T) {
	tempDir := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Test with a valid temporary directory
	err := runServer(ctx, tempDir)
	require.NoError(t, err)

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Cancel context to stop server
	cancel()
}

func TestRunServer_LongRunning(t *testing.T) {
	tempDir := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Test that runServer can run for a longer period
	err := runServer(ctx, tempDir)
	require.NoError(t, err)

	// Let it run for the full timeout
	<-ctx.Done()
}

func TestRunServer_AllCodePaths(t *testing.T) {
	tempDir := t.TempDir()

	// Create all buddy directories to ensure full initialization
	buddyDirs := []string{"rules", "knowledge", "database", "todos", "history", "backups"}
	for _, dir := range buddyDirs {
		err := os.MkdirAll(filepath.Join(tempDir, dir), 0755)
		require.NoError(t, err)
	}

	// Add some test files
	ruleFile := filepath.Join(tempDir, "rules", "test.md")
	err := os.WriteFile(ruleFile, []byte("# Test Rule\n- category: test\n- priority: high\n\nTest content"), 0644)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Run server
	err = runServer(ctx, tempDir)
	require.NoError(t, err)

	// Let it run for a bit to ensure all goroutines start
	time.Sleep(500 * time.Millisecond)

	// Cancel to trigger shutdown
	cancel()

	// Give time for cleanup
	time.Sleep(100 * time.Millisecond)
}

func TestRunServer_MultipleInstances(t *testing.T) {
	// Test running multiple instances in parallel
	var wg sync.WaitGroup

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			tempDir := t.TempDir()
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			err := runServer(ctx, tempDir)
			assert.NoError(t, err)

			// Let it run briefly
			time.Sleep(200 * time.Millisecond)
		}(i)
	}

	wg.Wait()
}
