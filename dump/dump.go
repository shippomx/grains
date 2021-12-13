package dump

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
)

type Head struct {
	GID	int
	Duration int
}

type Stack struct {
	Location string
	FuncName string
	Params string
}

type Frame struct {
	Reason string
	Size int

	Head
	Stacks []Stack

	LockInfo
}

type LockInfo struct {
	*Stack
	LockType string
	LockHolders []string
}

// Dump is an in-memory representation of dump.proto.
type Dump struct {
	TrimedFrames map[string]TrimedFrame
	RawFrames map[string]*Frame
	Surmary map[string]int64
	Goroutines map[int]int
}

type TrimedFrame struct {
	Frame
	Heads []Head
}

func NewDump() (p *Dump) {
	p = &Dump{
		RawFrames: make(map[string]*Frame),
		TrimedFrames: make(map[string]TrimedFrame),
		Surmary:  make(map[string]int64),
		Goroutines: make(map[int]int),
	}
	return
}

func (p *Dump) genTrimedKey(tf *Frame, idx int) string {
	tKey := fmt.Sprintf("%s_%d", tf.Reason, idx)
	f, ok := p.TrimedFrames[tKey]
	if ok {
		if f.hasHighSimilarity(tf) {
			return tKey
		} else {
			idx++
			return p.genTrimedKey(tf, idx)
		}
	}
	return tKey
}

func (p *Dump) InsertRawFrame(f *Frame) {
	key := fmt.Sprintf("%d_%d", f.GID, f.Duration)
	p.RawFrames[key] = f
	p.Surmary[f.Reason]++
	p.Goroutines[f.GID] = f.Duration
}



func (p *Dump) GetFrameByGID(gid int) (frame *Frame) {
	return p.getFrameByGID(gid, p.Goroutines[gid])
}


func (p *Dump) getFrameByGID(gid, idx int) (frame *Frame) {
	key := fmt.Sprintf("%d_%d", gid, idx)
	f, ok := p.RawFrames[key]
	if !ok {
		return nil
	}
	return f
}

func (p *Dump) GetFramesByReason(reason string, idx int) (frames []*Frame) {
	key := fmt.Sprintf("%s_%d", reason, idx)
	tf, ok := p.TrimedFrames[key]
	if ok {
		for _, head := range tf.Heads	{
			f := p.getFrameByGID(head.GID, head.Duration)
			frames = append(frames, f)
		}
		idx++
		fs := p.GetFramesByReason(reason, idx)
		if len(fs) == 0 {
			return
		}
		frames = append(frames, fs...)
	}
	return
}

func (p *Dump) InsertTrimedFrame(f *Frame) {
	key := p.genTrimedKey(f, 0)
	tf, ok := p.TrimedFrames[key]
	if ok {
		tf.Heads = append(tf.Heads, f.Head)
	} else {
		tf = TrimedFrame{
			Frame: *f,
		}
		tf.Heads = append(tf.Heads, f.Head)
	}
	p.TrimedFrames[key] = tf
}

func (p *Dump) unmarshal(data []byte) {
	var elems []string
	//elems = bytes.Split(data, []byte{13, 10, 13, 10}) // two new lines
	//if len(elems) == 1 {
	//	elems = bytes.Split(data, []byte{120, 98, 102, 10}) // two new lines
	//}
//		elems = strings.Split(string(data), `
//
//`)
	elems = strings.Split(string(data), "\n\r\n")
	if len(elems) <= 1 {
		return
	}
	for _, elem := range elems {
		frame := &Frame{}
		lines := strings.Split(string(elem), "\r\n")
		for k, line := range lines {
			if strings.HasPrefix(line, "goroutine") {
				frame.decodeHead(lines[k])
				frame.decodeBody(lines[k+1:])
				break
			}
		}
		if frame.GID > 0 {
			p.InsertTrimedFrame(frame)
			p.InsertRawFrame(frame)

			// TODO 这里可能会存在相同的GID
			p.Goroutines[frame.GID] = frame.Duration
		}
	}
}

// Parse parses a dump and checks for its validity. The input
// may be a gzip-compressed encoded protobuf or one of many legacy
// dump formats which may be unsupported in the future.
func (p *Dump) Parse(r io.Reader) error {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	return p.ParseData(data)
}

// ParseData parses a dump from a buffer and checks for its
// validity.
func (p *Dump) ParseData(data []byte) error {
	var err error
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		gz, err := gzip.NewReader(bytes.NewBuffer(data))
		if err == nil {
			data, err = ioutil.ReadAll(gz)
		}
		if err != nil {
			return fmt.Errorf("decompressing dump: %v", err)
		}
	}
	if err = p.ParseUncompressed(data); err != nil {
		return err
	}

	if err != nil {
		return fmt.Errorf("parsing dump: %v", err)
	}

	return nil
}

var errNoData = fmt.Errorf("empty input file")

// ParseUncompressed parses an uncompressed protobuf into a dump.
func  (p *Dump) ParseUncompressed(data []byte) (err error) {
	if len(data) == 0 {
		return errNoData
	}
	p.unmarshal(data)
	if len(p.RawFrames) <= 1 {
		return errors.New("Cannot unmarshal file")
	}
	return nil
}

func serialize(p *Dump) []byte {
	return nil
}

// Write writes the dump as a gzip-compressed marshaled protobuf.
func (p *Dump) Write(w io.Writer) error {
	zw := gzip.NewWriter(w)
	defer zw.Close()
	_, err := zw.Write(serialize(p))
	return err
}

// WriteUncompressed writes the dump as a marshaled protobuf.
func (p *Dump) WriteUncompressed(w io.Writer) error {
	_, err := w.Write(serialize(p))
	return err
}

func (p *Dump) Duplicated() (p2 *Dump) {
	return p
	//p2 = NewDump()
	//for k, v := range p.RawFrames {
	//	p2.RawFrames[k] = v
	//}
	//for k, v := range p.TrimedFrames {
	//	p2.TrimedFrames[k] = v
	//}
	//for k, v := range p.Surmary {
	//	p2.Surmary[k] = v
	//}
	//for k, v := range p.Goroutines{
	//	p2.Goroutines[k] = v
	//}
	//return
}
