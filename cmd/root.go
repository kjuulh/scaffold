package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/kjuulh/scaffold/internal/fetcher"
	"github.com/kjuulh/scaffold/internal/templates"
)

func Execute() error {
	var (
		registryPath     string
		forceCacheUpdate bool
	)

	rootCmd := &cobra.Command{
		Use:   "scaffold",
		Short: "pick a template, and scaffold a piece of code",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runScaffold(cmd.Context(), &registryPath, &forceCacheUpdate); err != nil {
				fmt.Printf("failed to run scaffold: %s\n", err.Error())
				os.Exit(1)
			}
		},
	}

	rootCmd.PersistentFlags().StringVar(&registryPath, "registry", "", "where to get the registry from, defaults to upstream repository")
	rootCmd.PersistentFlags().BoolVar(&forceCacheUpdate, "force-cache-update", false, "should we force an update of the cache?")
	_ = rootCmd.ParseFlags(os.Args)

	subCommands, err := getScaffoldCommands(&registryPath, &forceCacheUpdate)
	if err != nil {
		fmt.Printf("failed to setup subcommands: %s\n", err.Error())
		os.Exit(1)
	}
	if len(subCommands) > 0 {
		rootCmd.AddCommand(subCommands...)
	}

	return rootCmd.Execute()
}

func runScaffold(ctx context.Context, registryPath *string, forceCacheUpdate *bool) error {
	ui := slog.New(slog.NewTextHandler(os.Stderr, nil))
	fetcher := fetcher.NewFetcher(*forceCacheUpdate)
	templateIndexer := templates.NewTemplateIndexer()
	templateLoader := templates.NewTemplateLoader(ui)
	fileWriter := templates.NewFileWriter().WithPromptOverride(promptOverrideFile)

	if err := fetcher.CloneRepository(ctx, registryPath, ui); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	templates, err := templateIndexer.Index(ctx, *registryPath, ui)
	if err != nil {
		return fmt.Errorf("failed to index templates: %w", err)
	}

	template, err := chooseTemplate(templates)
	if err != nil {
		return fmt.Errorf("failed to choose a template: %w", err)
	}

	ui.Info("Loading template files", "name", template.File.Name)

	files, err := templateLoader.Load(ctx, template)
	if err != nil {
		return fmt.Errorf("failed to load template files: %w", err)
	}

	ui.Info("Loaded templates", "files", len(files))

	scaffoldDest, err := promptInput(template, files)
	if err != nil {
		return fmt.Errorf("failed to prompt input: %w", err)
	}

	ui.Info("Templating files")

	templatedFiles, err := templateLoader.TemplateFiles(template, files, scaffoldDest)
	if err != nil {
		return fmt.Errorf("failed to template files: %w", err)
	}

	ui.Info("Templated files", "files", len(templatedFiles))

	if err := fileWriter.Write(ctx, ui, templatedFiles); err != nil {
		return fmt.Errorf("failed to write files: %w", err)
	}

	return nil
}

func promptOverrideFile(file templates.TemplatedFile) (bool, error) {
	theme := huh.ThemeBase16()
	theme.FieldSeparator = lipgloss.NewStyle().SetString("\n")
	theme.Help.FullKey.MarginTop(1)

	confirm := false
	f := huh.
		NewForm(
			huh.
				NewGroup(
					huh.
						NewConfirm().
						Title(fmt.Sprintf("Should override existing file?: %s", file.DestinationPath)).
						Value(&confirm),
				),
		).
		WithTheme(theme)
	err := f.Run()
	if err != nil {
		return false, fmt.Errorf("failed to specify path for scaffold: %w", err)
	}

	return confirm, nil
}

func promptInput(template *templates.Template, files []templates.File) (string, error) {
	if len(template.File.Input) == 0 {
		return "", nil
	}

	theme := huh.ThemeBase16()
	theme.FieldSeparator = lipgloss.NewStyle().SetString("\n")
	theme.Help.FullKey.MarginTop(1)

	for input, inputSpec := range template.File.Input {
		inputVal := inputSpec.Default
		f := huh.
			NewForm(
				huh.
					NewGroup(
						huh.
							NewText().
							TitleFunc(
								func() string {
									return fmt.Sprintf("Template requires: %s", input)
								},
								&inputVal,
							).
							Value(&inputVal).
							Description(inputSpec.Description).
							WithHeight(1),
					),
			).
			WithTheme(theme)

		err := f.Run()
		if err != nil {
			return "", fmt.Errorf("failed to find template variable: %s, %w", input, err)
		}

		if inputSpec.Type == "int" {
			if _, err := strconv.Atoi(inputVal); err != nil {
				return "", fmt.Errorf("input: '%s' for variable: '%s' is not an int: \n%w", inputVal, input, err)
			}
		}

		template.Input[input] = inputVal
	}

	scaffoldDest, err := templates.TemplatePath(template)
	if err != nil {
		return "", fmt.Errorf("failed to template path: %w", err)
	}
	f := huh.
		NewForm(
			huh.
				NewGroup(
					huh.
						NewText().
						Title("Path: where to scaffold files: %s").
						Value(&scaffoldDest).
						DescriptionFunc(func() string {
							var sb strings.Builder

							_, err := sb.WriteString("Preview of file paths:\n")
							if err != nil {
								panic(err)
							}

							for _, file := range files {
								previewFilePath := path.Join(scaffoldDest, file.RelPath)

								if _, err := sb.WriteString(fmt.Sprintf("%s\n", previewFilePath)); err != nil {
									panic(err)
								}
							}

							return sb.String()
						}, &scaffoldDest),
				),
		).
		WithTheme(theme)
	err = f.Run()
	if err != nil {
		return "", fmt.Errorf("failed to specify path for scaffold: %w", err)
	}

	if scaffoldDest == "" {
		return "", errors.New("path cannot be an empty string")
	}

	return scaffoldDest, nil
}

func chooseTemplate(templates []templates.Template) (*templates.Template, error) {
	idx, err := fuzzyfinder.Find(
		templates,
		func(i int) string {
			return templates[i].File.Name
		},
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}

			template := templates[i]
			templateContent, err := yaml.Marshal(template.File)
			if err != nil {
				return fmt.Sprintf("failed to format template: %s", err.Error())
			}

			return fmt.Sprintf("Template:\n===\n%s\n===\n", string(templateContent))
		}))
	if err != nil {
		return nil, fmt.Errorf("failed to find a template: %w", err)
	}

	return &templates[idx], nil
}
