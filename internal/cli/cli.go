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
	Start       cmd.StartCmd     `cmd:"" help:"${start}"`
	Instance    cmd.InstanceCmd  `cmd:"" help:"${instance}" aliases:"inst"`
	Update      cmd.UpdateCmd    `cmd:"" help:"${update}"`
	Auth        cmd.AuthCmd      `cmd:"" help:"${auth}"`
	Search      cmd.SearchCmd    `cmd:"" help:"${search}"`
	Java        cmd.JavaCmd      `cmd:"" help:"${java}"`
	Completions komplete.Command `cmd:"" help:"${completions}"`
	About       aboutCmd         `cmd:"" help:"${about}"`

	Verbosity string `help:"${arg_verbosity}" enum:"info,extra,debug" default:"info"`
	Dir       string `help:"${arg_dir}" type:"path" placeholder:"PATH"`
	NoColor   bool   `help:"${arg_nocolor}"`
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

// Start creates the CLI parser and runs it. It returns an exit handler and code.
func Run() (func(int), int) {
	lang, err := locale.Detect()
	if err == nil {
		output.SetLang(lang)
	} else {
		// Default to Russian if locale detection fails
		output.SetLang(language.Russian)
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

	// If no arguments provided, show help and exit
	if len(os.Args) == 1 {
		color.New(color.Bold).Println("QMLauncher CLI")
		color.New(color.Underline).Println("Minecraft launcher with mod support")
		fmt.Println()
		fmt.Println("USAGE:")
		fmt.Println("  QMLauncher [command]")
		fmt.Println()
		fmt.Println("AVAILABLE COMMANDS:")
		fmt.Println("  start       Start Minecraft with specified options")
		fmt.Println("  instance    Manage Minecraft instances")
		fmt.Println("  update      Update the launcher")
		fmt.Println("  auth        Manage authentication")
		fmt.Println("  search      Search for Minecraft versions")
		fmt.Println("  java        Manage Java installations")
		fmt.Println("  about       Show version information")
		fmt.Println()
		fmt.Println("Use 'QMLauncher [command] --help' for more information about a command.")
		return func(int) {}, 0
	}

	ctx, err := parser.Parse(os.Args[1:])
	if err != nil {
		exitCode := 1
		var parseErr *kong.ParseError
		if errors.As(err, &parseErr) {
			parseErr.Context.PrintUsage(false)
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
