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
	"QMLauncher/internal/meta"
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

		// Also remove the instance directory
		if err := os.RemoveAll(filepath.Dir(inst.Dir())); err != nil {
			output.Warning("Не удалось удалить папку инстанса: %v", err)
			// Don't return error, instance was successfully removed from list
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
		// Format QMServer Cloud status
		qmStatus := "Нет"
		if inst.Config.IsUsingQMServerCloud {
			qmStatus = "Да"
		}

		// Format Premium status
		premiumStatus := "Нет"
		if inst.Config.IsPremium {
			premiumStatus = "Да"
		}

		rows = append(rows, table.Row{i, inst.Name, inst.GameVersion, inst.Loader, inst.Dir(), qmStatus, premiumStatus})
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
		"QMServer Cloud",
		"Premium",
	})
	t.AppendRows(rows)
	t.Render()
	return nil
}

// InstanceCmd enables management of Minecraft instances.
type InstanceCmd struct {
	Create        CreateCmd        `cmd:"" help:"${create}"`
	Delete        DeleteCmd        `cmd:"" help:"${delete}"`
	Rename        RenameCmd        `cmd:"" help:"${rename}"`
	Start         StartCmd         `cmd:"" help:"${start}"`
	List          ListCmd          `cmd:"" help:"${list}"`
	Export        ExportCmd        `cmd:"" help:"${export}"`
	Import        ImportCmd        `cmd:"" help:"${import}"`
	ListExports   ListExportsCmd   `cmd:"" help:"${list_exports}"`
	Mods          ModsCmd          `cmd:"" help:"${mods}"`
	ResourcePacks ResourcePacksCmd `cmd:"" help:"${resourcepacks}"`
	Shaders       ShadersCmd       `cmd:"" help:"${shaders}"`
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
	Merge bool   `help:"${import_arg_merge}" short:"m"`
}

func (c *ImportCmd) Run(ctx *kong.Context) error {
	importName := c.Name
	if importName == "" {
		// Extract name from archive filename
		baseName := filepath.Base(c.Path)
		importName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
	}

	inst, err := importInstance(c.Path, importName, c.Force, c.Merge)
	if err != nil {
		return fmt.Errorf("import instance: %w", err)
	}

	action := "imported"
	if c.Merge {
		action = "updated"
	} else if c.Force {
		action = "overwritten"
	}

	output.Success(fmt.Sprintf("Instance '%s' %s successfully", inst.Name, action))
	return nil
}

// ListExportsCmd lists all exported instance archives
type ListExportsCmd struct {
	Path string `help:"${list_exports_arg_path}" short:"p"`
}

// ModsCmd lists all mods in the specified instance
type ModsCmd struct {
	ID string `arg:"" help:"${mods.arg.id}"`
}

func (c *ModsCmd) Run(ctx *kong.Context) error {
	inst, err := launcher.FetchInstance(c.ID)
	if err != nil {
		return err
	}

	modsDir := filepath.Join(inst.Dir(), "mods")
	return listModsContents(modsDir, output.Translate("mods.empty"), inst.CachesDir(), inst)
}

// ResourcePacksCmd lists all resource packs in the specified instance
type ResourcePacksCmd struct {
	ID string `arg:"" help:"${resourcepacks.arg.id}"`
}

func (c *ResourcePacksCmd) Run(ctx *kong.Context) error {
	inst, err := launcher.FetchInstance(c.ID)
	if err != nil {
		return err
	}

	resourcePacksDir := filepath.Join(inst.Dir(), "resourcepacks")
	return listResourcePacksContents(resourcePacksDir, output.Translate("resourcepacks.empty"), inst.CachesDir(), inst)
}

// ShadersCmd lists all shader packs in the specified instance
type ShadersCmd struct {
	ID string `arg:"" help:"${shaders.arg.id}"`
}

func (c *ShadersCmd) Run(ctx *kong.Context) error {
	inst, err := launcher.FetchInstance(c.ID)
	if err != nil {
		return err
	}

	shadersDir := filepath.Join(inst.Dir(), "shaderpacks")
	return listDirectoryContents(shadersDir, output.Translate("shaders.empty"), "shaders")
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
func importInstance(archivePath, name string, force, merge bool) (launcher.Instance, error) {
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

	// Check for conflicting flags
	if force && merge {
		return launcher.Instance{}, fmt.Errorf("cannot use both --force and --merge flags")
	}

	var inst launcher.Instance
	instanceExists := launcher.DoesInstanceExist(instanceName)

	if instanceExists {
		if force {
			// Force mode: remove existing instance and create new one
			if err := launcher.RemoveInstance(instanceName); err != nil {
				return launcher.Instance{}, fmt.Errorf("remove existing instance: %w", err)
			}
			// Create new instance (will be done below)
		} else if merge {
			// Merge mode: use existing instance
			inst, err = launcher.FetchInstance(instanceName)
			if err != nil {
				return launcher.Instance{}, fmt.Errorf("fetch existing instance: %w", err)
			}
		} else {
			// Neither force nor merge: error
			return launcher.Instance{}, fmt.Errorf("instance '%s' already exists (use --force to overwrite or --merge to update)", instanceName)
		}
	}

	// Create new instance if it doesn't exist or was force-removed
	if !instanceExists || force {
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

		inst, err = launcher.CreateInstance(launcher.InstanceOptions{
			Name:          instanceName,
			GameVersion:   manifest.GameVersion,
			Loader:        loader,
			LoaderVersion: manifest.LoaderVersion,
			Config:        defaultInstanceConfig,
		})
		if err != nil {
			return launcher.Instance{}, fmt.Errorf("create instance: %w", err)
		}
	}

	// Extract files from archive (skip manifest.json)
	instanceDir := inst.Dir()

	for _, file := range zipReader.File {
		if file.Name == "manifest.json" {
			continue
		}

		// Normalize file path: replace backslashes with forward slashes for cross-platform compatibility
		normalizedName := strings.ReplaceAll(file.Name, "\\", "/")

		// Create destination path
		destPath := filepath.Join(instanceDir, normalizedName)

		// In merge mode, skip files that already exist
		if merge && instanceExists {
			if _, err := os.Stat(destPath); err == nil {
				// File already exists, skip it
				continue
			}
		}

		// Create directory if needed
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return launcher.Instance{}, fmt.Errorf("create directory for %s: %w", normalizedName, err)
		}

		// Extract file
		rc, err := file.Open()
		if err != nil {
			return launcher.Instance{}, fmt.Errorf("open file in archive %s: %w", file.Name, err)
		}
		defer rc.Close()

		destFile, err := os.Create(destPath)
		if err != nil {
			return launcher.Instance{}, fmt.Errorf("create destination file %s: %w", normalizedName, err)
		}
		defer destFile.Close()

		if _, err := io.Copy(destFile, rc); err != nil {
			return launcher.Instance{}, fmt.Errorf("extract file %s: %w", normalizedName, err)
		}
	}

	// Нормализуем пути в текстовых файлах для совместимости между Windows и Linux
	if err := normalizePathsInTextFiles(instanceDir); err != nil {
		return launcher.Instance{}, fmt.Errorf("normalize paths in extracted files: %w", err)
	}

	return inst, nil
}

// listDirectoryContents lists all files in the specified directory with their sizes using a pretty table
func listDirectoryContents(dirPath, emptyMessage, contentType string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			output.Info(emptyMessage)
			return nil
		}
		return fmt.Errorf("read directory %s: %w", dirPath, err)
	}

	// Filter out directories and collect files
	var files []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry)
		}
	}

	if len(files) == 0 {
		output.Info(emptyMessage)
		return nil
	}

	// Sort files by name
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	// Create table rows
	var rows []table.Row
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}

		// Format file size
		size := formatFileSize(info.Size())
		rows = append(rows, table.Row{file.Name(), size})
	}

	// Create and configure table
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{
		output.Translate(contentType + ".table.name"),
		output.Translate(contentType + ".table.size"),
	})
	t.AppendRows(rows)
	t.Render()

	return nil
}

// listModsContents lists all mods in the specified directory with links to CurseForge and Modrinth
func listModsContents(dirPath, emptyMessage, cachesDir string, inst launcher.Instance) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			output.Info(emptyMessage)
			return nil
		}
		return fmt.Errorf("read directory %s: %w", dirPath, err)
	}

	// Filter out directories and collect files
	var files []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".jar") {
			files = append(files, entry)
		}
	}

	if len(files) == 0 {
		output.Info(emptyMessage)
		return nil
	}

	// Sort files by name
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	// Create table rows with mod links
	var rows []table.Row
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}

		// Extract mod info from filename
		modInfo := meta.ExtractModInfoFromFilename(file.Name())

		// Get links from CurseForge and Modrinth with loader/version filtering
		modInfo = meta.GetModLinks(modInfo, cachesDir, string(inst.Loader), inst.GameVersion)

		// Format links (show full URLs for mods)
		curseForgeLink := ""
		if modInfo.CurseForgeURL != "" {
			curseForgeLink = modInfo.CurseForgeURL
		}

		modrinthLink := ""
		if modInfo.ModrinthURL != "" {
			modrinthLink = modInfo.ModrinthURL
		}

		// Format file size
		size := formatFileSize(info.Size())

		rows = append(rows, table.Row{file.Name(), curseForgeLink, modrinthLink, size})
	}

	// Create and configure table
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{
		output.Translate("mods.table.name"),
		output.Translate("mods.table.curseforge"),
		output.Translate("mods.table.modrinth"),
		output.Translate("mods.table.size"),
	})
	t.AppendRows(rows)
	t.Render()

	return nil
}

// listResourcePacksContents lists all resource packs in the specified directory with links to CurseForge and Modrinth
func listResourcePacksContents(dirPath, emptyMessage, cachesDir string, inst launcher.Instance) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			output.Info(emptyMessage)
			return nil
		}
		return fmt.Errorf("read directory %s: %w", dirPath, err)
	}

	// Filter out directories and collect resource pack files
	var files []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			// Resource packs can be .zip, .jar, or have no extension
			name := strings.ToLower(entry.Name())
			if strings.HasSuffix(name, ".zip") || strings.HasSuffix(name, ".jar") ||
				(!strings.Contains(name, ".") && !entry.IsDir()) {
				files = append(files, entry)
			}
		}
	}

	if len(files) == 0 {
		output.Info(emptyMessage)
		return nil
	}

	// Sort files by name
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	// Create table rows with resource pack links
	var rows []table.Row
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}

		// Extract resource pack info from filename
		rpInfo := meta.ExtractResourcePackInfo(file.Name())

		// Get links from CurseForge and Modrinth
		rpInfo = meta.GetResourcePackLinks(rpInfo, cachesDir, inst.GameVersion)

		// Format links
		curseForgeLink := ""
		if rpInfo.CurseForgeURL != "" {
			curseForgeLink = rpInfo.CurseForgeURL
		}

		modrinthLink := ""
		if rpInfo.ModrinthURL != "" {
			modrinthLink = rpInfo.ModrinthURL
		}

		// Format file size
		size := formatFileSize(info.Size())

		rows = append(rows, table.Row{rpInfo.Name, curseForgeLink, modrinthLink, size})
	}

	// Create and configure table
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{
		output.Translate("resourcepacks.table.name"),
		output.Translate("resourcepacks.table.curseforge"),
		output.Translate("resourcepacks.table.modrinth"),
		output.Translate("resourcepacks.table.size"),
	})
	t.AppendRows(rows)
	t.Render()

	return nil
}

// formatFileSize formats file size in human readable format
func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// normalizePathsInTextFiles нормализует пути в текстовых файлах после импорта,
// заменяя обратные слэши на прямые для совместимости между Windows и Linux
func normalizePathsInTextFiles(instanceDir string) error {
	// Файлы, которые могут содержать пути и требуют нормализации
	filesToProcess := []string{
		"options.txt",
	}

	// Также обрабатываем все файлы в директориях config и defaultconfigs
	directoriesToProcess := []string{
		"config",
		"defaultconfigs",
	}

	normalizeFile := func(filePath string) error {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}

		originalContent := string(content)

		// Заменяем обратные слэши на прямые слэши
		// Это простой и безопасный подход для большинства случаев
		normalizedContent := strings.ReplaceAll(originalContent, "\\", "/")

		// Если содержимое изменилось, записываем обратно
		if normalizedContent != originalContent {
			return os.WriteFile(filePath, []byte(normalizedContent), 0644)
		}

		return nil
	}

	// Обрабатываем отдельные файлы
	for _, file := range filesToProcess {
		filePath := filepath.Join(instanceDir, file)
		if _, err := os.Stat(filePath); err == nil {
			if err := normalizeFile(filePath); err != nil {
				return fmt.Errorf("normalize paths in %s: %w", file, err)
			}
		}
	}

	// Обрабатываем директории
	for _, dir := range directoriesToProcess {
		dirPath := filepath.Join(instanceDir, dir)
		if _, err := os.Stat(dirPath); err == nil {
			err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && (strings.HasSuffix(strings.ToLower(path), ".txt") ||
					strings.HasSuffix(strings.ToLower(path), ".cfg") ||
					strings.HasSuffix(strings.ToLower(path), ".toml") ||
					strings.HasSuffix(strings.ToLower(path), ".json") ||
					strings.HasSuffix(strings.ToLower(path), ".properties")) {
					return normalizeFile(path)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("normalize paths in directory %s: %w", dir, err)
			}
		}
	}

	return nil
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
