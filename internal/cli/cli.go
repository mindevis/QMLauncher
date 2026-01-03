package cli

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"

	"QMLauncher/internal/cli/cmd"
	"QMLauncher/internal/cli/output"
	"QMLauncher/internal/meta"
	"QMLauncher/internal/network"
	env "QMLauncher/pkg"
	"QMLauncher/pkg/auth"

	"github.com/Xuanwo/go-locale"
	"github.com/alecthomas/kong"
	"github.com/fatih/color"
	"go.abhg.dev/komplete"
	"golang.org/x/text/language"
)

const (
	name    = "QMLauncher"
	version = "1.1.0"
)

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
	// Check for explicit interactive flag first
	if hasInteractiveFlag() {
		return true
	}

	// Use interactive mode on Windows by default
	// On Unix-like systems, show help by default unless explicitly requested
	if runtime.GOOS == "windows" {
		return true
	}
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

// CommandHistory manages command history for interactive mode
type CommandHistory struct {
	commands []string
	current  int
	maxSize  int
}

// NewCommandHistory creates a new command history
func NewCommandHistory(maxSize int) *CommandHistory {
	return &CommandHistory{
		commands: make([]string, 0, maxSize),
		current:  -1,
		maxSize:  maxSize,
	}
}

// Add adds a command to history
func (h *CommandHistory) Add(cmd string) {
	if cmd == "" {
		return
	}

	// Don't add duplicates of the last command
	if len(h.commands) > 0 && h.commands[len(h.commands)-1] == cmd {
		return
	}

	h.commands = append(h.commands, cmd)
	if len(h.commands) > h.maxSize {
		h.commands = h.commands[1:]
	}
	h.current = len(h.commands)
}

// Previous returns the previous command
func (h *CommandHistory) Previous() string {
	if len(h.commands) == 0 {
		return ""
	}
	h.current--
	if h.current < 0 {
		h.current = 0
	}
	if h.current < len(h.commands) {
		return h.commands[h.current]
	}
	return ""
}

// Next returns the next command
func (h *CommandHistory) Next() string {
	if len(h.commands) == 0 {
		return ""
	}
	h.current++
	if h.current >= len(h.commands) {
		h.current = len(h.commands)
		return ""
	}
	return h.commands[h.current]
}

// Reset resets the current position
func (h *CommandHistory) Reset() {
	h.current = len(h.commands)
}

// GetHistory returns all commands in history
func (h *CommandHistory) GetHistory() []string {
	return h.commands
}

// runInteractiveMode starts the interactive command shell
func runInteractiveMode() (func(int), int) {
	// Set default language for interactive mode
	output.SetLang(language.Russian)

	color.New(color.Bold).Println(output.Translate("cli.title"))
	color.New(color.Underline).Println(output.Translate("cli.subtitle"))
	fmt.Println()
	fmt.Println(output.Translate("interactive.welcome"))
	fmt.Println(output.Translate("interactive.help"))
	fmt.Println()

	history := NewCommandHistory(100)
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print(output.Translate("interactive.prompt"))

		line, err := readLineWithHistory(reader, history)
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Add to history
		history.Add(line)

		// Handle interactive commands first (they take priority)
		if line == "exit" || line == "quit" || line == "q" {
			fmt.Println(output.Translate("interactive.goodbye"))
			break
		}

		if line == "help" || line == "h" || line == "?" {
			showInteractiveHelp()
			continue
		}

		if line == "history" {
			showCommandHistory(history)
			continue
		}

		if line == "clear" {
			history = NewCommandHistory(100)
			fmt.Println(output.Translate("interactive.history.cleared"))
			continue
		}

		// Parse and execute launcher command
		args := strings.Fields(line)
		if len(args) == 0 {
			continue
		}

		// Prepend program name for Kong
		fullArgs := append([]string{os.Args[0]}, expandAliases(args)...)

		// Save original args and replace with command args
		origArgs := os.Args
		os.Args = fullArgs

		// Execute command
		_, code := executeCommand()

		// Restore original args
		os.Args = origArgs

		if code != 0 {
			fmt.Printf("%s %d\n", output.Translate("interactive.error"), code)
		}

		fmt.Println()
	}

	return func(int) {}, 0
}

// readLineWithHistory reads a line with history navigation support
func readLineWithHistory(reader *bufio.Reader, history *CommandHistory) (string, error) {
	var buffer []rune
	cursor := 0

	for {
		char, _, err := reader.ReadRune()
		if err != nil {
			return "", err
		}

		switch char {
		case '\n', '\r':
			fmt.Println()
			return string(buffer), nil
		case '\b', 127: // Backspace
			if cursor > 0 {
				buffer = append(buffer[:cursor-1], buffer[cursor:]...)
				cursor--
				// Clear line and reprint
				fmt.Print("\r\033[K" + output.Translate("interactive.prompt") + string(buffer))
				fmt.Printf("\033[%dG", len(output.Translate("interactive.prompt"))+cursor+1)
			}
		case 27: // Escape sequence (arrow keys)
			seq, _, err := reader.ReadRune()
			if err != nil {
				continue
			}
			if seq != '[' {
				continue
			}
			dir, _, err := reader.ReadRune()
			if err != nil {
				continue
			}

			switch dir {
			case 'A': // Up arrow
				cmd := history.Previous()
				if cmd != "" {
					buffer = []rune(cmd)
					cursor = len(buffer)
					fmt.Print("\r\033[K" + output.Translate("interactive.prompt") + string(buffer))
				}
			case 'B': // Down arrow
				cmd := history.Next()
				buffer = []rune(cmd)
				cursor = len(buffer)
				fmt.Print("\r\033[K" + output.Translate("interactive.prompt") + string(buffer))
			}
		default:
			buffer = append(buffer[:cursor], append([]rune{char}, buffer[cursor:]...)...)
			cursor++
			fmt.Print("\r\033[K" + output.Translate("interactive.prompt") + string(buffer))
			fmt.Printf("\033[%dG", len(output.Translate("interactive.prompt"))+cursor+1)
		}
	}
}

// showCommandHistory displays the command history
func showCommandHistory(history *CommandHistory) {
	commands := history.GetHistory()
	if len(commands) == 0 {
		fmt.Println(output.Translate("interactive.history.empty"))
		return
	}

	fmt.Println(output.Translate("interactive.history.title"))
	for i, cmd := range commands {
		fmt.Printf(" %3d  %s\n", i+1, cmd)
	}
}

// executeCommand parses and executes a single command
func executeCommand() (func(int), int) {
	parser := kong.Must(&CLI{},
		kong.UsageOnError(),
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
			parseErr.Context.PrintUsage(false)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return parser.Exit, 1
	}

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
	fmt.Println("  history        ", output.Translate("interactive.help.cmd.history"))
	fmt.Println("  clear          ", output.Translate("interactive.help.cmd.clear"))
	fmt.Println("  <command>      ", output.Translate("interactive.help.cmd.command"))
	fmt.Println()
	fmt.Println(output.Translate("interactive.help.aliases"))
	fmt.Println("  -i             ", output.Translate("interactive.alias.i"))
	fmt.Println("  -s             ", output.Translate("interactive.alias.s"))
	fmt.Println("  -is            ", output.Translate("interactive.alias.is"))
	fmt.Println()
}

// Start creates the CLI parser and runs it. It returns an exit handler and code.
func Run() (func(int), int) {
	// Expand aliases first
	expandedArgs := expandAliases(os.Args[1:])

	// Check if we only have flags (no commands) - if so, show help or enter interactive mode
	if !hasCommands(expandedArgs) {
		// Set default language first
		output.SetLang(language.Russian)

		// Check for --lang flag to override default
		langFlag := parseLangFlag()
		if langFlag == "en" {
			output.SetLang(language.English)
		}

		// Check if we should enter interactive mode
		if shouldUseInteractiveMode() {
			return runInteractiveMode()
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
		// Debug: check if translations work
		// fmt.Printf("DEBUG: cli.aliases = '%s'\n", output.Translate("cli.aliases"))
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
		kong.UsageOnError(),
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
