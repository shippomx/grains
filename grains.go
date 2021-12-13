package main

import (
	"fmt"
	"github.com/shippomx/grains/driver"
	"os"
	"strings"
	"syscall"

	"github.com/chzyer/readline"
)

func main() {
	if err := driver.Grains(&driver.Options{UI: newUI()}); err != nil {
		fmt.Fprintf(os.Stderr, "grains: %v\n", err)
		os.Exit(2)
	}
}

// readlineUI implements the driver.UI interface using the
// github.com/chzyer/readline library.
// This is contained in grains.go to avoid adding the readline
// dependency in the vendored copy of grains in the Go distribution,
// which does not use this file.
type readlineUI struct {
	rl *readline.Instance
}

func newUI() driver.UI {
	rl, err := readline.New("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "readline: %v", err)
		return nil
	}
	return &readlineUI{
		rl: rl,
	}
}

// ReadLine returns a line of text (a command) read from the user.
// prompt is printed before reading the command.
func (r *readlineUI) ReadLine(prompt string) (string, error) {
	r.rl.SetPrompt(prompt)
	return r.rl.Readline()
}

// Print shows a message to the user.
// It is printed over stderr as stdout is reserved for regular output.
func (r *readlineUI) Print(args ...interface{}) {
	text := fmt.Sprint(args...)
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	fmt.Fprint(r.rl.Stderr(), text)
}

// PrintErr shows a message to the user, colored in red for emphasis.
// It is printed over stderr as stdout is reserved for regular output.
func (r *readlineUI) PrintErr(args ...interface{}) {
	text := fmt.Sprint(args...)
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if readline.IsTerminal(int(syscall.Stderr)) {
		text = colorize(text)
	}
	fmt.Fprint(r.rl.Stderr(), text)
}

// colorize the msg using ANSI color escapes.
func colorize(msg string) string {
	var red = 31
	var colorEscape = fmt.Sprintf("\033[0;%dm", red)
	var colorResetEscape = "\033[0m"
	return colorEscape + msg + colorResetEscape
}

// IsTerminal returns whether the UI is known to be tied to an
// interactive terminal (as opposed to being redirected to a file).
func (r *readlineUI) IsTerminal() bool {
	return readline.IsTerminal(int(syscall.Stdout))
}

// WantBrowser starts a browser on interactive mode.
func (r *readlineUI) WantBrowser() bool {
	return r.IsTerminal()
}

// SetAutoComplete instructs the UI to call complete(cmd) to obtain
// the auto-completion of cmd, if the UI supports auto-completion at all.
func (r *readlineUI) SetAutoComplete(complete func(string) string) {
	// TODO: Implement auto-completion support.
}
