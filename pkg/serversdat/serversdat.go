package serversdat

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Tnze/go-mc/nbt"
)

// ServerEntry represents a single server in servers.dat
type ServerEntry struct {
	Name           string `nbt:"name"`
	IP             string `nbt:"ip"`
	Icon           string `nbt:"icon,omitempty"`
	AcceptTextures byte   `nbt:"acceptTextures,omitempty"`
}

// ServersDatRoot is the root structure of servers.dat
type ServersDatRoot struct {
	Servers []ServerEntry `nbt:"servers"`
}

// UpdateOrAddServer updates or adds the given server to servers.dat in the instance directory.
// instanceDir is the Minecraft instance/game directory (e.g. .qmlauncher/instances/MyInstance).
// serverName is the display name, serverAddress is "host:port".
func UpdateOrAddServer(instanceDir, serverName, serverAddress string) error {
	if serverName == "" || serverAddress == "" {
		return nil
	}
	path := filepath.Join(instanceDir, "servers.dat")
	var root ServersDatRoot

	// Read existing if present
	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		if err := nbt.Unmarshal(data, &root); err != nil {
			// If unmarshal fails, start fresh
			root = ServersDatRoot{Servers: []ServerEntry{}}
		}
	}
	if root.Servers == nil {
		root.Servers = []ServerEntry{}
	}

	// Normalize address (ensure no protocol prefix)
	addr := strings.TrimSpace(serverAddress)
	if idx := strings.Index(addr, "://"); idx >= 0 {
		addr = addr[idx+3:]
	}

	// Update existing by IP or add new
	found := false
	for i := range root.Servers {
		if root.Servers[i].IP == addr {
			root.Servers[i].Name = serverName
			found = true
			break
		}
	}
	if !found {
		root.Servers = append(root.Servers, ServerEntry{
			Name: serverName,
			IP:   addr,
		})
	}

	// Write back (uncompressed NBT)
	var buf bytes.Buffer
	if err := nbt.NewEncoder(&buf).Encode(root, ""); err != nil {
		return fmt.Errorf("encode servers.dat: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write servers.dat: %w", err)
	}
	return nil
}
