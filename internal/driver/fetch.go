package driver

import (
	"fmt"
	"github.com/shippomx/grains/dump"
	"github.com/shippomx/grains/internal/plugin"
	"os"
	"sync"
)

// fetchDumps fetches and symbolizes the dumps specified by s.
// It will merge all the dumps it is able to retrieve, even if
// there are some failures. It will return an error if it is unable to
// fetch any dumps.
func fetchDumps(s *source, o *plugin.Options) (*dump.Dump, error) {
	sources := make([]dumpSource, 0, len(s.Sources))
	for _, src := range s.Sources {
		sources = append(sources, dumpSource{
			addr:   src,
			source: s,
		})
	}

	bases := make([]dumpSource, 0, len(s.Base))
	for _, src := range s.Base {
		bases = append(bases, dumpSource{
			addr:   src,
			source: s,
		})
	}

	p, _, _, err := grabSourcesAndBases(sources, bases, o.UI)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func grabSourcesAndBases(sources, bases []dumpSource, ui plugin.UI) (*dump.Dump, *dump.Dump, bool, error) {
	wg := sync.WaitGroup{}
	wg.Add(2)
	var psrc, pbase *dump.Dump
	var savesrc, savebase bool
	var errsrc, errbase error
	var countsrc, countbase int
	go func() {
		defer wg.Done()
		psrc, savesrc, countsrc, errsrc = chunkedGrab(sources, ui)
	}()
	go func() {
		defer wg.Done()
		pbase, savebase, countbase, errbase = chunkedGrab(bases, ui)
	}()
	wg.Wait()
	save := savesrc || savebase

	if want, got := len(sources), countsrc; want != got {
		ui.PrintErr(fmt.Sprintf("Fetched %d source dumps out of %d", got, want))
	}
	if want, got := len(bases), countbase; want != got {
		ui.PrintErr(fmt.Sprintf("Fetched %d base dumps out of %d", got, want))
	}

	return psrc, pbase, save, nil
}

// chunkedGrab fetches the dumps described in source and merges them into
// a single dump. It fetches a chunk of dumps concurrently, with a maximum
// chunk size to limit its memory usage.
func chunkedGrab(sources []dumpSource, ui plugin.UI) (p *dump.Dump, save bool, count int, chunkErr error) {
	const chunkSize = 64

	for start := 0; start < len(sources); start += chunkSize {
		end := start + chunkSize
		if end > len(sources) {
			end = len(sources)
		}
		p, save, count, chunkErr = concurrentGrab(sources[start:end], ui)
		if chunkErr != nil {
			return nil, false, 0, chunkErr
		}
	}

	return p, save, count, nil
}

// concurrentGrab fetches multiple dumps concurrently
func concurrentGrab(sources []dumpSource, ui plugin.UI) (*dump.Dump, bool, int, error) {
	wg := sync.WaitGroup{}
	wg.Add(len(sources))
	for i := range sources {
		go func(s *dumpSource) {
			defer wg.Done()
			s.p, s.err = grabDump(s.source, s.addr)
		}(&sources[i])
	}
	wg.Wait()

	var save bool
	dumps := make([]*dump.Dump, 0, len(sources))
	for _, s := range sources {
		if err := s.err; err != nil {
			ui.PrintErr(s.addr + ": " + err.Error())
			continue
		}
		dumps = append(dumps, s.p)
	}

	if len(dumps) == 0 {
		return nil, false, 0, nil
	}

	//p, err := combineDumps(dumps)
	//if err != nil {
	//	return nil, false, 0, err
	//}
	return sources[0].p, save, len(dumps), nil
}

func combineDumps(dumps []*dump.Dump) (*dump.Dump, error) {
	return nil, nil
}

type dumpSource struct {
	addr   string
	source *source

	p      *dump.Dump
	err    error
}

// setTmpDir prepares the directory to use to save dumps retrieved
// remotely. It is selected from PPROF_TMPDIR, defaults to $HOME/grains, and, if
// $HOME is not set, falls back to os.TempDir().
func setTmpDir(ui plugin.UI) (string, error) {
	var dirs []string
	if dumpDir := os.Getenv("PPROF_TMPDIR"); dumpDir != "" {
		dirs = append(dirs, dumpDir)
	}

	dirs = append(dirs, os.TempDir())
	for _, tmpDir := range dirs {
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			ui.PrintErr("Could not use temp dir ", tmpDir, ": ", err.Error())
			continue
		}
		return tmpDir, nil
	}
	return "", fmt.Errorf("failed to identify temp dir")
}

// grabDump fetches a dump. Returns the dump, sources for the
// dump mappings, a bool indicating if the dump was fetched
// remotely, and an error.
func grabDump(s *source, source string) (p *dump.Dump, err error) {
	return fetch(source)
}

// fetch fetches a dump from source, within the timeout specified,
// producing messages through the ui. It returns the dump and the
// url of the actual source of the dump for remote dumps.
func fetch(source string) (p *dump.Dump, err error) {
	f, err := os.Open(source)
	if err == nil {
		defer f.Close()
		p = dump.NewDump()
		err = p.Parse(f)
	}
	return
}
