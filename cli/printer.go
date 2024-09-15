package cli

import (
	"fmt"
	"io"
	"os"
)

// Printer is provided to easily establish policies for user messages.
// It exposes Print, Println, and Printf methods.
//
// Printer writes to [os.Stderr] by default, but his can be overridden with [Printer.Redirect].
type Printer struct {
	out io.Writer
}

func NewPrinter() *Printer {
	return &Printer{out: os.Stderr}
}

func (p *Printer) Redirect(writer io.Writer) {
	p.out = writer
}

func (p *Printer) Print(msg ...any) {
	_, _ = fmt.Fprint(p.out, msg...)
}

func (p *Printer) Printf(format string, args ...any) {
	_, _ = fmt.Fprintf(p.out, format, args...)
}

func (p *Printer) Println(msg ...any) {
	_, _ = fmt.Fprintln(p.out, msg...)
}
