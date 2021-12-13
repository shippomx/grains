package driver

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/shippomx/grains/internal/plugin"
)

// setDefaults returns a new plugin.Options with zero fields sets to
// sensible defaults.
func setDefaults(o *plugin.Options) *plugin.Options {
	d := &plugin.Options{}
	if o != nil {
		*d = *o
	}
	if d.Writer == nil {
		d.Writer = oswriter{}
	}
	if d.Flagset == nil {
		d.Flagset = &GoFlags{}
	}
	if d.UI == nil {
		d.UI = &stdUI{r: bufio.NewReader(os.Stdin)}
	}
	return d
}

type stdUI struct {
	r *bufio.Reader
}

func (ui *stdUI) ReadLine(prompt string) (string, error) {
	os.Stdout.WriteString(prompt)
	return ui.r.ReadString('\n')
}

func (ui *stdUI) Print(args ...interface{}) {
	ui.fprint(os.Stderr, args)
}

func (ui *stdUI) PrintErr(args ...interface{}) {
	ui.fprint(os.Stderr, args)
}

func (ui *stdUI) IsTerminal() bool {
	return false
}

func (ui *stdUI) WantBrowser() bool {
	return true
}

func (ui *stdUI) SetAutoComplete(func(string) string) {
}

func (ui *stdUI) fprint(f *os.File, args []interface{}) {
	text := fmt.Sprint(args...)
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	f.WriteString(text)
}

// oswriter implements the Writer interface using a regular file.
type oswriter struct{}

func (oswriter) Open(name string) (io.WriteCloser, error) {
	f, err := os.Create(name)
	return f, err
}
