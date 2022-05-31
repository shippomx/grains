package driver

import (
	"bytes"
	"errors"
	"github.com/shippomx/grains/dump"
	"github.com/shippomx/grains/internal/plugin"
	"github.com/shippomx/grains/internal/report"
	"os"
)

// Grains acquires a dump, and symbolizes it using a dump
// manager. Then it generates a report formatted according to the
// options selected through the flags package.
func Grains(eo *plugin.Options) error {
	// Remove any temporary files created during grains processing.
	defer cleanupTempFiles()

	o := setDefaults(eo)

	src, cmd, err := parseFlags(o)
	if err != nil {
		return err
	}

	p, err := fetchDumps(src, o)
	if err != nil {
		return err
	}

	if p == nil {
		return errors.New("No such file " + src.Base[0])
	}

	if cmd != nil {
		return generateReport(p, cmd, currentConfig(), o)
	}

	return interactive(p, o)
}

func generateRawReport(p *dump.Dump, cmd []string) (c *command, rpt *report.Report, err error) {
	// Get report output format
	c = grainsCommands[cmd[0]]
	if c == nil {
		err = errors.New("unexpected nil command")
		return
	}

	ro := &report.Options{}
	rpt = report.New(p, ro)

	return c, rpt, err
}

func generateReport(p *dump.Dump, cmd []string, cfg config, o *plugin.Options) error {
	c, rpt, err := generateRawReport(p, cmd)
	if err != nil {
		return err
	}

	// Generate the report.
	dst := new(bytes.Buffer)
	if err := report.Generate(dst, rpt, cmd); err != nil {
		return err
	}
	src := dst

	// If necessary, perform any data post-processing.
	if c.postProcess != nil {
		dst = new(bytes.Buffer)
		if err := c.postProcess(src, dst, o.UI); err != nil {
			return err
		}
		src = dst
	}

	// If no output is specified, use default visualizer.
	output := cfg.Output
	if output == "" {
		if c.visualizer != nil {
			return c.visualizer(src, os.Stdout, o.UI)
		}
		_, err := src.WriteTo(os.Stdout)
		return err
	}

	// Output to specified file.
	o.UI.PrintErr("Generating report in ", output)
	out, err := o.Writer.Open(output)
	if err != nil {
		return err
	}
	if _, err := src.WriteTo(out); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
