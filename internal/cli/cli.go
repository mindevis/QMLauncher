package cli

import (
	"errors"
	"fmt"
	"net"
	"os"
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

	Verbosity string `help:"${arg_verbosity}" enum:"info,extra,debug" default:"info"`
	Dir       string `help:"${arg_dir}" type:"path" placeholder:"PATH"`
	NoColor   bool   `help:"${arg_nocolor}"`
	Lang      string `help:"${arg_lang}" default:"ru"`
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
	args := os.Args[1:] // Skip program name
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

// Start creates the CLI parser and runs it. It returns an exit handler and code.
func Run() (func(int), int) {
	// Check if we only have flags (no commands) - if so, show help
	args := os.Args[1:]
	if !hasCommands(args) {
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

	ctx, err := parser.Parse(os.Args[1:])
	if err != nil {
		exitCode := 1
		var parseErr *kong.ParseError
		if errors.As(err, &parseErr) {
			// Show usage only if there are actual commands (not just flags)
			if hasCommands(os.Args[1:]) {
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
