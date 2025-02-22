package fetcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"time"
)

// Fetcher allows pulling from an upstream scaffold registry. This is hard coded to the lunarway/scaffold registry, it can also be provided by a path which in that case, will not do anything
type Fetcher struct{}

func NewFetcher() *Fetcher {
	return &Fetcher{}
}

const readWriteExec = 0o644

const githubProject = "kjuulh/scaffold"

var (
	scaffoldFolder = os.ExpandEnv("$HOME/.scaffold")
	scaffoldClone  = path.Join(scaffoldFolder, "upstream")
	scaffoldCache  = path.Join(scaffoldFolder, "scaffold.updates.json")
)

func (f *Fetcher) CloneRepository(ctx context.Context, registryPath *string, ui *slog.Logger) error {
	if err := os.MkdirAll(scaffoldFolder, readWriteExec); err != nil {
		return fmt.Errorf("failed to create scaffold folder: %w", err)
	}

	if *registryPath == "" {
		if _, err := os.Stat(scaffoldClone); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("failed to find the upstream folder: %w", err)
			}

			ui.Info("cloning upstream templates")
			if err := cloneUpstream(ctx); err != nil {
				return fmt.Errorf("failed to clone upstream registry: %w", err)
			}
		} else {
			now := time.Now()
			lastUpdatedUnix := getCacheUpdate(ui, ctx)
			lastUpdated := time.Unix(lastUpdatedUnix, 0)

			// Cache for 7 days
			if lastUpdated.Before(now.Add(-time.Hour * 24 * 7)) {
				ui.Info("update templates folder")
				if err := f.UpdateUpstream(ctx); err != nil {
					return fmt.Errorf("failed to update upstream scaffold folder: %w", err)
				}
			}
		}
	}

	return nil
}

func (f *Fetcher) UpdateUpstream(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "git", "pull", "--rebase")
	cmd.Dir = scaffoldClone

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("git pull failed with output: %s\n\n", string(output))
		return fmt.Errorf("git pull failed: %w", err)
	}

	if err := createCacheUpdate(ctx); err != nil {
		return err
	}

	return nil
}

func cloneUpstream(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "coffee", "repo", "clone", githubProject, scaffoldClone)

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("git clone failed with output: %s\n\n", string(output))
		return fmt.Errorf("git clone failed: %w", err)
	}

	if err := createCacheUpdate(ctx); err != nil {
		return err
	}

	return nil
}

type CacheUpdate struct {
	LastUpdated int64 `json:"lastUpdated"`
}

func createCacheUpdate(_ context.Context) error {
	content, err := json.Marshal(CacheUpdate{
		LastUpdated: time.Now().Unix(),
	})
	if err != nil {
		return fmt.Errorf("failed to prepare cache update: %w", err)
	}

	if err := os.WriteFile(scaffoldCache, content, readWriteExec); err != nil {
		return fmt.Errorf("failed to write cache update: %w", err)
	}

	return nil
}

func getCacheUpdate(ui *slog.Logger, _ context.Context) int64 {
	content, err := os.ReadFile(scaffoldCache)
	if err != nil {
		return 0
	}

	var cacheUpdate CacheUpdate
	if err := json.Unmarshal(content, &cacheUpdate); err != nil {
		ui.Warn("failed to read cache, it might be invalid", "error", err)
		return 0
	}

	return cacheUpdate.LastUpdated
}
