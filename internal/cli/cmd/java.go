package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"QMLauncher/internal/cli/output"
	env "QMLauncher/pkg"

	"github.com/alecthomas/kong"
	"github.com/jedib0t/go-pretty/v6/table"
)

// ListCmd lists all installed Java versions.
type JavaListCmd struct{}

func (c *JavaListCmd) Run(ctx *kong.Context) error {
	javas, err := listInstalledJavaVersions()
	if err != nil {
		return fmt.Errorf("list java versions: %w", err)
	}

	if len(javas) == 0 {
		output.Info("No Java versions installed")
		return nil
	}

	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{
		"#",
		output.Translate("java.table.name"),
		output.Translate("java.table.path"),
	})

	for i, java := range javas {
		t.AppendRow(table.Row{i, java.Name, java.Path})
	}
	t.Render()
	return nil
}

type JavaVersion struct {
	Name string
	Path string
}

func listInstalledJavaVersions() ([]JavaVersion, error) {
	entries, err := os.ReadDir(env.JavaDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var javas []JavaVersion
	for _, entry := range entries {
		if entry.IsDir() {
			name := entry.Name()
			path := filepath.Join(env.JavaDir, name)

			// Check if this directory contains a valid Java installation
			binDir := filepath.Join(path, "bin")
			if _, err := os.Stat(binDir); err == nil {
				javas = append(javas, JavaVersion{
					Name: name,
					Path: path,
				})
			}
		}
	}

	// Sort by name
	sort.Slice(javas, func(i, j int) bool {
		return strings.ToLower(javas[i].Name) < strings.ToLower(javas[j].Name)
	})

	return javas, nil
}

// JavaCmd manages Java runtime installations.
type JavaCmd struct {
	List JavaListCmd `cmd:"" help:"${java_list}"`
}
