package cmd

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"QMLauncher/internal/cli/output"

	"github.com/alecthomas/kong"
	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
)

// ServersCmd represents the servers command
type ServersCmd struct {
	Search string `help:"Поиск серверов по названию"`
	Filter string `help:"Фильтр серверов (premium, online)" enum:"premium,online,all" default:"all"`
	Limit  int    `help:"Ограничить количество отображаемых серверов" default:"10"`
}

func (c *ServersCmd) Run(ctx *kong.Context) error {
	// Fetch servers list from QMServer Cloud with progress indication
	output.Progress("Получение списка серверов из QMServer Cloud...")

	// Create indeterminate progress bar for API call
	bar := output.CreateIndeterminateBar("Загрузка данных")

	serversResponse, err := getQMServersList()
	bar.Finish()

	if err != nil {
		return fmt.Errorf("не удалось получить список серверов: %w", err)
	}

	if serversResponse.Error != "" {
		return fmt.Errorf("ошибка QMServer Cloud: %s", serversResponse.Error)
	}

	if len(serversResponse.ServerProfiles) == 0 {
		output.Info("Серверы не найдены")
		return nil
	}

	// Apply filters and search
	filteredServers := filterServers(serversResponse.ServerProfiles, c.Search, c.Filter)

	if len(filteredServers) == 0 {
		output.Info("После применения фильтров серверы не найдены")
		return nil
	}

	// Sort servers by priority: Premium first, then by creation date (newest first)
	sort.Slice(filteredServers, func(i, j int) bool {
		a, b := filteredServers[i], filteredServers[j]

		// First priority: Premium servers
		if a.IsPremium != b.IsPremium {
			return a.IsPremium // Premium first
		}

		// Second priority: By creation date (newest first)
		return a.CreatedAt > b.CreatedAt
	})

	// Apply limit
	if c.Limit > 0 && len(filteredServers) > c.Limit {
		filteredServers = filteredServers[:c.Limit]
	}

	// Display servers in table
	color.New(color.Bold).Println("Список серверов QMServer Cloud:")
	fmt.Println()

	t := table.NewWriter()
	t.SetOutputMirror(color.Output)
	t.AppendHeader(table.Row{
		"ID", "Название", "Адрес", "Версия", "Модлоадер", "Premium",
	})

	for _, server := range filteredServers {
		// Format address
		address := fmt.Sprintf("%s:%d", server.Host, server.Port)

		// Format mod loader
		modLoader := server.ModLoader
		if server.ModLoaderVersion != "" {
			modLoader = fmt.Sprintf("%s %s", server.ModLoader, server.ModLoaderVersion)
		}

		// Format premium
		premium := "Нет"
		if server.IsPremium {
			premium = color.New(color.FgYellow).Sprint("Да")
		}

		t.AppendRow(table.Row{
			strconv.Itoa(int(server.ID)),
			server.Name,
			address,
			server.Version,
			modLoader,
			premium,
		})
	}

	t.Render()
	fmt.Printf("\nПоказаны серверы: %d (всего: %d)\n", len(filteredServers), serversResponse.Count)

	return nil
}

// filterServers applies search and filter criteria to server list
func filterServers(servers []QMServerInfo, search, filter string) []QMServerInfo {
	var filtered []QMServerInfo

	for _, server := range servers {
		// Apply search filter (case-insensitive name search)
		if search != "" {
			if !strings.Contains(strings.ToLower(server.Name), strings.ToLower(search)) {
				continue
			}
		}

		// Apply category filter
		switch strings.ToLower(filter) {
		case "premium":
			if !server.IsPremium {
				continue
			}
		case "online":
			// For now, assume all servers are online since we don't have real status
			// In future, this could check actual server status
			// For now, include all servers (no online/offline distinction)
		case "all", "":
			// Include all servers
		default:
			// Unknown filter, include server
		}

		filtered = append(filtered, server)
	}

	return filtered
}
