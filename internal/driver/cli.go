package driver

import (
	"errors"
	"fmt"

	"github.com/shippomx/grains/internal/plugin"
)

type source struct {
	Sources   []string
	ExecName  string
	Base      []string
	Normalize bool
}

// parseFlags parses the command lines through the specified flags package
// and returns the source of the dump and optionally the command
// for the kind of report to generate (nil for interactive use).
func parseFlags(o *plugin.Options) (*source, []string, error) {
	flag := o.Flagset
	// Comparisons.
	flagBase := flag.StringList("base", "", "Source of base dump for dump subtraction")

	cfg := currentConfig()
	configFlagSetter := installConfigFlags(flag, &cfg)

	flagCommands := make(map[string]*bool)
	flagParamCommands := make(map[string]*string)
	for name, cmd := range grainsCommands {
		if cmd.hasParam {
			flagParamCommands[name] = flag.String(name, "", "Generate a report in "+name+" format, matching regexp")
		}
	}

	args := flag.Parse(func() {
		o.UI.Print(usageMsgHdr +
			usage(true) +
			usageMsgSrc +
			flag.ExtraUsage() +
			usageMsgVars)
	})
	if len(args) == 0 {
		return nil, nil, errors.New("no goroutine dump file specified")
	}

	// Apply any specified flags to cfg.
	if err := configFlagSetter(); err != nil {
		return nil, nil, err
	}

	cmd, err := outputFormat(flagCommands, flagParamCommands)
	if err != nil {
		return nil, nil, err
	}

	source := &source{
		Sources: args,
	}

	if err := source.addBaseDumps(*flagBase); err != nil {
		return nil, nil, err
	}

	setCurrentConfig(cfg)
	return source, cmd, nil
}

// addBaseDumps adds the list of base dumps or diff base dumps to
// the source. This function will return an error if both base and diff base
// dumps are specified.
func (source *source) addBaseDumps(flagBase []*string) error {
	source.Base = dropEmpty(flagBase)
	return nil
}

// dropEmpty list takes a slice of string pointers, and outputs a slice of
// non-empty strings associated with the flag.
func dropEmpty(list []*string) []string {
	var l []string
	for _, s := range list {
		if *s != "" {
			l = append(l, *s)
		}
	}
	return l
}

// installConfigFlags creates command line flags for configuration
// fields and returns a function which can be called after flags have
// been parsed to copy any flags specified on the command line to
// *cfg.
func installConfigFlags(flag plugin.FlagSet, cfg *config) func() error {
	// List of functions for setting the different parts of a config.
	var setters []func()
	var err error // Holds any errors encountered while running setters.

	for _, field := range configFields {
		n := field.name
		help := configHelp[n]
		var setter func()
		switch ptr := cfg.fieldPtr(field).(type) {
		case *bool:
			f := flag.Bool(n, *ptr, help)
			setter = func() { *ptr = *f }
		case *int:
			f := flag.Int(n, *ptr, help)
			setter = func() { *ptr = *f }
		case *float64:
			f := flag.Float64(n, *ptr, help)
			setter = func() { *ptr = *f }
		case *string:
			if len(field.choices) == 0 {
				f := flag.String(n, *ptr, help)
				setter = func() { *ptr = *f }
			} else {
				// Make a separate flag per possible choice.
				// Set all flags to initially false so we can
				// identify conflicts.
				bools := make(map[string]*bool)
				for _, choice := range field.choices {
					bools[choice] = flag.Bool(choice, false, configHelp[choice])
				}
				setter = func() {
					var set []string
					for k, v := range bools {
						if *v {
							set = append(set, k)
						}
					}
					switch len(set) {
					case 0:
						// Leave as default value.
					case 1:
						*ptr = set[0]
					default:
						err = fmt.Errorf("conflicting options set: %v", set)
					}
				}
			}
		}
		setters = append(setters, setter)
	}

	return func() error {
		// Apply the setter for every flag.
		for _, setter := range setters {
			setter()
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func outputFormat(bcmd map[string]*bool, acmd map[string]*string) (cmd []string, err error) {
	for n, b := range bcmd {
		if *b {
			if cmd != nil {
				return nil, errors.New("must set at most one output format")
			}
			cmd = []string{n}
		}
	}
	for n, s := range acmd {
		if *s != "" {
			if cmd != nil {
				return nil, errors.New("must set at most one output format")
			}
			cmd = []string{n, *s}
		}
	}
	return cmd, nil
}

var usageMsgHdr = `usage:

Produce output in the specified format.

   grains <format> [options] [binary] <source> ...

Details:
`
var usageMsgSrc = "\n\n" +
	"  Source options:\n" +
	"    -base source       Source of base dump for dump subtraction\n" +
	"    dockerd.tar.gz		Dump in compressed protobuf format\n" +
	"    dockerd.dlog		Dump in string format\n"

var usageMsgVars = "\n\n" +
	"  Environment Variables:\n" +
	"   PPROF_TMPDIR       Location for saved dumps (default $HOME/grains)\n" +
	"   PPROF_BINARY_PATH  Search path for local binary files\n" +
	"                      default: $HOME/grains/binaries\n" +
	"                      searches $name, $path, $buildid/$name, $path/$buildid\n"
