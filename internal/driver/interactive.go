package driver

import (
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/shippomx/grains/dump"
	"github.com/shippomx/grains/internal/plugin"
)

var commentStart = "//:" // Sentinel for comments on options
var tailDigitsRE = regexp.MustCompile("[0-9]+$")

// interactive starts a shell to read grains commands.
func interactive(p *dump.Dump, o *plugin.Options) error {
	// Do not wait for the visualizer to complete, to allow multiple
	// graphs to be visualized simultaneously.

	shortcuts := shortcuts{
		":": []string{"list="},
	}
	greetings(p, o.UI)
	for {
		input, err := o.UI.ReadLine("(grains) ")
		if err != nil {
			if err != io.EOF {
				return err
			}
			if input == "" {
				return nil
			}
		}

		for _, input := range shortcuts.expand(input) {
			// Process assignments of the form variable=value
			if s := strings.SplitN(input, "=", 2); len(s) > 0 {
				//name := strings.TrimSpace(s[0])
				var value string
				if len(s) == 2 {
					value = s[1]
					if comment := strings.LastIndex(value, commentStart); comment != -1 {
						value = value[:comment]
					}
					value = strings.TrimSpace(value)
				}
				//if isConfigurable(name) {
				//	// All non-bool options require inputs
				//	if len(s) == 1 && !isBoolConfig(name) {
				//		o.UI.PrintErr(fmt.Errorf("please specify a value, e.g. %s=<val>", name))
				//		continue
				//	}
				//	if err := configure(name, value); err != nil {
				//		o.UI.PrintErr(err)
				//	}
				//	continue
				//}
			}

			tokens := strings.Fields(input)
			if len(tokens) == 0 {
				continue
			}

			switch tokens[0] {
			case "o", "options":
				printCurrentOptions(p, o.UI)
				continue
			case "exit", "quit", "q":
				return nil
			case "help":
				commandHelp(strings.Join(tokens[1:], " "), o.UI)
				continue
			}

			args, cfg, err := parseCommandLine(tokens)
			if err == nil {
				err = generateReportWrapper(p, args, cfg, o)
			}

			if err != nil {
				o.UI.PrintErr(err)
			}
		}
	}
}

var generateReportWrapper = generateReport // For testing purposes.

// greetings prints a brief welcome and some overall dump
// information before accepting interactive commands.
func greetings(p *dump.Dump, ui plugin.UI) {
	//numLabelUnits := identifyNumLabelUnits(p, ui)
	//ropt, err := reportOptions(p, numLabelUnits, currentConfig())
	//if err == nil {
	//	rpt := report.New(p, ropt)
	//}
	ui.Print(`Entering interactive mode (type "help" for commands, "o" for options)`)
}

// shortcuts represents composite commands that expand into a sequence
// of other commands.
type shortcuts map[string][]string

func (a shortcuts) expand(input string) []string {
	input = strings.TrimSpace(input)
	if a != nil {
		if r, ok := a[input]; ok {
			return r
		}
	}
	return []string{input}
}

func printCurrentOptions(p *dump.Dump, ui plugin.UI) {
	var args []string
	current := currentConfig()
	for _, f := range configFields {
		n := f.name
		v := current.get(f)
		comment := ""
		switch {
		case len(f.choices) > 0:
			values := append([]string{}, f.choices...)
			sort.Strings(values)
			comment = "[" + strings.Join(values, " | ") + "]"
		case n == "source_path":
			continue
		case v == "":
			// Add quotes for empty values.
			v = `""`
		}
		if comment != "" {
			comment = commentStart + " " + comment
		}
		args = append(args, fmt.Sprintf("  %-25s = %-20s %s", n, v, comment))
	}
	sort.Strings(args)
	ui.Print(strings.Join(args, "\n"))
}

// parseCommandLine parses a command and returns the grains command to
// execute and the configuration to use for the report.
func parseCommandLine(input []string) ([]string, config, error) {
	cmd, args := input[:1], input[1:]
	name := cmd[0]

	c := grainsCommands[name]
	if c == nil {
		// Attempt splitting digits on abbreviated commands (eg top10)
		if d := tailDigitsRE.FindString(name); d != "" && d != name {
			name = name[:len(name)-len(d)]
			cmd[0], args = name, append([]string{d}, args...)
			c = grainsCommands[name]
		}
	}
	if c == nil {
		if _, ok := configHelp[name]; ok {
			value := "<val>"
			if len(args) > 0 {
				value = args[0]
			}
			return nil, config{}, fmt.Errorf("did you mean: %s=%s", name, value)
		}
		return nil, config{}, fmt.Errorf("unrecognized command: %q", name)
	}

	if c.hasParam {
		if len(args) == 0 {
			return nil, config{}, fmt.Errorf("command %s requires an argument", name)
		}
		cmd = append(cmd, args[0])
		args = args[1:]
	}

	// Copy config since options set in the command line should not persist.
	vcopy := currentConfig()

	var focus, ignore string
	for i := 0; i < len(args); i++ {
		t := args[i]
		switch t[0] {
		case '>':
			outputFile := t[1:]
			if outputFile == "" {
				i++
				if i >= len(args) {
					return nil, config{}, fmt.Errorf("unexpected end of line after >")
				}
				outputFile = args[i]
			}
			vcopy.Output = outputFile
		case '-':
			if t == "--cum" || t == "-cum" {
				continue
			}
			ignore = catRegex(ignore, t[1:])
		default:
			focus = catRegex(focus, t)
		}
	}

	return cmd, vcopy, nil
}

func catRegex(a, b string) string {
	if a != "" && b != "" {
		return a + "|" + b
	}
	return a + b
}

// commandHelp displays help and usage information for all Commands
// and Variables or a specific Command or Variable.
func commandHelp(args string, ui plugin.UI) {
	if args == "" {
		help := usage(false)
		help = help + `
  :   Clear focus/ignore/hide/tagfocus/tagignore

  type "help <cmd|option>" for more information
`

		ui.Print(help)
		return
	}

	if c := grainsCommands[args]; c != nil {
		ui.Print(c.help(args))
		return
	}

	if help, ok := configHelp[args]; ok {
		ui.Print(help + "\n")
		return
	}

	ui.PrintErr("Unknown command: " + args)
}

// newCompleter creates an autocompletion function for a set of commands.
func newCompleter(fns []string) func(string) string {
	return func(line string) string {
		switch tokens := strings.Fields(line); len(tokens) {
		case 0:
			// Nothing to complete
		case 1:
			// Single token -- complete command name
			if match := matchVariableOrCommand(tokens[0]); match != "" {
				return match
			}
		case 2:
			if tokens[0] == "help" {
				if match := matchVariableOrCommand(tokens[1]); match != "" {
					return tokens[0] + " " + match
				}
				return line
			}
			fallthrough
		default:
			// Multiple tokens -- complete using functions, except for tags
			if cmd := grainsCommands[tokens[0]]; cmd != nil && tokens[0] != "tags" {
				lastTokenIdx := len(tokens) - 1
				lastToken := tokens[lastTokenIdx]
				if strings.HasPrefix(lastToken, "-") {
					lastToken = "-" + functionCompleter(lastToken[1:], fns)
				} else {
					lastToken = functionCompleter(lastToken, fns)
				}
				return strings.Join(append(tokens[:lastTokenIdx], lastToken), " ")
			}
		}
		return line
	}
}

// matchVariableOrCommand attempts to match a string token to the prefix of a Command.
func matchVariableOrCommand(token string) string {
	token = strings.ToLower(token)
	var matches []string
	for cmd := range grainsCommands {
		if strings.HasPrefix(cmd, token) {
			matches = append(matches, cmd)
		}
	}
	matches = append(matches, completeConfig(token)...)
	if len(matches) == 1 {
		return matches[0]
	}
	return ""
}

// functionCompleter replaces provided substring with a function
// name retrieved from a dump if a single match exists. Otherwise,
// it returns unchanged substring. It defaults to no-op if the dump
// is not specified.
func functionCompleter(substring string, fns []string) string {
	found := ""
	for _, fName := range fns {
		if strings.Contains(fName, substring) {
			if found != "" {
				return substring
			}
			found = fName
		}
	}
	if found != "" {
		return found
	}
	return substring
}
