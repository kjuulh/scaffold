package templates

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	gotmpl "text/template"

	"github.com/iancoleman/strcase"
	"golang.org/x/sync/errgroup"
)

// TemplateLoader reads a templates files and runs their respective templating on them.
type TemplateLoader struct {
	logger *slog.Logger
}

func NewTemplateLoader(logger *slog.Logger) *TemplateLoader {
	return &TemplateLoader{logger}
}

type File struct {
	content []byte
	path    string
	RelPath string
}

var funcs = gotmpl.FuncMap{
	"ReplaceAll":   strings.ReplaceAll,
	"ToLower":      strings.ToLower,
	"ToUpper":      strings.ToUpper,
	"ToPascalCase": strcase.ToCamel,
	"ToCamelCase":  strcase.ToLowerCamel,
	"ToSnakeCase":  strcase.ToSnake,
	"ToCompressedCase": func(i string) string {
		return strings.ReplaceAll(strcase.ToSnake(i), "_", "")
	},
}

// TemplatePath formats the template file path using go templates, this is useful for programmatically changing the output string using go tmpls
func TemplatePath(template *Template) (string, error) {
	tmpl, err := gotmpl.New("path").Funcs(funcs).Parse(template.File.Default.Path)
	if err != nil {
		return "", err
	}

	output := bytes.NewBufferString("")
	if err := tmpl.Execute(output, template); err != nil {
		return "", err
	}

	templatePath := strings.TrimSpace(output.String())

	return templatePath, nil
}

// Load loads the template files from disk
func (t *TemplateLoader) Load(ctx context.Context, template *Template) ([]File, error) {
	templateFilePath := path.Join(template.Path, "files")
	if _, err := os.Stat(templateFilePath); err != nil {
		return nil, fmt.Errorf("failed to lookup template files %s, %w", templateFilePath, err)
	}

	filePaths := make([]string, 0)
	err := filepath.WalkDir(templateFilePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Is a file
		if d.Type().IsRegular() {
			filePaths = append(filePaths, path)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read template files: %w", err)
	}

	var (
		filesLock sync.Mutex
		files     = make([]File, 0)
	)
	egrp, _ := errgroup.WithContext(ctx)
	for _, filePath := range filePaths {
		egrp.Go(func() error {
			fileContent, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read file: %s, %w", filePath, err)
			}

			filesLock.Lock()
			defer filesLock.Unlock()

			files = append(files, File{
				content: fileContent,
				path:    filePath,
				RelPath: strings.TrimPrefix(strings.TrimPrefix(filePath, templateFilePath), "/"),
			})

			return nil
		})

	}
	if err := egrp.Wait(); err != nil {
		return nil, err
	}

	return files, nil
}

type TemplatedFile struct {
	Content         []byte
	DestinationPath string
}

// TemplateFiles runs the actual templating on the files, and tells it where to go. The writes doesn't happen here yet.
func (l *TemplateLoader) TemplateFiles(template *Template, files []File, scaffoldDest string) ([]TemplatedFile, error) {
	templatedFiles := make([]TemplatedFile, 0)
	for _, file := range files {
		tmpl, err := gotmpl.
			New(file.RelPath).
			Funcs(funcs).
			Parse(string(file.content))
		if err != nil {
			return nil, fmt.Errorf("failed to parse template file: %s, %w", file.RelPath, err)
		}

		output := bytes.NewBufferString("")
		if err := tmpl.Execute(output, template); err != nil {
			return nil, fmt.Errorf("failed to write template file: %s, %w", file.RelPath, err)
		}

		fileDir := path.Dir(file.RelPath)
		fileName := strings.TrimSuffix(path.Base(file.RelPath), ".gotmpl")
		filePath := path.Join(fileDir, fileName)

		//slog.Info("file renames", "renames", template.File.Files)

		if fileConfig, ok := template.File.Files[strings.TrimSuffix(file.RelPath, ".gotmpl")]; ok && fileConfig.Rename != "" {
			l.logger.Debug("templating file", "path", file.RelPath, "rename", fileConfig.Rename)

			renameTmpl, err := gotmpl.New(file.RelPath).Funcs(funcs).Parse(fileConfig.Rename)
			if err != nil {
				return nil, fmt.Errorf("failed to parse rename for: %s in scaffold.yaml: %w", file.RelPath, err)
			}

			type RenameContext struct {
				Template
				OriginalFileName string
				OriginalFilePath string
			}

			output := bytes.NewBufferString("")
			if err := renameTmpl.Execute(output, RenameContext{
				Template:         *template,
				OriginalFileName: fileName,
				OriginalFilePath: strings.TrimSuffix(file.RelPath, ".gotmpl"),
			}); err != nil {
				return nil, fmt.Errorf("failed to template rename: %s, %w", file.RelPath, err)
			}

			filePath = strings.TrimSpace(output.String())
		} else {
			l.logger.Debug("using raw file path", "path", file.RelPath)
		}

		templatedFiles = append(templatedFiles, TemplatedFile{
			Content:         output.Bytes(),
			DestinationPath: path.Join(scaffoldDest, filePath),
		})
	}

	return templatedFiles, nil
}
