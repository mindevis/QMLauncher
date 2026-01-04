package output

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
)

// Info prints an general informational message.
func Info(format string, a ...any) {
	color.New(color.Bold, color.FgBlue).Print("| ")
	fmt.Printf(format+"\n", a...)
}

// Success prints a success information message.
//
// Indicates a command or task has successfully completed.
func Success(format string, a ...any) {
	color.New(color.Bold, color.FgGreen).Print("| ")
	fmt.Printf(format+"\n", a...)
}

// Warning prints a cautionary message.
//
// Indicates that there may be an issue.
func Warning(format string, a ...any) {
	color.New(color.Bold, color.FgYellow).Printf("| %s: ", Translate("launcher.warning"))
	fmt.Printf(format+"\n", a...)
}

// Debug prints a debug message.
//
// Used to print information messages useful for debugging the launcher.
func Debug(format string, a ...any) {
	color.New(color.Bold, color.FgMagenta).Printf("| %s: ", Translate("launcher.debug"))
	fmt.Printf(format+"\n", a...)
}

// Error prints an error message.
//
// Indicates a fatal error.
func Error(format string, a ...any) {
	color.New(color.Bold, color.FgRed).Printf("| %s: ", Translate("launcher.error"))
	fmt.Printf(format+"\n", a...)
}

// Tip prints a tip message.
//
// Indicates an action that should be performed.
func Tip(format string, a ...any) {
	color.New(color.Bold, color.FgYellow).Printf("| %s: ", Translate("launcher.tip"))
	fmt.Printf(format+"\n", a...)
}

// Progress prints a progress message.
//
// Used to indicate ongoing operations with progress indicators.
func Progress(format string, a ...any) {
	color.New(color.Bold, color.FgCyan).Print("🔄 ")
	fmt.Printf(format+"\n", a...)
}

// Notification prints a notification message.
//
// Used for system notifications and alerts.
func Notification(format string, a ...any) {
	color.New(color.Bold, color.FgCyan).Print("🔔 ")
	fmt.Printf(format+"\n", a...)
}

// Header prints a header message.
//
// Used for section headers and titles.
func Header(format string, a ...any) {
	color.New(color.Bold, color.Underline, color.FgWhite).Printf(format+"\n", a...)
}

// Status prints a status message.
//
// Used to show current state or status information.
func Status(format string, a ...any) {
	color.New(color.Faint, color.FgWhite).Printf(format+"\n", a...)
}

// Highlight prints highlighted text.
//
// Used to draw attention to important information.
func Highlight(format string, a ...any) {
	color.New(color.Bold, color.BgBlue, color.FgWhite).Printf(" %s ", fmt.Sprintf(format, a...))
	fmt.Println()
}

// ErrorHighlight prints an error with highlighting.
//
// Used for important error messages that need attention.
func ErrorHighlight(format string, a ...any) {
	color.New(color.Bold, color.BgRed, color.FgWhite).Printf(" ERROR ")
	color.New(color.Bold, color.FgRed).Printf(format+"\n", a...)
}

// SuccessHighlight prints a success message with highlighting.
//
// Used for important success confirmations.
func SuccessHighlight(format string, a ...any) {
	color.New(color.Bold, color.BgGreen, color.FgWhite).Printf(" SUCCESS ")
	color.New(color.Bold, color.FgGreen).Printf(format+"\n", a...)
}

// CreateProgressBar creates a beautiful progress bar for operations
func CreateProgressBar(total int64, description string) *progressbar.ProgressBar {
	return progressbar.NewOptions64(total,
		progressbar.OptionSetDescription(fmt.Sprintf("[cyan][%s][reset] ", description)),
		progressbar.OptionSetWriter(color.Output),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWidth(50),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionOnCompletion(func() {
			fmt.Print("\n")
		}),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "█",
			SaucerHead:    "█",
			SaucerPadding: "░",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)
}

// CreateIndeterminateBar creates a progress bar for operations with unknown duration
func CreateIndeterminateBar(description string) *progressbar.ProgressBar {
	return progressbar.NewOptions(-1,
		progressbar.OptionSetDescription(fmt.Sprintf("[cyan][%s][reset] ", description)),
		progressbar.OptionSetWriter(color.Output),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionOnCompletion(func() {
			fmt.Print("\n")
		}),
	)
}

// ShowSpinner shows a simple spinner with message
func ShowSpinner(message string) {
	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	for i := 0; i < len(spinner); i++ {
		fmt.Printf("\r%s %s", spinner[i], message)
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Print("\r" + strings.Repeat(" ", len(message)+2) + "\r")
}

// ShowProgressWithMessage shows progress with custom message format
func ShowProgressWithMessage(current, total int, message string) {
	percentage := float64(current) / float64(total) * 100
	barWidth := 40
	filled := int(float64(barWidth) * percentage / 100)

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	color.New(color.FgCyan).Printf("\r%s [", message)
	color.New(color.FgGreen).Printf("%s", bar)
	color.New(color.FgCyan).Printf("] %.1f%% (%d/%d)", percentage, current, total)
}
