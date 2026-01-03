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
	version = "1.2.0"
)

var CurrentVerbosity int

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

// runInteractiveMode starts the interactive command shell
func runInteractiveMode(verbosity int) (func(int), int) {
	CurrentVerbosity = verbosity
	// Set default language for interactive mode
	output.SetLang(language.Russian)

	color.New(color.Bold).Println(output.Translate("cli.title"))
	color.New(color.Underline).Println(output.Translate("cli.subtitle"))
	fmt.Println()
	fmt.Println(output.Translate("interactive.welcome"))
	fmt.Println(output.Translate("interactive.help"))
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
		_, code := executeCommand(CurrentVerbosity)

		// Restore original args
		os.Args = origArgs

		if code != 0 {
			fmt.Printf("%s %d\n", output.Translate("interactive.error"), code)
		}

		fmt.Println()
	}

	return func(int) {}, 0
}

// readLine reads a line of input with history support
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
		case '\b', 127: // Backspace
			if cursor > 0 {
				buffer = append(buffer[:cursor-1], buffer[cursor:]...)
				cursor--
				// Clear line and reprint
				fmt.Print("\r\033[K" + output.Translate("interactive.prompt") + string(buffer))
				fmt.Printf("\033[%dG", len(output.Translate("interactive.prompt"))+cursor+1)
			}
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
	fmt.Println("  exit, quit, q  ", output.Translate("interactive.help.cmd.exit"))
	fmt.Println("  <command>      ", output.Translate("interactive.help.cmd.command"))
	fmt.Println("  <number>       ", "Быстрый запуск по номеру из таблицы недавних подключений")
	fmt.Println()
	fmt.Println("Навигация по истории команд:")
	fmt.Println("  ↑ (стрелка вверх)   ", "Предыдущая команда")
	fmt.Println("  ↓ (стрелка вниз)    ", "Следующая команда")
	fmt.Println()
	fmt.Println(output.Translate("cli.commands"))
	fmt.Println(output.Translate("cli.cmd.instance"))
	fmt.Println(output.Translate("cli.cmd.update"))
	fmt.Println(output.Translate("cli.cmd.auth"))
	fmt.Println(output.Translate("cli.cmd.search"))
	fmt.Println(output.Translate("cli.cmd.java"))
	fmt.Println(output.Translate("cli.cmd.about"))
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
