package driver

import (
	"github.com/shippomx/grains/dump"
	internaldriver "github.com/shippomx/grains/internal/driver"
	"github.com/shippomx/grains/internal/plugin"
	"io"
	"time"
)

// Grains acquires a dump, and symbolizes it using a dump
// manager. Then it generates a report formatted according to the
// options selected through the flags package.
func Grains(o *Options) error {
	return internaldriver.Grains(o.internalOptions())
}

func (o *Options) internalOptions() *plugin.Options {
	return &plugin.Options{
		Writer:        o.Writer,
		Flagset:       o.Flagset,
		Fetch:         o.Fetch,
		UI:            o.UI,
	}
}

// Options groups all the optional plugins into grains.
type Options struct {
	Writer        Writer
	Flagset       FlagSet
	Fetch         Fetcher
	Sym           Symbolizer
	UI            UI
}

// Writer provides a mechanism to write data under a certain name,
// typically a filename.
type Writer interface {
	Open(name string) (io.WriteCloser, error)
}

// A FlagSet creates and parses command-line flags.
// It is similar to the standard flag.FlagSet.
type FlagSet interface {
	// Bool, Int, Float64, and String define new flags,
	// like the functions of the same name in package flag.
	Bool(name string, def bool, usage string) *bool
	Int(name string, def int, usage string) *int
	Float64(name string, def float64, usage string) *float64
	String(name string, def string, usage string) *string

	// StringList is similar to String but allows multiple values for a
	// single flag
	StringList(name string, def string, usage string) *[]*string

	// ExtraUsage returns any additional text that should be printed after the
	// standard usage message. The extra usage message returned includes all text
	// added with AddExtraUsage().
	// The typical use of ExtraUsage is to show any custom flags defined by the
	// specific grains plugins being used.
	ExtraUsage() string

	// AddExtraUsage appends additional text to the end of the extra usage message.
	AddExtraUsage(eu string)

	// Parse initializes the flags with their values for this run
	// and returns the non-flag command line arguments.
	// If an unknown flag is encountered or there are no arguments,
	// Parse should call usage and return nil.
	Parse(usage func()) []string
}

// A Fetcher reads and returns the dump named by src, using
// the specified duration and timeout. It returns the fetched
// dump and a string indicating a URL from where the dump
// was fetched, which may be different than src.
type Fetcher interface {
	Fetch(src string, duration, timeout time.Duration) (*dump.Dump, string, error)
}

// A Symbolizer introduces symbol information into a dump.
type Symbolizer interface {
	Symbolize(mode string, srcs MappingSources, prof *dump.Dump) error
}

// MappingSources map each dump.Mapping to the source of the dump.
// The key is either Mapping.File or Mapping.BuildId.
type MappingSources map[string][]struct {
	Source string // URL of the source the mapping was collected from
	Start  uint64 // delta applied to addresses from this source (to represent Merge adjustments)
}

// A UI manages user interactions.
type UI interface {
	// Read returns a line of text (a command) read from the user.
	// prompt is printed before reading the command.
	ReadLine(prompt string) (string, error)

	// Print shows a message to the user.
	// It formats the text as fmt.Print would and adds a final \n if not already present.
	// For line-based UI, Print writes to standard error.
	// (Standard output is reserved for report data.)
	Print(...interface{})

	// PrintErr shows an error message to the user.
	// It formats the text as fmt.Print would and adds a final \n if not already present.
	// For line-based UI, PrintErr writes to standard error.
	PrintErr(...interface{})

	// IsTerminal returns whether the UI is known to be tied to an
	// interactive terminal (as opposed to being redirected to a file).
	IsTerminal() bool

	// SetAutoComplete instructs the UI to call complete(cmd) to obtain
	// the auto-completion of cmd, if the UI supports auto-completion at all.
	SetAutoComplete(complete func(string) string)
}
