package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/golang-cz/devslog"
	"github.com/kjuulh/scaffold/internal/fetcher"
	"github.com/kjuulh/scaffold/internal/templates"
	"github.com/spf13/cobra"
)

func getScaffoldCommands(registryPath *string, forceCacheUpdate *bool) ([]*cobra.Command, error) {
	var (
		ctx             = context.Background()
		ui              = slog.New(devslog.NewHandler(os.Stderr, &devslog.Options{HandlerOptions: &slog.HandlerOptions{Level: slog.LevelInfo}}))
		fetcher         = fetcher.NewFetcher(*forceCacheUpdate)
		templateIndexer = templates.NewTemplateIndexer()
		templateLoader  = templates.NewTemplateLoader(ui)
		fileWriter      = templates.NewFileWriter().WithPromptOverride(promptOverrideFile)
	)

	// if err := fetcher.CloneRepository(ctx, registryPath, ui); err != nil {
	// 	return nil, fmt.Errorf("failed to clone repository: %w", err)
	// }

	localRegistryPath, available := fetcher.Available(registryPath)
	if !available {
		return nil, nil
	}

	templateFiles, err := templateIndexer.Index(ctx, localRegistryPath, ui)
	if err != nil {
		return nil, fmt.Errorf("failed to index templates: %w", err)
	}

	commands := make([]*cobra.Command, 0)
	for _, template := range templateFiles {
		var templatePath string
		variables := make([]*LazyVariable, 0)

		for name, variable := range template.File.Input {
			variables = append(variables, &LazyVariable{
				Name:        name,
				Description: variable.Description,
				Value:       variable.Default,
			})
		}

		cmd := &cobra.Command{
			Use:          template.File.Name,
			SilenceUsage: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				ui.Info("Loading template files", "name", template.File.Name)

				for _, variable := range variables {
					ui.Info("found value", "key", variable.Name, "value", variable.Value)
					template.Input[variable.Name] = variable.Value
				}

				if templatePath == "" {
					scaffoldDest, err := templates.TemplatePath(&template)
					if err != nil {
						return err
					}

					templatePath = scaffoldDest
				}

				files, err := templateLoader.Load(ctx, &template)
				if err != nil {
					return fmt.Errorf("failed to load template files: %w", err)
				}

				templatedFiles, err := templateLoader.TemplateFiles(&template, files, templatePath)
				if err != nil {
					return fmt.Errorf("failed to template files: %w", err)
				}

				ui.Info("Templated files", "files", len(templatedFiles))

				if err := fileWriter.Write(ctx, ui, templatedFiles); err != nil {
					return fmt.Errorf("failed to write files: %w", err)
				}

				return nil
			},
		}

		cmd.Flags().StringVar(&templatePath, "path", "", "which path to put the output files")

		for _, variable := range variables {
			cmd.Flags().StringVar(&variable.Value, variable.Name, variable.Value, variable.Description)
		}

		commands = append(commands, cmd)
	}

	return commands, nil
}

type LazyVariable struct {
	Name        string
	Description string
	Value       string
}
