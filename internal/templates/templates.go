package templates

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"sync"

	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

// TemplateIndexer loads all template specifications from the registry, allowing the caller to choose on of them, or simply display their properties.
type TemplateIndexer struct{}

func NewTemplateIndexer() *TemplateIndexer {
	return &TemplateIndexer{}
}

type Template struct {
	File TemplateFile
	Path string

	Input map[string]string
}

type TemplateDefault struct {
	Path string `yaml:"path"`
}
type TemplateInputs map[string]TemplateInput

type TemplateInput struct {
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
	Default     string `yaml:"default"`
}

type TemplateFileConfig struct {
	Rename string `yaml:"rename"`
}

type TemplateFile struct {
	Name    string          `yaml:"name"`
	Default TemplateDefault `yaml:"default"`
	Input   TemplateInputs  `yaml:"input"`
	Files   map[string]TemplateFileConfig
}

func (t *TemplateIndexer) Index(ctx context.Context, scaffoldRegistryFolder string, ui *slog.Logger) ([]Template, error) {
	ui.Debug("Loading templates...")

	templateDirEntries, err := os.ReadDir(scaffoldRegistryFolder)
	if err != nil {
		return nil, fmt.Errorf("failed to read templates dir: %s, %w", scaffoldRegistryFolder, err)
	}

	var (
		templates     = make([]Template, 0)
		templatesLock sync.Mutex
	)
	egrp, _ := errgroup.WithContext(ctx)
	for _, templateDirEntry := range templateDirEntries {
		egrp.Go(func() error {
			templatePath := path.Join(scaffoldRegistryFolder, templateDirEntry.Name())

			content, err := os.ReadFile(path.Join(templatePath, "scaffold.yaml"))
			if err != nil {
				return fmt.Errorf("failed to read: %s, %w", templateDirEntry.Name(), err)
			}

			var template TemplateFile
			if err := yaml.Unmarshal(content, &template); err != nil {
				return fmt.Errorf("failed to unmarshal template: %s, %w", string(content), err)
			}

			templatesLock.Lock()
			defer templatesLock.Unlock()
			templates = append(templates, Template{
				File:  template,
				Path:  templatePath,
				Input: make(map[string]string),
			})

			return nil
		})
	}

	if err := egrp.Wait(); err != nil {
		return nil, err
	}

	ui.Debug("Done loading templates...", "amount", len(templates))

	return templates, nil
}
