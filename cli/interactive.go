package cli

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"
)

var (
	InteractiveFlag         = "-i"                  // InteractiveFlag specifies the flag that the user should pass to trigger [CommandSet.RespondInteractive].
	InteractiveQuitCommands = []string{"quit", "x"} // InteractiveQuitCommands is a slice of strings that should escape from interactive mode.
)

// RespondInteractive will launch an interactive "shell" version of the [CommandSet] if the [InteractiveFlag] is the first argument, indicating that the user is requesting interactive mode.
// This allows printing usage and calling sub-commands.
// Returns false if interactive mode was not requested by the user.
//
// This loop may be interrupted with one of the [InteractiveQuitCommands].
func (s *CommandSet) RespondInteractive() bool {
	args := os.Args[1:]
	if len(args) == 0 {
		return false
	}
	if args[0] != InteractiveFlag {
		return false
	}

	if err := s.interactiveLoop(os.Args[0]); err != nil {
		s.printer.Println("Error running command interactively:", err)
	}
	return true
}

func (s *CommandSet) interactiveLoop(command string) error {
	scanner := bufio.NewScanner(os.Stdin)
	p := s.printer
	p.Printf("Running '%s' interactively. Enter %s to exit.\n", command, strings.Join(InteractiveQuitCommands, " or "))
	for {
		p.Printf("%s> ", s.parent)
		switch {
		case scanner.Scan():
			line := strings.TrimSpace(scanner.Text())
			if len(line) == 0 {
				continue
			}
			if slices.Contains(InteractiveQuitCommands, strings.TrimSpace(strings.ToLower(line))) {
				return nil
			}
			segments := translate(strings.Split(line, " "), func(element string) (string, bool) {
				val := strings.TrimSpace(element)
				if len(val) == 0 {
					return "", false
				}
				return val, true
			})
			if len(segments) > 0 && segments[0] == InteractiveFlag {
				p.Println("Cannot run interactively twice")
				continue
			}
			err := func() error {
				timeout, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
				defer cancel()
				cmd := exec.CommandContext(timeout, command, segments...)
				cmd.Stdout = p.out
				cmd.Stderr = p.out
				return cmd.Run()
			}()
			if err != nil {
				p.Println("Error running command:", err)
			}
		default:
			return scanner.Err()
		}
	}
}

func translate[S ~[]E, E any](slice S, tx func(element E) (E, bool)) S {
	mutated := make(S, 0, len(slice))
	for i := 0; i < len(slice); i++ {
		val, include := tx(slice[i])
		if !include {
			continue
		}
		mutated = append(mutated, val)
	}
	return mutated
}
