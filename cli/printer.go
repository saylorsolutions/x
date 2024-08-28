package cli

import (
	"fmt"
	"io"
	"os"
)

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
