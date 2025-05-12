package tests

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kjuulh/scaffold/internal/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ScaffoldFixture provides an api on top of the scaffold templater, this is opposed to calling the cli
type ScaffoldFixture struct {
	vars map[string]string
	path string
}

func (s *ScaffoldFixture) WithVariable(key, val string) *ScaffoldFixture {
	s.vars[key] = val

	return s
}

func (s *ScaffoldFixture) WithPath(path string) *ScaffoldFixture {
	s.path = path

	return s
}

// TestFixture is an opinionated way for the templates to be able to test their code, this also works as an accepttest for the scaffolder itself.
type TestFixture struct {
	pkg string
	t   *testing.T
}

func Test(t *testing.T, pkg string) *TestFixture {
	return &TestFixture{
		pkg: pkg,
		t:   t,
	}
}

// ScaffoldDefaultTest tests that the code can be run with the default variables. We want to have sane defaults for most things. As such there is an opinionated way of running these tests
func (f *TestFixture) ScaffoldDefaultTest(testName string) *TestFixture {
	f.ScaffoldTest(testName, func(fixture *ScaffoldFixture) {})

	return f
}

// ScaffoldTest is a large fixture, which allows running the accepttest over a template, in turn creating files and comparing them. either way they have to match, if the templater generated more files we expect, it fails, if they're different the test fails
func (f *TestFixture) ScaffoldTest(testName string, input func(fixture *ScaffoldFixture)) *TestFixture {
	f.t.Run(
		f.pkg,
		func(t *testing.T) {
			t.Parallel()

			testName := strings.ToLower(strings.ReplaceAll(testName, " ", "_"))

			fixture := &ScaffoldFixture{
				vars: make(map[string]string),
			}
			input(fixture)

			ctx := t.Context()

			indexer := templates.NewTemplateIndexer()
			ui := slog.New(slog.NewTextHandler(os.Stderr, nil))
			loader := templates.NewTemplateLoader(ui)
			writer := templates.NewFileWriter()

			templateFiles, err := indexer.Index(ctx, "../", ui)
			require.NoError(t, err)

			template, err := find(templateFiles, f.pkg)
			require.NoError(t, err, "failed to find a template")

			files, err := loader.Load(ctx, template)
			require.NoError(t, err, "failed to load template files")

			for input, inputSpec := range template.File.Input {
				template.Input[input] = inputSpec.Default
			}

			for key, val := range fixture.vars {
				template.Input[key] = val
			}

			templatePath, err := templates.TemplatePath(template)
			require.NoError(t, err)
			if fixture.path != "" {
				templatePath = fixture.path
			}

			actualPath := path.Join("testdata", testName, "actual")
			expectedPath := path.Join("testdata", testName, "expected")

			templatedFiles, err := loader.TemplateFiles(template, files, path.Join(actualPath, templatePath))
			require.NoError(t, err, "failed to template files")

			err = os.RemoveAll(actualPath)
			require.NoError(t, err)

			err = writer.Write(ctx, ui, templatedFiles)
			require.NoError(t, err, "failed to write files")

			actualFiles, err := getFiles(actualPath)
			require.NoError(t, err, "failed to get actual files")

			expectedFiles, err := getFiles(expectedPath)
			assert.NoError(t, err, "failed to get expected files")

			slices.Sort(actualFiles)
			slices.Sort(expectedFiles)

			assert.Equal(
				t,
				makeRelative(expectedPath, expectedFiles),
				makeRelative(actualPath, actualFiles),
				"expected and actual files didn't match",
			)

			compareFiles(t,
				expectedPath, actualPath,
				expectedFiles, actualFiles,
			)
		})

	return f
}

func compareFiles(t *testing.T, expectedPath, actualPath string, expectedFiles, actualFiles []string) {
	expectedRelativeFiles := makeRelative(expectedPath, expectedFiles)
	actualRelativeFiles := makeRelative(actualPath, actualFiles)

	for expectedIndex, expectedRelativeFile := range expectedRelativeFiles {
		for actualIndex, actualRelativeFile := range actualRelativeFiles {
			if expectedRelativeFile == actualRelativeFile {
				expectedFilePath := expectedFiles[expectedIndex]
				actualFilePath := actualFiles[actualIndex]

				expectedFile, err := os.ReadFile(expectedFilePath)
				require.NoError(t, err, "failed to read expected file")

				actualFile, err := os.ReadFile(actualFilePath)
				require.NoError(t, err, "failed to read actual file")

				assert.Equal(t,
					string(expectedFile), string(actualFile),
					"expected and actual file doesn't match\n\texpected path=%s\n\t  actual path=%s",
					expectedFilePath, actualFilePath,
				)
			}
		}
	}
}

// makeRelative, test files are prefixed with either actual or expected, this makes it hard to compare, this makes them comparable by removing their unique folder prefix.
func makeRelative(prefix string, filePaths []string) []string {
	output := make([]string, 0, len(filePaths))
	for _, filePath := range filePaths {
		relative := strings.TrimPrefix(strings.TrimPrefix(filePath, prefix), "/")

		output = append(output, relative)
	}
	return output
}

func find(templates []templates.Template, templateName string) (*templates.Template, error) {
	templateNames := make([]string, 0)
	for _, template := range templates {
		if template.File.Name == templateName {
			return &template, nil
		}

		templateNames = append(templateNames, template.File.Name)
	}

	return nil, fmt.Errorf("template was not found: %s", strings.Join(templateNames, ", "))
}

func getFiles(root string) ([]string, error) {
	actualFiles := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.Type().IsRegular() {
			actualFiles = append(actualFiles, path)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return actualFiles, nil
}
