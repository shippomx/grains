package driver

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/shippomx/grains/internal/plugin"
	"github.com/shippomx/grains/internal/report"
)

// commands describes the commands accepted by grains.
type commands map[string]*command

// command describes the actions for a grains command. Includes a
// function for command-line completion, the report format to use
// during report generation, any postprocessing functions, and whether
// the command expects a regexp parameter (typically a function name).
type command struct {
	format      int           // report format to generate
	postProcess PostProcessor // postprocessing to run on report
	visualizer  PostProcessor // display output using some callback
	hasParam    bool          // collect a parameter from the CLI
	description string        // single-line description text saying what the command does
	usage       string        // multi-line help text saying how the command is used
}

// help returns a help string for a command.
func (c *command) help(name string) string {
	message := c.description + "\n"
	if c.usage != "" {
		message += "  Usage:\n"
		lines := strings.Split(c.usage, "\n")
		for _, line := range lines {
			message += fmt.Sprintf("    %s\n", line)
		}
	}
	return message + "\n"
}

// PostProcessor is a function that applies post-processing to the report output
type PostProcessor func(input io.Reader, output io.Writer, ui plugin.UI) error

// grainsCommands are the report generation commands recognized by grains.
var grainsCommands = commands{
	"trim":      {report.Text, nil, nil, false, "Trim the dump", reportHelp("trim", true, true)},
	"summary":   {report.Text, nil, nil, false, "summary the dump", reportHelp("summary", true, true)},
	"show":      {report.Text, nil, nil, true, "show the goroutine", reportHelp("show", true, true)},
}

// configHelp contains help text per configuration parameter.
var configHelp = map[string]string{
	// Filename for file-based output formats, stdout by default.
	"output": helpText("Output filename for file-based outputs"),
	"trim": helpText(
		"trim dump file more readable",
		""),
}

func helpText(s ...string) string {
	return strings.Join(s, "\n") + "\n"
}

// usage returns a string describing the grains commands and configuration
// options.  if commandLine is set, the output reflect cli usage.
func usage(commandLine bool) string {
	var prefix string
	if commandLine {
		prefix = "-"
	}
	fmtHelp := func(c, d string) string {
		return fmt.Sprintf("    %-16s %s", c, strings.SplitN(d, "\n", 2)[0])
	}

	var commands []string
	for name, cmd := range grainsCommands {
		commands = append(commands, fmtHelp(prefix+name, cmd.description))
	}
	sort.Strings(commands)

	var help string
	if commandLine {
		help = "  Output formats (select at most one):\n"
	} else {
		help = "  Commands:\n"
		commands = append(commands, fmtHelp("o/options", "List options and their current values"))
		commands = append(commands, fmtHelp("q/quit/exit/^D", "Exit grains"))
	}

	help = help + strings.Join(commands, "\n") + "\n\n" +
		"  Options:\n"

	// Print help for configuration options after sorting them.
	// Collect choices for multi-choice options print them together.
	var variables []string
	var radioStrings []string
	for _, f := range configFields {
		if len(f.choices) == 0 {
			variables = append(variables, fmtHelp(prefix+f.name, configHelp[f.name]))
			continue
		}
		// Format help for for this group.
		s := []string{fmtHelp(f.name, "")}
		for _, choice := range f.choices {
			s = append(s, "  "+fmtHelp(prefix+choice, configHelp[choice]))
		}
		radioStrings = append(radioStrings, strings.Join(s, "\n"))
	}
	sort.Strings(variables)
	sort.Strings(radioStrings)
	return help + strings.Join(variables, "\n") + "\n\n" +
		"  Option groups (only set one per group):\n" +
		strings.Join(radioStrings, "\n")
}

func reportHelp(c string, cum, redirect bool) string {
	h := []string{
		c + " [n] [focus_regex]* [-ignore_regex]*",
		"Include up to n samples",
		"Include samples matching focus_regex, and exclude ignore_regex.",
	}
	if cum {
		h[0] += " [-cum]"
		h = append(h, "-cum sorts the output by cumulative weight")
	}
	if redirect {
		h[0] += " >f"
		h = append(h, "Optionally save the report on the file f")
	}
	return strings.Join(h, "\n")
}

func listHelp(c string, redirect bool) string {
	h := []string{
		c + "<func_regex|address> [-focus_regex]* [-ignore_regex]*",
		"Include functions matching func_regex, or including the address specified.",
		"Include samples matching focus_regex, and exclude ignore_regex.",
	}
	if redirect {
		h[0] += " >f"
		h = append(h, "Optionally save the report on the file f")
	}
	return strings.Join(h, "\n")
}

// stringToBool is a custom parser for bools. We avoid using strconv.ParseBool
// to remain compatible with old grains behavior (e.g., treating "" as true).
func stringToBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "true", "t", "yes", "y", "1", "":
		return true, nil
	case "false", "f", "no", "n", "0":
		return false, nil
	default:
		return false, fmt.Errorf(`illegal value "%s" for bool variable`, s)
	}
}
