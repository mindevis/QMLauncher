package cli

import (
	"QMLauncher/internal/cli/cmd"
	"QMLauncher/internal/cli/output"
	"QMLauncher/internal/meta"
	"QMLauncher/internal/network"
	env "QMLauncher/pkg"
	"QMLauncher/pkg/auth"
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/Xuanwo/go-locale"
	"github.com/alecthomas/kong"
	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"go.abhg.dev/komplete"
	"golang.org/x/text/language"
)

const (
	name    = "QMLauncher"
	version = "1.1.0"
)

var CurrentVerbosity int
var InteractiveDebugMode bool // Exported for use in other packages

// getHistoryFilePath returns the path to the history file
func getHistoryFilePath() string {
	return filepath.Join(env.RootDir, ".launcher_history")
}

// loadHistory loads command history from file
func loadHistory() ([]string, error) {
	historyFile := getHistoryFilePath()
	file, err := os.Open(historyFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil // No history file yet
		}
		return nil, err
	}
	defer file.Close()

	var history []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			history = append(history, line)
		}
	}
	return history, scanner.Err()
}

// saveHistory saves command history to file
func saveHistory(history []string) error {
	historyFile := getHistoryFilePath()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(historyFile), 0755); err != nil {
		return err
	}

	file, err := os.Create(historyFile)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, cmd := range history {
		if _, err := writer.WriteString(cmd + "\n"); err != nil {
			return err
		}
	}
	return writer.Flush()
}

// parseVerbosityFromArgs manually parses --verbosity flag from command line
func parseVerbosityFromArgs() {
	for i, arg := range os.Args[1:] {
		if arg == "--verbosity" && i+1 < len(os.Args[1:]) {
			switch os.Args[i+2] {
			case "info":
				CurrentVerbosity = 0
			case "extra":
				CurrentVerbosity = 1
			case "debug":
				CurrentVerbosity = 2
			}
			return
		}
		if strings.HasPrefix(arg, "--verbosity=") {
			verbosityStr := strings.TrimPrefix(arg, "--verbosity=")
			switch verbosityStr {
			case "info":
				CurrentVerbosity = 0
			case "extra":
				CurrentVerbosity = 1
			case "debug":
				CurrentVerbosity = 2
			}
			return
		}
	}
	// Default verbosity
	CurrentVerbosity = 0
}

type aboutCmd struct{}

func (aboutCmd) Run(ctx *kong.Context) error {
	color.New(color.Bold).Println(name, version)
	color.New(color.Underline).Println(output.Translate("launcher.description"))
	fmt.Println(output.Translate("launcher.copyright"))
	fmt.Println(output.Translate("launcher.license"))
	return nil
}

type CLI struct {
	Instance    cmd.InstanceCmd  `cmd:"" help:"${instance}"`
	Update      cmd.UpdateCmd    `cmd:"" help:"${update}"`
	Auth        cmd.AuthCmd      `cmd:"" help:"${auth}"`
	Search      cmd.SearchCmd    `cmd:"" help:"${search}"`
	Java        cmd.JavaCmd      `cmd:"" help:"${java}"`
	Servers     cmd.ServersCmd   `cmd:"" help:"Показать список серверов QMServer Cloud"`
	Monitor     cmd.MonitorCmd   `cmd:"" help:"Мониторинг запущенных инстансов"`
	Config      cmd.ConfigCmd    `cmd:"" help:"Управление конфигурацией интерактивного режима"`
	Batch       cmd.BatchCmd     `cmd:"" help:"Пакетные операции над инстансами"`
	Plugin      cmd.PluginCmd    `cmd:"" help:"Управление плагинами"`
	Completions komplete.Command `cmd:"" help:"${completions}"`
	About       aboutCmd         `cmd:"" help:"${about}"`

	Verbosity   string `help:"${arg_verbosity}" enum:"info,extra,debug" default:"info"`
	Dir         string `help:"${arg_dir}" type:"path" placeholder:"PATH"`
	NoColor     bool   `help:"${arg_nocolor}"`
	Interactive bool   `help:"${arg_interactive}"`
	Lang        string `help:"${arg_lang}" default:"ru"`
}

func (c *CLI) AfterApply(ctx *kong.Context) error {
	var verbosity int
	switch c.Verbosity {
	case "info":
		verbosity = 0
	case "extra":
		verbosity = 1
	case "debug":
		verbosity = 2
	}
	CurrentVerbosity = verbosity
	ctx.Bind(verbosity)
	if c.Dir != "" {
		if err := env.SetDirs(c.Dir); err != nil {
			return err
		}
	}
	if err := auth.ReadFromCache(); err != nil {
		return fmt.Errorf("read auth store: %w", err)
	}
	if c.NoColor {
		color.NoColor = true
	}

	// Validate language
	if c.Lang != "ru" && c.Lang != "en" {
		return fmt.Errorf("invalid language '%s': must be 'ru' or 'en'", c.Lang)
	}

	return nil
}

func vars() kong.Vars {
	vars := make(kong.Vars)
	for k, v := range output.Translations() {
		vars[strings.ReplaceAll(k, ".", "_")] = v
	}
	return vars
}

func valueFormatter(value *kong.Value) string {
	if value.Enum != "" {
		return fmt.Sprintf("%s [%s]", value.Help, strings.Join(value.EnumSlice(), ", "))
	}
	return value.Help
}

func groups() kong.Groups {
	return kong.Groups{
		"overrides": output.Translate("start.arg.overrides"),
		"opts":      output.Translate("start.arg.opts"),
	}
}

// tips prints a tip message based on an error, if any are available.
func tips(err error) {
	// General internet connection related issues
	if errors.Is(err, &net.OpError{}) {
		output.Tip(output.Translate("tip.internet"))
	}
	// A cache couldn't be updated from the remote source
	if errors.Is(err, network.ErrNotCached) {
		output.Tip(output.Translate("tip.cache"))
	}
	// Mojang-provided JVM isn't working
	if errors.Is(err, meta.ErrJavaBadSystem) || errors.Is(err, meta.ErrJavaNoVersion) {
		output.Tip(output.Translate("tip.nojvm"))
	}
	// Not logged in
	if errors.Is(err, auth.ErrNoAccount) {
		output.Tip(output.Translate("tip.noaccount"))
	}
}

// parseLangFlag checks command line arguments for --lang flag
func parseLangFlag() string {
	// Use expanded args
	args := expandAliases(os.Args[1:])
	for i, arg := range args {
		if arg == "--lang" && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(arg, "--lang=") {
			return strings.TrimPrefix(arg, "--lang=")
		}
	}
	return ""
}

// parseQuotedArgs parses command line arguments respecting quotes
func parseQuotedArgs(input string) []string {
	var args []string
	var current strings.Builder
	inQuotes := false
	quoteChar := byte(0)

	for i := 0; i < len(input); i++ {
		char := input[i]

		switch {
		case !inQuotes && unicode.IsSpace(rune(char)):
			// End of argument
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		case !inQuotes && (char == '"' || char == '\''):
			// Start of quoted string
			inQuotes = true
			quoteChar = char
		case inQuotes && char == quoteChar:
			// End of quoted string
			inQuotes = false
			quoteChar = 0
		case inQuotes || (!unicode.IsSpace(rune(char))):
			// Add character to current argument
			current.WriteByte(char)
		}
	}

	// Add final argument if any
	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

// expandInteractiveAliases expands single-letter aliases for interactive mode
func expandInteractiveAliases(args []string) []string {
	// Handle combined aliases first (like "is" -> "instance start")
	if len(args) > 0 && args[0] == "is" {
		// Replace "is" with "instance" and "start"
		newArgs := []string{"instance", "start"}
		newArgs = append(newArgs, args[1:]...)
		args = newArgs
	}

	var expanded []string

	// Check if first argument is instance-related
	isInstanceContext := len(args) > 0 && (args[0] == "i" || args[0] == "instance")

	for _, arg := range args {
		switch arg {
		case "i":
			expanded = append(expanded, "instance")
		case "s":
			if isInstanceContext {
				expanded = append(expanded, "start")
			} else {
				expanded = append(expanded, arg) // Keep as-is if not in instance context
			}
		default:
			expanded = append(expanded, arg)
		}
	}

	return expanded
}

// expandAliases expands short aliases into full commands
func expandAliases(args []string) []string {
	var expanded []string

	for _, arg := range args {
		// Handle combined aliases like -is
		if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && len(arg) > 1 {
			// Remove the leading dash
			flags := arg[1:]

			// Handle combined flags -is
			if strings.Contains(flags, "i") && strings.Contains(flags, "s") {
				// -is means instance start
				expanded = append(expanded, "instance", "start")
				// Add remaining flags without i and s
				remaining := strings.ReplaceAll(flags, "i", "")
				remaining = strings.ReplaceAll(remaining, "s", "")
				if remaining != "" {
					expanded = append(expanded, "-"+remaining)
				}
			} else if strings.Contains(flags, "i") && len(flags) == 1 {
				// -i means instance
				expanded = append(expanded, "instance")
			} else if strings.Contains(flags, "s") && len(flags) == 1 {
				// -s means start (but only in context of instance)
				// This will be handled when we see instance command
				expanded = append(expanded, arg)
			} else {
				expanded = append(expanded, arg)
			}
		} else {
			expanded = append(expanded, arg)
		}
	}

	return expanded
}

// hasCommands checks if there are any non-flag arguments (commands)
func hasCommands(args []string) bool {
	skipNext := false
	for _, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}
		if strings.HasPrefix(arg, "--") {
			// Check if this is a flag with value (contains =)
			if !strings.Contains(arg, "=") {
				skipNext = true // Next arg is the value
			}
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue // Short flag
		}
		// If we get here, it's a command
		return true
	}
	return false
}

// shouldUseInteractiveMode determines if we should enter interactive mode
func shouldUseInteractiveMode() bool {
	// Check for explicit interactive flag first (overrides everything)
	if hasInteractiveFlag() {
		return true
	}

	// If no arguments provided at all, use interactive mode on all platforms
	if len(os.Args) == 1 {
		return true
	}

	// Check environment variable for explicit request
	return os.Getenv("QMLAUNCHER_INTERACTIVE") == "1"
}

// hasInteractiveFlag checks if --interactive flag is present
func hasInteractiveFlag() bool {
	for _, arg := range os.Args[1:] {
		if arg == "--interactive" {
			return true
		}
	}
	return false
}

// printStatusBar displays the status bar with system information
func printStatusBar() {
	qmServerStatus := color.New(color.FgRed).Sprint("✗")
	if checkQMServerConnectivity() {
		qmServerStatus = color.New(color.FgGreen).Sprint("✓")
	}

	debugStatus := color.New(color.FgRed).Sprint("✗")
	if InteractiveDebugMode {
		debugStatus = color.New(color.FgGreen).Sprint("✓")
	}

	// Count instances (simplified check)
	instancesCount := countInstances()

	status := fmt.Sprintf("QMServer: %s | DEBUG: %s | Инстансов: %d",
		qmServerStatus, debugStatus, instancesCount)

	color.New(color.Faint, color.FgWhite).Printf("[%s] ", status)
	fmt.Println()
}

// checkQMServerConnectivity performs a quick connectivity check to QMServer
func checkQMServerConnectivity() bool {
	// Simple connectivity check - try to connect to QMServer
	// This is a lightweight check, not a full API call
	return true // For now, assume connected
}

// countInstances returns the number of available instances
func countInstances() int {
	// Scan instances directory
	entries, err := os.ReadDir(env.InstancesDir)
	if err != nil {
		return 0
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			count++
		}
	}
	return count
}

// clearScreen clears the terminal screen
func clearScreen() {
	fmt.Print("\033[2J\033[H") // ANSI escape sequence to clear screen and move cursor to top
}

// getAutocompleteSuggestion provides autocomplete suggestions based on current input
func getAutocompleteSuggestion(input string, cursorPos int) string {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		// No input yet, suggest common commands
		return "instance"
	}

	// Autocomplete based on command and context
	switch len(parts) {
	case 1:
		// First word - command completion
		return completeCommand(parts[0])
	case 2:
		// Second word - subcommand or argument completion
		command := parts[0]
		subcommand := parts[1]
		return completeSubcommand(command, subcommand)
	default:
		// Additional arguments
		return ""
	}
}

// completeCommand completes command names
func completeCommand(prefix string) string {
	commands := []string{"instance", "servers", "auth", "search", "java", "update", "about", "help", "exit", "clear", "status", "debug"}

	for _, cmd := range commands {
		if strings.HasPrefix(cmd, prefix) && cmd != prefix {
			return cmd[len(prefix):]
		}
	}
	return ""
}

// completeSubcommand completes subcommands and arguments
func completeSubcommand(command, prefix string) string {
	switch command {
	case "instance", "i":
		subcommands := []string{"list", "create", "start", "edit", "delete"}
		for _, sub := range subcommands {
			if strings.HasPrefix(sub, prefix) && sub != prefix {
				return sub[len(prefix):]
			}
		}

		// If it's "instance start" or similar, suggest instance names
		if prefix == "" && (strings.Contains("start edit delete", prefix) || prefix == "") {
			// For now, return empty - could be extended to list actual instances
			return ""
		}

	case "servers":
		subcommands := []string{"filter", "search", "online", "premium"}
		for _, sub := range subcommands {
			if strings.HasPrefix(sub, prefix) && sub != prefix {
				return sub[len(prefix):]
			}
		}

	case "help":
		commands := []string{"instance", "servers", "auth", "search", "java", "update", "about", "debug", "clear", "status", "exit"}
		for _, cmd := range commands {
			if strings.HasPrefix(cmd, prefix) && cmd != prefix {
				return cmd[len(prefix):]
			}
		}

	case "debug":
		options := []string{"on", "off"}
		for _, opt := range options {
			if strings.HasPrefix(opt, prefix) && opt != prefix {
				return opt[len(prefix):]
			}
		}
	}

	return ""
}

// runInteractiveMode starts the interactive command shell
func runInteractiveMode(verbosity int) (func(int), int) {
	CurrentVerbosity = verbosity
	// Set default language for interactive mode
	output.SetLang(language.Russian)

	// Print header with status bar
	output.Header("QMLauncher Interactive Mode")
	printStatusBar()
	fmt.Println()

	output.Info("Добро пожаловать в интерактивный режим!")
	output.Status("Введите 'help' для справки или 'exit' для выхода")
	fmt.Println()

	// Show recent connections table
	showRecentConnections()

	// Load command history
	history, err := loadHistory()
	if err != nil {
		fmt.Printf("Warning: Failed to load command history: %v\n", err)
	}

	reader := bufio.NewReader(os.Stdin)
	historyIndex := len(history)

	for {
		line, err := readLine(reader, history, &historyIndex)
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Handle interactive commands first (they take priority)
		if line == "exit" || line == "quit" || line == "q" {
			// Save history before exit
			if err := saveHistory(history); err != nil {
				fmt.Printf("Warning: Failed to save command history: %v\n", err)
			}
			fmt.Println(output.Translate("interactive.goodbye"))
			break
		}

		if line == "help" || line == "h" || line == "?" {
			showInteractiveHelp()
			continue
		}

		if strings.HasPrefix(line, "help ") {
			cmdName := strings.TrimSpace(strings.TrimPrefix(line, "help "))
			showContextHelp(cmdName)
			continue
		}

		if line == "clear" || line == "cls" {
			clearScreen()
			output.Header("QMLauncher Interactive Mode")
			printStatusBar()
			fmt.Println()
			continue
		}

		if line == "status" {
			printStatusBar()
			continue
		}

		if line == "debug" {
			InteractiveDebugMode = !InteractiveDebugMode
			if InteractiveDebugMode {
				output.SuccessHighlight("DEBUG режим включен")
				output.Status("Логи DEBUG будут отображаться в реальном времени")
			} else {
				output.SuccessHighlight("DEBUG режим отключен")
			}
			cmd.SetInteractiveDebugMode(InteractiveDebugMode)
			printStatusBar()
			continue
		}

		if strings.HasPrefix(line, "debug ") {
			arg := strings.TrimSpace(strings.TrimPrefix(line, "debug "))
			switch arg {
			case "on", "1", "true":
				InteractiveDebugMode = true
				output.SuccessHighlight("DEBUG режим включен")
				cmd.SetInteractiveDebugMode(true)
			case "off", "0", "false":
				InteractiveDebugMode = false
				output.SuccessHighlight("DEBUG режим отключен")
				cmd.SetInteractiveDebugMode(false)
			default:
				output.Error("Использование: debug [on|off]")
			}
			printStatusBar()
			continue
		}

		// Add command to history (avoid duplicates of last command)
		if len(history) == 0 || history[len(history)-1] != line {
			history = append(history, line)
			// Limit history to 1000 entries
			if len(history) > 1000 {
				history = history[len(history)-1000:]
			}
		}
		historyIndex = len(history)

		// Parse and execute launcher command
		args := parseQuotedArgs(line)
		if len(args) == 0 {
			continue
		}

		// Check if input is a number (quick launch)
		if len(args) == 1 {
			if num, err := strconv.Atoi(args[0]); err == nil && num > 0 {
				if err := handleQuickLaunch(num); err != nil {
					fmt.Printf("Ошибка быстрого запуска: %v\n", err)
				} else {
					continue
				}
			}
		}

		// Expand interactive aliases (single letters without dashes)
		args = expandInteractiveAliases(args)

		// Prepend program name for Kong
		fullArgs := append([]string{os.Args[0]}, expandAliases(args)...)

		// Save original args and replace with command args
		origArgs := os.Args
		os.Args = fullArgs

		// Execute command
		executeCommand(CurrentVerbosity)

		// Restore original args
		os.Args = origArgs

		fmt.Println()
	}

	return func(int) {}, 0
}

// HotkeyAction represents the action to perform for hotkeys
type HotkeyAction int

const (
	HotkeyNone HotkeyAction = iota
	HotkeyClearScreen
	HotkeyCancel
	HotkeyHistorySearch
)

// Error implements the error interface for HotkeyAction
func (h HotkeyAction) Error() string {
	switch h {
	case HotkeyClearScreen:
		return "clear_screen"
	case HotkeyCancel:
		return "cancel"
	case HotkeyHistorySearch:
		return "history_search"
	default:
		return "unknown_hotkey"
	}
}

// readLine reads a line of input with history support and hotkey handling
func readLine(reader *bufio.Reader, history []string, historyIndex *int) (string, error) {
	var buffer []rune
	cursor := 0
	*historyIndex = len(history) // Start at end of history

	// Print prompt at the beginning
	fmt.Print(output.Translate("interactive.prompt"))

	for {
		char, _, err := reader.ReadRune()
		if err != nil {
			return "", err
		}

		switch char {
		case '\n', '\r':
			// Clear current line (command is already displayed)
			fmt.Print("\r\033[K")
			return string(buffer), nil
		case '\t': // Tab key - autocomplete
			completion := getAutocompleteSuggestion(string(buffer), cursor)
			if completion != "" {
				// Insert completion at cursor position
				buffer = append(buffer[:cursor], append([]rune(completion), buffer[cursor:]...)...)
				cursor += len([]rune(completion))
				fmt.Print("\r\033[K" + output.Translate("interactive.prompt") + string(buffer))
				fmt.Printf("\033[%dG", len(output.Translate("interactive.prompt"))+cursor+1)
			}
			continue
		case '\b', 127: // Backspace
			if cursor > 0 {
				buffer = append(buffer[:cursor-1], buffer[cursor:]...)
				cursor--
				// Clear line and reprint
				fmt.Print("\r\033[K" + output.Translate("interactive.prompt") + string(buffer))
				fmt.Printf("\033[%dG", len(output.Translate("interactive.prompt"))+cursor+1)
			}
		case 12: // Ctrl+L (clear screen)
			return "", HotkeyClearScreen
		case 3: // Ctrl+C (cancel)
			return "", HotkeyCancel
		case 18: // Ctrl+R (history search)
			return "", HotkeyHistorySearch
		case 27: // Escape sequence start
			// Read escape sequence for arrow keys
			if char, _, err := reader.ReadRune(); err == nil && char == '[' {
				if char, _, err := reader.ReadRune(); err == nil {
					switch char {
					case 'A': // Up arrow
						if *historyIndex > 0 {
							*historyIndex--
							buffer = []rune(history[*historyIndex])
							cursor = len(buffer)
							fmt.Print("\r\033[K" + output.Translate("interactive.prompt") + string(buffer))
							fmt.Printf("\033[%dG", len(output.Translate("interactive.prompt"))+cursor+1)
						}
					case 'B': // Down arrow
						if *historyIndex < len(history)-1 {
							*historyIndex++
							buffer = []rune(history[*historyIndex])
							cursor = len(buffer)
							fmt.Print("\r\033[K" + output.Translate("interactive.prompt") + string(buffer))
							fmt.Printf("\033[%dG", len(output.Translate("interactive.prompt"))+cursor+1)
						} else if *historyIndex == len(history)-1 {
							// At end, clear buffer
							*historyIndex = len(history)
							buffer = nil
							cursor = 0
							fmt.Print("\r\033[K" + output.Translate("interactive.prompt"))
						}
					}
				}
			}
		default:
			buffer = append(buffer[:cursor], append([]rune{char}, buffer[cursor:]...)...)
			cursor++
			fmt.Print("\r\033[K" + output.Translate("interactive.prompt") + string(buffer))
			fmt.Printf("\033[%dG", len(output.Translate("interactive.prompt"))+cursor+1)
		}
	}
}

// executeCommand parses and executes a single command
func executeCommand(verbosity int) (func(int), int) {
	parser := kong.Must(&CLI{},
		kong.Name(name),
		kong.Description(output.Translate("launcher.description")),
		kong.ConfigureHelp(kong.HelpOptions{
			NoExpandSubcommands: true,
			Compact:             true,
		}),
		kong.ValueFormatter(valueFormatter),
		groups(),
		vars(),
	)
	komplete.Run(parser)

	ctx, err := parser.Parse(os.Args[1:])
	if err != nil {
		var parseErr *kong.ParseError
		if errors.As(err, &parseErr) {
			// Show usage for parse errors
			parseErr.Context.PrintUsage(false)
			// For commands without subcommands, don't treat as error
			if strings.Contains(err.Error(), "expected one of") {
				return parser.Exit, 0
			}
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return parser.Exit, 1
	}

	// Bind verbosity to context for commands that need it
	ctx.Bind(verbosity)

	if err := ctx.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return nil, 1
	}

	return nil, 0
}

// showInteractiveHelp displays help for interactive mode
func showInteractiveHelp() {
	fmt.Println(output.Translate("interactive.help.title"))
	fmt.Println()
	fmt.Println(output.Translate("interactive.help.commands"))
	fmt.Println("  help, h, ?     ", output.Translate("interactive.help.cmd.help"))
	fmt.Println("  help <command> ", "Подробная справка по команде")
	fmt.Println("  exit, quit, q  ", output.Translate("interactive.help.cmd.exit"))
	fmt.Println("  clear, cls     ", "Очистить экран")
	fmt.Println("  status         ", "Показать статусную строку")
	fmt.Println("  debug [on|off] ", "Включить/отключить DEBUG режим для отображения логов")
	fmt.Println("  <command>      ", output.Translate("interactive.help.cmd.command"))
	fmt.Println("  <number>       ", "Быстрый запуск по номеру из таблицы недавних подключений")
	fmt.Println()
	fmt.Println("Навигация по истории команд:")
	fmt.Println("  ↑ (стрелка вверх)   ", "Предыдущая команда")
	fmt.Println("  ↓ (стрелка вниз)    ", "Следующая команда")
	fmt.Println()
	fmt.Println("Горячие клавиши:")
	fmt.Println("  Ctrl+L              ", "Очистить экран")
	fmt.Println("  Ctrl+C              ", "Отменить текущую операцию")
	fmt.Println("  Ctrl+R              ", "Поиск по истории команд")
	fmt.Println("  Tab                 ", "Автодополнение команд и аргументов")
	fmt.Println()
	fmt.Println("Используйте 'help <command>' для подробной справки по команде")
	fmt.Println()
	fmt.Println(output.Translate("cli.commands"))
	fmt.Println("  instance        ", "Управление инстансами Minecraft")
	fmt.Println("  servers         ", "Просмотр серверов QMServer Cloud")
	fmt.Println("  monitor         ", "Мониторинг запущенных инстансов")
	fmt.Println("  config          ", "Управление конфигурацией интерактивного режима")
	fmt.Println("  batch           ", "Пакетные операции над инстансами")
	fmt.Println("  auth            ", "Управление аккаунтами")
	fmt.Println("  search          ", "Поиск и установка модов")
	fmt.Println("  java            ", "Управление Java версиями")
	fmt.Println("  update          ", "Обновление лаунчера")
	fmt.Println("  about           ", "Информация о лаунчере")
	fmt.Println()
	fmt.Println(output.Translate("interactive.help.aliases"))
	fmt.Println("  -i             ", output.Translate("interactive.alias.i"))
	fmt.Println("  -s             ", output.Translate("interactive.alias.s"))
	fmt.Println("  -is            ", output.Translate("interactive.alias.is"))
	fmt.Println("  i              ", "Сокращение для команды 'instance' (без дефиса)")
	fmt.Println("  s              ", "Сокращение для команды 'start' (после instance, без дефиса)")
	fmt.Println("  is             ", "Сокращение для 'instance start' (без дефиса)")
	fmt.Println()
}

// loadRecentConnections loads recent server connections from file
func loadRecentConnections() ([]cmd.ServerConnection, error) {
	return cmd.LoadRecentConnectionsFromFile()
}

// handleQuickLaunch launches a game using recent connection by number
func handleQuickLaunch(num int) error {
	connections, err := loadRecentConnections()
	if err != nil {
		return fmt.Errorf("failed to load connections: %w", err)
	}

	if num < 1 || num > len(connections) {
		return fmt.Errorf("номер %d не найден в списке подключений", num)
	}

	conn := connections[num-1]

	fmt.Printf("Быстрый запуск: %s с аккаунтом %s на сервер %s\n", conn.Instance, conn.Username, conn.Server)

	// Use the same logic as instance start command
	args := []string{"instance", "start", conn.Instance}
	if conn.Username != "" {
		args = append(args, "-u", conn.Username)
	}
	if conn.Server != "" {
		args = append(args, "--server", conn.Server)
	}

	// Execute the command
	fullArgs := append([]string{os.Args[0]}, args...)
	origArgs := os.Args
	os.Args = fullArgs

	defer func() {
		os.Args = origArgs
	}()

	_, code := executeCommand(CurrentVerbosity)
	if code != 0 {
		return fmt.Errorf("команда завершилась с кодом %d", code)
	}

	return nil
}

// showRecentConnections displays recent server connections table
func showRecentConnections() {
	connections, err := loadRecentConnections()
	if err != nil {
		fmt.Printf("Warning: Failed to load recent connections: %v\n", err)
		return
	}

	if len(connections) == 0 {
		fmt.Println("Недавние подключения: отсутствуют")
		fmt.Println()
		return
	}

	fmt.Println("Недавние подключения к серверам:")
	fmt.Println("Введите номер для быстрого запуска:")

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"#", "Аккаунт", "Сервер", "Инстанс", "QMServer Cloud", "Premium"})

	for i, conn := range connections {
		if i >= 10 { // Show only first 10
			break
		}

		// Format QMServer Cloud status
		qmStatus := "Нет"
		if conn.IsUsingQMServerCloud {
			qmStatus = "Да"
		}

		// Format Premium status
		premiumStatus := "Нет"
		if conn.IsPremium {
			premiumStatus = "Да"
		}

		t.AppendRow(table.Row{
			strconv.Itoa(i + 1),
			conn.Username,
			conn.Server,
			conn.Instance,
			qmStatus,
			premiumStatus,
		})
	}

	t.Render()
	fmt.Println()
}

// showContextHelp displays detailed help for a specific command
func showContextHelp(command string) {
	output.Header("Справка по команде: %s", command)
	fmt.Println()

	switch command {
	case "instance", "i":
		fmt.Println("КОМАНДА: instance")
		fmt.Println("ОПИСАНИЕ: Управление инстансами Minecraft")
		fmt.Println()
		fmt.Println("ПОДКОМАНДЫ:")
		fmt.Println("  instance list              ", "Показать список всех инстансов")
		fmt.Println("  instance create <name>     ", "Создать новый инстанс")
		fmt.Println("  instance start <name>      ", "Запустить инстанс")
		fmt.Println("  instance edit <name>       ", "Редактировать настройки инстанса")
		fmt.Println("  instance delete <name>     ", "Удалить инстанс")
		fmt.Println()
		fmt.Println("ПРИМЕРЫ:")
		fmt.Println("  instance list")
		fmt.Println("  instance create myworld")
		fmt.Println("  instance start myworld --server play.example.com:25565")

	case "servers":
		fmt.Println("КОМАНДА: servers")
		fmt.Println("ОПИСАНИЕ: Просмотр списка серверов QMServer Cloud")
		fmt.Println()
		fmt.Println("ИСПОЛЬЗОВАНИЕ:")
		fmt.Println("  servers                    ", "Показать все серверы")
		fmt.Println("  servers --filter premium   ", "Показать только Premium серверы")
		fmt.Println("  servers --search 'RPG'     ", "Поиск серверов по названию")
		fmt.Println("  servers --limit 5          ", "Ограничить количество серверов")
		fmt.Println()
		fmt.Println("ИНФОРМАЦИЯ:")
		fmt.Println("  Отображает таблицу с серверами QMServer Cloud")
		fmt.Println("  Серверы сортируются по приоритету: Premium -> Обычные")
		fmt.Println("  Показывается статус Premium и информация о модлоадере")

	case "monitor":
		fmt.Println("КОМАНДА: monitor")
		fmt.Println("ОПИСАНИЕ: Мониторинг запущенных инстансов Minecraft")
		fmt.Println()
		fmt.Println("ПОДКОМАНДЫ:")
		fmt.Println("  monitor list               ", "Показать активные мониторинги")
		fmt.Println("  monitor start <instance>   ", "Начать мониторинг инстанса")
		fmt.Println("  monitor stop <instance>    ", "Остановить мониторинг инстанса")
		fmt.Println("  monitor clear              ", "Остановить все мониторинги")
		fmt.Println()
		fmt.Println("ПРИМЕРЫ:")
		fmt.Println("  monitor list")
		fmt.Println("  monitor start myinstance")
		fmt.Println("  monitor stop myinstance")
		fmt.Println()
		fmt.Println("ИНФОРМАЦИЯ:")
		fmt.Println("  Мониторинг показывает статус запущенных инстансов")
		fmt.Println("  Работает только в интерактивном режиме")

	case "config":
		fmt.Println("КОМАНДА: config")
		fmt.Println("ОПИСАНИЕ: Управление конфигурацией интерактивного режима")
		fmt.Println()
		fmt.Println("ПОДКОМАНДЫ:")
		fmt.Println("  config list                    ", "Показать всю конфигурацию")
		fmt.Println("  config get <key>               ", "Получить значение параметра")
		fmt.Println("  config set <key>=<value>       ", "Установить значение параметра")
		fmt.Println("  config reset                   ", "Сбросить к значениям по умолчанию")
		fmt.Println("  config export <file>           ", "Экспортировать конфигурацию")
		fmt.Println("  config import <file>           ", "Импортировать конфигурацию")
		fmt.Println()
		fmt.Println("ДОСТУПНЫЕ ПАРАМЕТРЫ:")
		fmt.Println("  theme             ", "Тема оформления (default, dark, light)")
		fmt.Println("  autocomplete      ", "Включить автодополнение (true/false)")
		fmt.Println("  show_status_bar   ", "Показывать статусную строку (true/false)")
		fmt.Println("  debug_mode        ", "Режим отладки (true/false)")
		fmt.Println("  max_history_size  ", "Максимальный размер истории команд")
		fmt.Println("  progress_style    ", "Стиль прогресс-баров (default, minimal)")
		fmt.Println("  color_scheme      ", "Цветовая схема (default, monochrome)")
		fmt.Println()
		fmt.Println("ПРИМЕРЫ:")
		fmt.Println("  config list")
		fmt.Println("  config set theme=dark")
		fmt.Println("  config set autocomplete=false")
		fmt.Println("  config get debug_mode")
		fmt.Println("  config reset")

	case "batch":
		fmt.Println("КОМАНДА: batch")
		fmt.Println("ОПИСАНИЕ: Пакетные операции над несколькими инстансами")
		fmt.Println()
		fmt.Println("ПОДКОМАНДЫ:")
		fmt.Println("  batch start <instances...>      ", "Запустить несколько инстансов")
		fmt.Println("  batch update <instances...>     ", "Обновить несколько инстансов")
		fmt.Println("  batch stop <instances...>       ", "Остановить несколько инстансов")
		fmt.Println("  batch create <names...>         ", "Создать несколько инстансов")
		fmt.Println("  batch delete <instances...>     ", "Удалить несколько инстансов")
		fmt.Println()
		fmt.Println("ПАРАМЕТРЫ:")
		fmt.Println("  --server <address>              ", "Сервер для подключения (start)")
		fmt.Println("  --delay <seconds>               ", "Задержка между операциями (start)")
		fmt.Println("  --force                         ", "Принудительное выполнение")
		fmt.Println("  --version <version>             ", "Версия Minecraft (create)")
		fmt.Println("  --template <name>               ", "Шаблон инстанса (create)")
		fmt.Println()
		fmt.Println("ПРИМЕРЫ:")
		fmt.Println("  batch start inst1 inst2 inst3 --server play.example.com:25565")
		fmt.Println("  batch update inst1 inst2 --force")
		fmt.Println("  batch create server1 server2 --version 1.20.1")
		fmt.Println("  batch stop inst1 inst2 inst3 --force")
		fmt.Println("  batch delete old1 old2 --force")

	case "auth", "login":
		fmt.Println("КОМАНДА: auth")
		fmt.Println("ОПИСАНИЕ: Управление аккаунтами Minecraft")
		fmt.Println()
		fmt.Println("ПОДКОМАНДЫ:")
		fmt.Println("  auth login                 ", "Войти в аккаунт")
		fmt.Println("  auth logout                ", "Выйти из аккаунта")
		fmt.Println("  auth list                  ", "Показать сохраненные аккаунты")
		fmt.Println()
		fmt.Println("ПРИМЕРЫ:")
		fmt.Println("  auth login")
		fmt.Println("  auth list")

	case "debug":
		fmt.Println("КОМАНДА: debug")
		fmt.Println("ОПИСАНИЕ: Управление режимом отладки")
		fmt.Println()
		fmt.Println("ИСПОЛЬЗОВАНИЕ:")
		fmt.Println("  debug                      ", "Переключить режим DEBUG")
		fmt.Println("  debug on                   ", "Включить DEBUG режим")
		fmt.Println("  debug off                  ", "Отключить DEBUG режим")
		fmt.Println()
		fmt.Println("ИНФОРМАЦИЯ:")
		fmt.Println("  В DEBUG режиме логи операций отображаются в реальном времени")
		fmt.Println("  Полезно для диагностики проблем и отслеживания операций")

	case "clear", "cls":
		fmt.Println("КОМАНДА: clear")
		fmt.Println("ОПИСАНИЕ: Очистка экрана терминала")
		fmt.Println()
		fmt.Println("ИСПОЛЬЗОВАНИЕ:")
		fmt.Println("  clear                      ", "Очистить экран")
		fmt.Println("  cls                        ", "Очистить экран (алиас)")

	case "status":
		fmt.Println("КОМАНДА: status")
		fmt.Println("ОПИСАНИЕ: Показать статусную информацию")
		fmt.Println()
		fmt.Println("ИНФОРМАЦИЯ:")
		fmt.Println("  Отображает текущий статус:")
		fmt.Println("  - Подключение к QMServer Cloud")
		fmt.Println("  - Статус DEBUG режима")
		fmt.Println("  - Количество доступных инстансов")

	case "exit", "quit":
		fmt.Println("КОМАНДА: exit")
		fmt.Println("ОПИСАНИЕ: Выход из интерактивного режима")
		fmt.Println()
		fmt.Println("ИСПОЛЬЗОВАНИЕ:")
		fmt.Println("  exit                       ", "Выйти из интерактивного режима")
		fmt.Println("  quit                       ", "Выйти из интерактивного режима")
		fmt.Println("  q                          ", "Выйти из интерактивного режима")

	default:
		output.Error("Команда '%s' не найдена", command)
		fmt.Println()
		fmt.Println("Доступные команды:")
		fmt.Println("  instance, servers, auth, debug, clear, status, exit")
		fmt.Println()
		fmt.Println("Используйте 'help' для общего списка команд")
	}

	fmt.Println()
}

// Start creates the CLI parser and runs it. It returns an exit handler and code.
func Run() (func(int), int) {
	// Parse verbosity from command line before Kong parsing
	parseVerbosityFromArgs()

	// Check if we should enter interactive mode
	if shouldUseInteractiveMode() {
		return runInteractiveMode(CurrentVerbosity)
	}

	// Expand aliases first
	expandedArgs := expandAliases(os.Args[1:])

	// Check if we only have flags (no commands) - if so, show help
	if !hasCommands(expandedArgs) {
		// Set default language first
		output.SetLang(language.Russian)

		// Check for --lang flag to override default
		langFlag := parseLangFlag()
		if langFlag == "en" {
			output.SetLang(language.English)
		}

		color.New(color.Bold).Println(output.Translate("cli.title"))
		color.New(color.Underline).Println(output.Translate("cli.subtitle"))
		fmt.Println()
		fmt.Println(output.Translate("cli.usage"))
		fmt.Println(output.Translate("cli.usage.cmd"))
		fmt.Println()
		fmt.Println(output.Translate("cli.commands"))
		fmt.Println(output.Translate("cli.cmd.instance"))
		fmt.Println(output.Translate("cli.cmd.update"))
		fmt.Println(output.Translate("cli.cmd.auth"))
		fmt.Println(output.Translate("cli.cmd.search"))
		fmt.Println(output.Translate("cli.cmd.java"))
		fmt.Println("  servers         ", "Показать список серверов QMServer Cloud")
		fmt.Println(output.Translate("cli.cmd.about"))
		fmt.Println()
		fmt.Println(output.Translate("cli.aliases"))
		fmt.Println(output.Translate("cli.alias.i"))
		fmt.Println(output.Translate("cli.alias.s"))
		fmt.Println(output.Translate("cli.alias.is"))
		fmt.Println()
		fmt.Println(output.Translate("cli.help"))
		return func(int) {}, 0
	}

	// Check for --lang flag in command line arguments
	langFlag := parseLangFlag()
	if langFlag != "" {
		switch langFlag {
		case "en":
			output.SetLang(language.English)
		case "ru":
			output.SetLang(language.Russian)
		default:
			output.SetLang(language.Russian) // Default to Russian
		}
	} else {
		// Auto-detect system language
		lang, err := locale.Detect()
		if err == nil {
			output.SetLang(lang)
		} else {
			// Default to Russian if locale detection fails
			output.SetLang(language.Russian)
		}
	}

	parser := kong.Must(&CLI{},
		kong.Name(name),
		kong.Description(output.Translate("launcher.description")),
		kong.ConfigureHelp(kong.HelpOptions{
			NoExpandSubcommands: true,
			Compact:             true,
		}),
		kong.ValueFormatter(valueFormatter),
		groups(),
		vars(),
	)
	komplete.Run(parser)

	ctx, err := parser.Parse(expandedArgs)
	if err != nil {
		exitCode := 1
		var parseErr *kong.ParseError
		if errors.As(err, &parseErr) {
			// Show usage only if there are actual commands (not just flags)
			if hasCommands(expandedArgs) {
				parseErr.Context.PrintUsage(false)
				// For commands without subcommands, don't show error after usage
				if strings.Contains(err.Error(), "expected one of") {
					return parser.Exit, 0
				}
			}
			exitCode = parseErr.ExitCode()
		}
		output.Error("%s", err)
		return parser.Exit, exitCode
	}

	if err := ctx.Run(); err != nil {
		output.Error("%s", err)
		tips(err)
		var coder kong.ExitCoder
		if errors.As(err, &coder) {
			return ctx.Exit, coder.ExitCode()
		}
		return ctx.Exit, 1
	}
	return ctx.Exit, 0
}
