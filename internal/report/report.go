package report

import (
	"fmt"
	"github.com/shippomx/grains/dump"
	"io"
	"os"
	"strconv"
	"time"
)

// Output formats.
const (
	Text = iota
	Raw
)

// Options are the formatting and filtering options used to generate a
// dump.
type Options struct {
	OutputFormat int
}

// Generate generates a report as directed by the Report.
func Generate(w io.Writer, rpt *Report, cmd []string) (err error) {
	switch cmd[0] {
	case "trim":
		trimStacks(w, rpt)
	case "show":
		printFrame(w, rpt, cmd[1])
	case "dump":
		saveTrimed(w, rpt)
	}

	return
}

// Report contains the data and associated routines to extract a
// report from a dump.
type Report struct {
	prof    *dump.Dump
	options *Options
}

// New builds a new report indexing the sample values interpreting the
// samples with the provided function.
func New(prof *dump.Dump, o *Options) *Report {
	// Trim
	return &Report{
		prof:    prof.Duplicated(),
		options: o,
	}
}

func hasDeadLock(f1, f2 *dump.Frame) bool {
	if len(f1.LockHolders) < 1 || len(f2.LockHolders) < 1 {
		return false
	}
	for idxi, hi1 := range f1.LockHolders {
		for idxj, hj1 := range f2.LockHolders {
			if hi1 == hj1 { // find the first equal pair
				for j := idxj + 1; j < len(f2.LockHolders); j++ {
					thj := f2.LockHolders[j]
					for i := idxi - 1; i > 0; i-- {
						thi := f1.LockHolders[i]
						if thi == thj { // find the second equal pair
							return true
						}
					}
				}
			}
		}
	}
	return false
}

func trimStacks(w io.Writer, rpt *Report) {
	fmt.Fprintf(w, "================= Summary =================\n")
	fmt.Fprint(w, "[blocked goroutine types]:\n")
	for reason, cnt := range rpt.prof.Surmary {
		fmt.Fprintf(w, "%s: %d\n", reason, cnt)
		if cnt > 1 {
			frames := rpt.prof.GetFramesByReason(reason, 0)
			for i := 0; i < len(frames); i++ {
				for j := i + 1; j < len(frames); j++ {
					if hasDeadLock(frames[i], frames[j]) {
						fmt.Fprintf(w, "================= WARNING DEAD LOCK %s =================\n", frames[i].Reason)
						fmt.Fprintf(w, "goroutine %d has surspicous DEAD LOCK with goroutine %d\n", frames[i].GID, frames[j].GID)
						fmt.Fprintf(w, "LockHolders of goroutine %d: %v\n", frames[i].GID, frames[i].LockHolders)
						fmt.Fprintf(w, "LockHolders of goroutine %d: %v\n", frames[j].GID, frames[j].LockHolders)
					}
				}
			}
		}
	}
	return
}

func printFrame(w io.Writer, rpt *Report, gid string) {
	id, _ := strconv.Atoi(gid)
	f := rpt.prof.GetFrameByGID(id)
	if f == nil {
		fmt.Fprintf(w, "no such goroutine %s, try another one\n", gid)
		return
	}
	fmt.Fprintf(w, "================= goroutine %s start =================\n", gid)

	fmt.Fprintf(w, "goroutine %d [%s, %d minutes]:\n", f.GID, f.Reason, f.Duration)
	for _, stack := range f.Stacks {
		fmt.Fprintf(w, "%s(%s)\n\t%s\n", stack.FuncName, stack.Params, stack.Location)
	}

	fmt.Fprintf(w, "================= goroutine %s end =================\n", gid)
	return
}

func saveTrimed(w io.Writer, rpt *Report) {
	file, err := os.Create(fmt.Sprintf("trimed.%d.log", time.Now().Unix()))
	if err != nil {
		fmt.Fprint(w, err)
		return
	}
	defer file.Close()

	printTrimed(file, rpt)
}

func printTrimed(w io.Writer, rpt *Report) {
	for reason, frame := range rpt.prof.TrimedFrames {
		fmt.Fprintf(w, "[%s]:\n", reason)
		if frame.LockInfo.Stack != nil {
			fmt.Fprint(w, "[LockType:%s, FuncName: %s, Location: %s]\n", frame.LockInfo.LockType, frame.LockInfo.FuncName, frame.Location)
		}
		for _, head := range frame.Heads {
			fmt.Fprintf(w, "{gid: %d, duration: %d min}, ", head.GID, head.Duration)
		}

		fmt.Fprintf(w, "\n")
		for _, stack := range frame.Stacks {
			fmt.Fprintf(w, "\t%s %s\n%s\n", stack.FuncName, stack.Params, stack.Location)
		}

		fmt.Fprintf(w, "\n")
	}
}
