package cli

import (
	"bufio"
	"fmt"
	"golang.org/x/term"
	"io"
	"os"
)

// Printer is provided to easily establish policies for user messages.
// It exposes Print, Println, and Printf methods.
//
// Printer writes to [os.Stderr] by default, but this can be overridden with [Printer.Redirect].
type Printer struct {
	out io.Writer
	in  *os.File
}

func NewPrinter() *Printer {
	return &Printer{out: os.Stderr, in: os.Stdin}
}

// Redirect will make the Printer print to this output instead.
// Defaults to [os.Stderr].
func (p *Printer) Redirect(writer io.Writer) {
	p.out = writer
}

// RedirectInput will make the Printer read from a different file when prompting.
func (p *Printer) RedirectInput(in *os.File) {
	p.in = in
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

// Prompt will prompt the user for input, then read and return the next line of text.
func (p *Printer) Prompt(msg string, args ...any) (string, error) {
	p.Printf(msg, args...)
	scanner := bufio.NewScanner(p.in)
	if !scanner.Scan() {
		return "", fmt.Errorf("failed to scan from input: %w", scanner.Err())
	}
	return scanner.Text(), nil
}

// PromptNoEcho will prompt the user for input, then read and return the next line of text without printing input to the terminal.
func (p *Printer) PromptNoEcho(msg string, args ...any) ([]byte, error) {
	p.Printf(msg, args...)
	line, err := term.ReadPassword(int(p.in.Fd()))
	if err != nil {
		return nil, fmt.Errorf("failed to read from terminal: %w", err)
	}
	p.Println()
	return line, nil
}
