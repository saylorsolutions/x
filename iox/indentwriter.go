package iox

import (
	"bytes"
	"io"
)

var (
	crlf = []byte("\r\n")
	lf   = []byte("\n")
)

// LineEnding is used to specify a particular line ending for IndentWriter.
// If no LineEnding is specified or is unrecognized, then EndingLF is used.
type LineEnding int

const (
	EndingDefault LineEnding = iota
	EndingCRLF
	EndingLF
)

func (le LineEnding) ending() []byte {
	switch le {
	case EndingCRLF:
		return crlf
	case EndingDefault:
		fallthrough
	case EndingLF:
		fallthrough
	default:
		return lf
	}
}

// IndentWriter is a type that intercepts Write calls to the original io.Writer to add indentation depending on the number of calls to Indent and Outdent.
// The specific IndentString may be customized for all calls to Write.
//
// If no call to Indent has been made before a Write call, then no indentation is added.
// Line endings will still be normalized for all calls to Write.
type IndentWriter struct {
	Out          io.Writer
	IndentString string
	EOL          LineEnding
	stack        []*bytes.Buffer
}

func NewIndentWriter(out io.Writer, indent string) *IndentWriter {
	return &IndentWriter{
		Out:          out,
		IndentString: indent,
	}
}

func (iw *IndentWriter) targetBuf() *bytes.Buffer {
	if len(iw.stack) == 0 {
		return nil
	}
	return iw.stack[len(iw.stack)-1]
}

func (iw *IndentWriter) target() io.Writer {
	if len(iw.stack) == 0 {
		return iw.Out
	}
	return iw.stack[len(iw.stack)-1]
}

func (iw *IndentWriter) popStack() []byte {
	switch len(iw.stack) {
	case 0:
		return nil
	default:
		buf := iw.targetBuf()
		iw.stack = iw.stack[:len(iw.stack)-1]
		return buf.Bytes()
	}
}

func (iw *IndentWriter) indentLines(data []byte) (int, error) {
	out := iw.target()
	eol := iw.EOL.ending()
	lines := bytes.Split(data, lf)
	if len(iw.IndentString) == 0 {
		iw.IndentString = "\t"
	}
	indent := []byte(iw.IndentString)
	var written int
	for i, line := range lines {
		line = bytes.TrimRight(line, "\r")
		if len(bytes.TrimSpace(line)) > 0 {
			line = append(indent, line...)
		}
		if i > 0 {
			line = append(eol, line...)
		}
		num, err := out.Write(line)
		written += num
		if err != nil {
			return written, err
		}
	}
	return written, nil
}

func (iw *IndentWriter) pushStack() {
	iw.stack = append(iw.stack, new(bytes.Buffer))
}

func (iw *IndentWriter) Write(buf []byte) (int, error) {
	if len(iw.stack) == 0 {
		lines := bytes.Split(buf, lf)
		var (
			totalWritten int
			written      int
			err          error
		)
		for i, line := range lines {
			line = bytes.Trim(line, "\r")
			if i > 0 {
				line = append(iw.EOL.ending(), line...)
			}
			written, err = iw.Out.Write(line)
			totalWritten += written
			if err != nil {
				return totalWritten, err
			}
		}
		return totalWritten, nil
	}
	return iw.target().Write(buf)
}

// Indent adds a level of indentation to output.
func (iw *IndentWriter) Indent() {
	iw.pushStack()
}

// Outdent removes a level of indentation from output.
// If Indent hasn't been called, then Outdent does nothing.
func (iw *IndentWriter) Outdent() (int, error) {
	stackBytes := iw.popStack()
	if len(stackBytes) == 0 {
		return 0, nil
	}
	return iw.indentLines(stackBytes)
}
