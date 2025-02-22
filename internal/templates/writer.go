package templates

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"sync"

	"golang.org/x/sync/errgroup"
)

const readWriteExec = 0o755
const readExec = 0o644

type PromptOverride func(file TemplatedFile) (bool, error)

// FileWriter writes the actual files to disk, it optionally takes a promptOverride which allows the caller to stop a potential override of a file
type FileWriter struct {
	promptOverride PromptOverride
}

func NewFileWriter() *FileWriter {
	return &FileWriter{
		promptOverride: nil,
	}
}

func (f *FileWriter) WithPromptOverride(po PromptOverride) *FileWriter {
	f.promptOverride = po

	return f
}

func (f *FileWriter) Write(ctx context.Context, ui *slog.Logger, templatedFiles []TemplatedFile) error {
	var fileExistsLock sync.Mutex

	egrp, _ := errgroup.WithContext(ctx)
	for _, file := range templatedFiles {
		egrp.Go(func() error {
			if _, err := os.Stat(file.DestinationPath); err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("failed to check if file exists: %s, %w", file.DestinationPath, err)
				}
			} else {
				if f.promptOverride != nil {
					fileExistsLock.Lock()
					defer fileExistsLock.Unlock()

					override, err := f.promptOverride(file)
					if err != nil {
						return fmt.Errorf("failed to get answer to whether a file should be overwritten or not: %w", err)
					}

					if !override {
						ui.Warn("Skipping file", "file", file.DestinationPath)
						return nil
					}
				}
			}

			if parent := path.Dir(file.DestinationPath); parent != "" && parent != "/" {
				if err := os.MkdirAll(parent, readWriteExec); err != nil {
					return fmt.Errorf("failed to create parent dir for: %s, %w", file.DestinationPath, err)
				}
			}

			ui.Info("writing file", "path", file.DestinationPath)
			if err := os.WriteFile(file.DestinationPath, file.Content, readExec); err != nil {
				return fmt.Errorf("failed to write file: %s, %w", file.DestinationPath, err)
			}

			return nil
		})
	}

	if err := egrp.Wait(); err != nil {
		return err
	}

	return nil
}
