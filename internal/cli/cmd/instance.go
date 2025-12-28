package cmd

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"QMLauncher/internal/cli/output"
	"QMLauncher/pkg/launcher"

	"github.com/alecthomas/kong"
	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
)

// CreateCmd creates a new instance with specified parameters.
type CreateCmd struct {
	ID            string `arg:"" help:"${create_arg_id}"`
	Loader        string `help:"${create_arg_loader}" enum:"fabric,quilt,neoforge,forge,vanilla" default:"vanilla" short:"l"`
	Version       string `help:"${create_arg_version}" default:"release" short:"v"`
	LoaderVersion string `help:"${create_arg_loaderversion}" default:"latest"`
}

func (c *CreateCmd) Run(ctx *kong.Context) error {
	var loader launcher.Loader
	switch c.Loader {
	case "fabric":
		loader = launcher.LoaderFabric
	case "quilt":
		loader = launcher.LoaderQuilt
	case "vanilla":
		loader = launcher.LoaderVanilla
	case "neoforge":
		loader = launcher.LoaderNeoForge
	case "forge":
		loader = launcher.LoaderForge
	}
	inst, err := launcher.CreateInstance(launcher.InstanceOptions{
		GameVersion:   c.Version,
		Name:          c.ID,
		Loader:        loader,
		LoaderVersion: c.LoaderVersion,
		Config:        defaultInstanceConfig,
	})
	if err != nil {
		return fmt.Errorf("create instance: %w", err)
	}

	l := inst.LoaderVersion
	if l != "" {
		l = " " + l
	}
	output.Success(output.Translate("create.complete"), color.New(color.Bold).Sprint(inst.Name), inst.GameVersion, inst.Loader, l)
	output.Tip(output.Translate("tip.configure"))
	return nil
}

// DeleteCmd removes the specified instance.
type DeleteCmd struct {
	ID  string `arg:"" name:"id" help:"${delete_arg_id}"`
	Yes bool   `name:"yes" short:"y" help:"${delete_arg_yes}"`
}

func (c *DeleteCmd) Run(ctx *kong.Context) error {
	inst, err := launcher.FetchInstance(c.ID)
	if err != nil {
		return err
	}
	delete := c.Yes
	if !delete {
		var input string

		output.Warning(output.Translate("delete.confirm"))
		fmt.Printf(output.Translate("delete.warning"), color.New(color.Bold).Sprint(inst.Name))
		fmt.Scanln(&input)
		delete = input == "y" || input == "Y"
	}
	if delete {
		if err := launcher.RemoveInstance(c.ID); err != nil {
			return fmt.Errorf("remove instance: %w", err)
		}
		output.Success(output.Translate("delete.complete"), color.New(color.Bold).Sprint(inst.Name))
	} else {
		output.Info(output.Translate("delete.abort"))
	}
	return nil
}

// RenameCmd renames the specified instance.
type RenameCmd struct {
	ID  string `arg:"" help:"${rename_arg_id}"`
	New string `arg:"" help:"${rename_arg_new}"`
}

func (c *RenameCmd) Run(ctx *kong.Context) error {
	inst, err := launcher.FetchInstance(c.ID)
	if err != nil {
		return err
	}
	if err := inst.Rename(c.New); err != nil {
		return fmt.Errorf("rename instance: %w", err)
	}
	output.Success(output.Translate("rename.complete"))
	return nil
}

// ListCmd lists all installed instances.
type ListCmd struct{}

func (c *ListCmd) Run(ctx *kong.Context) error {
	var rows []table.Row
	instances, err := launcher.FetchAllInstances()
	if err != nil {
		return fmt.Errorf("fetch all instances: %w", err)
	}
	for i, inst := range instances {
		rows = append(rows, table.Row{i, inst.Name, inst.GameVersion, inst.Loader, inst.Dir()})
	}

	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{
		"#",
		output.Translate("search.table.name"),
		output.Translate("search.table.version"),
		output.Translate("search.table.type"),
		output.Translate("instance.table.path"),
	})
	t.AppendRows(rows)
	t.Render()
	return nil
}

// InstanceCmd enables management of Minecraft instances.
type InstanceCmd struct {
	Create      CreateCmd      `cmd:"" help:"${create}"`
	Delete      DeleteCmd      `cmd:"" help:"${delete}"`
	Rename      RenameCmd      `cmd:"" help:"${rename}"`
	List        ListCmd        `cmd:"" help:"${list}"`
	Export      ExportCmd      `cmd:"" help:"${export}"`
	Import      ImportCmd      `cmd:"" help:"${import}"`
	ListExports ListExportsCmd `cmd:"" help:"${list_exports}"`
}

// ExportManifest represents metadata for exported instance
type ExportManifest struct {
	Name          string `json:"name"`
	GameVersion   string `json:"game_version"`
	Loader        string `json:"loader"`
	LoaderVersion string `json:"loader_version"`
	ExportedAt    string `json:"exported_at"`
	Version       string `json:"version"`
}

// ExportCmd exports an instance to a ZIP archive
type ExportCmd struct {
	ID     string `arg:"" help:"${export_arg_id}"`
	Output string `help:"${export_arg_output}" short:"o"`
	Update bool   `help:"${export_arg_update}" short:"u"`
}

func (c *ExportCmd) Run(ctx *kong.Context) error {
	inst, err := launcher.FetchInstance(c.ID)
	if err != nil {
		return fmt.Errorf("fetch instance: %w", err)
	}

	outputPath := c.Output
	if outputPath == "" {
		if c.Update {
			// Try to find existing export file
			if existingPath := findExistingExport(c.ID); existingPath != "" {
				outputPath = existingPath
			} else {
				outputPath = fmt.Sprintf("%s.zip", c.ID)
			}
		} else {
			outputPath = fmt.Sprintf("%s.zip", c.ID)
		}
	}

	if err := exportInstance(inst, outputPath); err != nil {
		return fmt.Errorf("export instance: %w", err)
	}

	action := "exported"
	if c.Update {
		action = "updated"
	}
	output.Success(fmt.Sprintf("Instance '%s' %s to %s", c.ID, action, outputPath))
	return nil
}

// ImportCmd imports an instance from a ZIP archive
type ImportCmd struct {
	Path  string `arg:"" help:"${import_arg_path}"`
	Name  string `help:"${import_arg_name}" short:"n"`
	Force bool   `help:"${import_arg_force}" short:"f"`
}

func (c *ImportCmd) Run(ctx *kong.Context) error {
	importName := c.Name
	if importName == "" {
		// Extract name from archive filename
		baseName := filepath.Base(c.Path)
		importName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
	}

	if !c.Force && launcher.DoesInstanceExist(importName) {
		return fmt.Errorf("instance '%s' already exists (use --force to overwrite)", importName)
	}

	inst, err := importInstance(c.Path, importName, c.Force)
	if err != nil {
		return fmt.Errorf("import instance: %w", err)
	}

	output.Success(fmt.Sprintf("Instance '%s' imported successfully", inst.Name))
	return nil
}

// ListExportsCmd lists all exported instance archives
type ListExportsCmd struct {
	Path string `help:"${list_exports_arg_path}" short:"p"`
}

func (c *ListExportsCmd) Run(ctx *kong.Context) error {
	searchPath := c.Path
	if searchPath == "" {
		searchPath = "."
	}

	// Find all .zip files
	zipFiles, err := findZipFiles(searchPath)
	if err != nil {
		return fmt.Errorf("find ZIP files: %w", err)
	}

	if len(zipFiles) == 0 {
		output.Info("No exported instances found in " + searchPath)
		return nil
	}

	// Display table header
	fmt.Printf("%-20s %-15s %-12s %-15s %s\n", "NAME", "VERSION", "LOADER", "EXPORTED", "PATH")
	fmt.Println(strings.Repeat("-", 80))

	for _, zipFile := range zipFiles {
		info, err := readExportManifest(zipFile)
		if err != nil {
			// Skip files that don't have valid manifest
			continue
		}

		// Parse exported time from string to int64
		exportedAt, err := strconv.ParseInt(info.ExportedAt, 10, 64)
		if err != nil {
			exportedAt = 0 // Use epoch if parsing fails
		}
		exportedTime := time.Unix(exportedAt, 0).Format("2006-01-02 15:04")

		fmt.Printf("%-20s %-15s %-12s %-15s %s\n",
			info.Name, info.GameVersion, info.Loader, exportedTime, zipFile)
	}

	return nil
}

var defaultInstanceConfig = func() launcher.InstanceConfig {
	config := launcher.InstanceConfig{
		WindowResolution: struct {
			Width  int `toml:"width" json:"width"`
			Height int `toml:"height" json:"height"`
		}{
			Width:  1708,
			Height: 960,
		},
		MinMemory: 512,
		MaxMemory: 4096,
	}

	// Note: Java path is left empty by default
	// This will cause QMLauncher to download and use Mojang Java runtime
	// which is specifically designed for Minecraft and ensures compatibility
	// System Java can still be manually specified in instance.toml if needed

	return config
}()

// exportInstance creates a ZIP archive of the instance
func exportInstance(inst launcher.Instance, outputPath string) error {
	// Create ZIP file
	zipFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create archive: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	instanceDir := inst.Dir()

	// Create manifest
	manifest := ExportManifest{
		Name:          inst.Name,
		GameVersion:   inst.GameVersion,
		Loader:        string(inst.Loader),
		LoaderVersion: inst.LoaderVersion,
		ExportedAt:    fmt.Sprintf("%d", time.Now().Unix()),
		Version:       "1.0",
	}

	// Add manifest to archive
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	manifestWriter, err := zipWriter.Create("manifest.json")
	if err != nil {
		return fmt.Errorf("create manifest in archive: %w", err)
	}
	if _, err := manifestWriter.Write(manifestData); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	// Define files and directories to export
	// Note: Some files and directories may not exist and will be skipped automatically
	exportItems := []string{
		"instance.toml",   // QMLauncher configuration (required)
		"options.txt",     // Minecraft settings (optional)
		"servers.dat",     // Server list (optional)
		"config/",         // Mod configurations (optional)
		"defaultconfigs/", // Default configurations (optional)
		"mods/",           // Installed mods (optional)
		"resourcepacks/",  // Resource packs (optional)
		"shaderpacks/",    // Shader packs (optional)
	}

	// Export specified files and directories
	for _, item := range exportItems {
		itemPath := filepath.Join(instanceDir, item)

		// Check if item exists
		info, err := os.Stat(itemPath)
		if os.IsNotExist(err) {
			// Skip if doesn't exist
			continue
		}
		if err != nil {
			return fmt.Errorf("stat %s: %w", item, err)
		}

		if info.IsDir() {
			// Export directory recursively
			err = filepath.Walk(itemPath, func(path string, fileInfo os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if fileInfo.IsDir() {
					return nil
				}

				relPath, err := filepath.Rel(instanceDir, path)
				if err != nil {
					return fmt.Errorf("get relative path: %w", err)
				}

				return addFileToZip(zipWriter, instanceDir, relPath)
			})
			if err != nil {
				return fmt.Errorf("export directory %s: %w", item, err)
			}
		} else {
			// Export single file
			err = addFileToZip(zipWriter, instanceDir, item)
			if err != nil {
				return fmt.Errorf("export file %s: %w", item, err)
			}
		}
	}

	return nil
}

// addFileToZip adds a single file to the ZIP archive
func addFileToZip(zipWriter *zip.Writer, baseDir, relPath string) error {
	fullPath := filepath.Join(baseDir, relPath)

	file, err := os.Open(fullPath)
	if err != nil {
		return fmt.Errorf("open file %s: %w", relPath, err)
	}
	defer file.Close()

	zipFileWriter, err := zipWriter.Create(relPath)
	if err != nil {
		return fmt.Errorf("create file in archive %s: %w", relPath, err)
	}

	if _, err := io.Copy(zipFileWriter, file); err != nil {
		return fmt.Errorf("copy file to archive %s: %w", relPath, err)
	}

	return nil
}

// importInstance extracts an instance from a ZIP archive
func importInstance(archivePath, name string, force bool) (launcher.Instance, error) {
	// Open ZIP archive
	zipReader, err := zip.OpenReader(archivePath)
	if err != nil {
		return launcher.Instance{}, fmt.Errorf("open archive: %w", err)
	}
	defer zipReader.Close()

	// Read manifest
	var manifest ExportManifest
	var manifestFound bool
	for _, file := range zipReader.File {
		if file.Name == "manifest.json" {
			manifestFound = true
			rc, err := file.Open()
			if err != nil {
				return launcher.Instance{}, fmt.Errorf("open manifest: %w", err)
			}
			defer rc.Close()

			data, err := io.ReadAll(rc)
			if err != nil {
				return launcher.Instance{}, fmt.Errorf("read manifest: %w", err)
			}

			if err := json.Unmarshal(data, &manifest); err != nil {
				return launcher.Instance{}, fmt.Errorf("parse manifest: %w", err)
			}
			break
		}
	}

	if !manifestFound {
		return launcher.Instance{}, fmt.Errorf("manifest.json not found in archive")
	}

	// Use provided name or name from manifest
	instanceName := name
	if instanceName == "" {
		instanceName = manifest.Name
	}

	// If force is false and instance exists, return error
	if !force && launcher.DoesInstanceExist(instanceName) {
		return launcher.Instance{}, fmt.Errorf("instance '%s' already exists", instanceName)
	}

	// If force is true and instance exists, remove it first
	if force && launcher.DoesInstanceExist(instanceName) {
		if err := launcher.RemoveInstance(instanceName); err != nil {
			return launcher.Instance{}, fmt.Errorf("remove existing instance: %w", err)
		}
	}

	// Create new instance
	var loader launcher.Loader
	switch manifest.Loader {
	case "fabric":
		loader = launcher.LoaderFabric
	case "quilt":
		loader = launcher.LoaderQuilt
	case "vanilla":
		loader = launcher.LoaderVanilla
	case "neoforge":
		loader = launcher.LoaderNeoForge
	case "forge":
		loader = launcher.LoaderForge
	default:
		return launcher.Instance{}, fmt.Errorf("unknown loader: %s", manifest.Loader)
	}

	inst, err := launcher.CreateInstance(launcher.InstanceOptions{
		Name:          instanceName,
		GameVersion:   manifest.GameVersion,
		Loader:        loader,
		LoaderVersion: manifest.LoaderVersion,
		Config:        defaultInstanceConfig,
	})
	if err != nil {
		return launcher.Instance{}, fmt.Errorf("create instance: %w", err)
	}

	// Extract files from archive (skip manifest.json)
	instanceDir := inst.Dir()

	for _, file := range zipReader.File {
		if file.Name == "manifest.json" {
			continue
		}

		// Create destination path
		destPath := filepath.Join(instanceDir, file.Name)

		// Create directory if needed
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return launcher.Instance{}, fmt.Errorf("create directory for %s: %w", file.Name, err)
		}

		// Extract file
		rc, err := file.Open()
		if err != nil {
			return launcher.Instance{}, fmt.Errorf("open file in archive %s: %w", file.Name, err)
		}
		defer rc.Close()

		destFile, err := os.Create(destPath)
		if err != nil {
			return launcher.Instance{}, fmt.Errorf("create destination file %s: %w", file.Name, err)
		}
		defer destFile.Close()

		if _, err := io.Copy(destFile, rc); err != nil {
			return launcher.Instance{}, fmt.Errorf("extract file %s: %w", file.Name, err)
		}
	}

	return inst, nil
}

// findExistingExport looks for existing export file for the given instance
func findExistingExport(instanceName string) string {
	// Check current directory for instanceName.zip
	expectedPath := fmt.Sprintf("%s.zip", instanceName)
	if _, err := os.Stat(expectedPath); err == nil {
		return expectedPath
	}
	return ""
}

// findZipFiles finds all ZIP files in the given directory
func findZipFiles(dir string) ([]string, error) {
	var zipFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".zip") {
			zipFiles = append(zipFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort by modification time (newest first)
	sort.Slice(zipFiles, func(i, j int) bool {
		infoI, _ := os.Stat(zipFiles[i])
		infoJ, _ := os.Stat(zipFiles[j])
		return infoI.ModTime().After(infoJ.ModTime())
	})

	return zipFiles, nil
}

// readExportManifest reads manifest.json from a ZIP file
func readExportManifest(zipPath string) (ExportManifest, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return ExportManifest{}, err
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.Name == "manifest.json" {
			rc, err := file.Open()
			if err != nil {
				return ExportManifest{}, err
			}
			defer rc.Close()

			var manifest ExportManifest
			if err := json.NewDecoder(rc).Decode(&manifest); err != nil {
				return ExportManifest{}, err
			}

			return manifest, nil
		}
	}

	return ExportManifest{}, fmt.Errorf("manifest.json not found")
}
