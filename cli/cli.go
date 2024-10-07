package cli

import (
	"errors"
	"fmt"
	flag "github.com/spf13/pflag"
	"os"
	"regexp"
	"slices"
	"strings"
)

var (
	ErrUnknownCommand = errors.New("unknown command")
	HelpPatterns      = []string{"--help", "-h"} // HelpPatterns is a slice of flags that should trigger the output of usage information with the top-level [CommandSet].

	keyCleansePattern = regexp.MustCompile(`\s`)
)

// CommandFunc is a function that may be executed within a [Command].
type CommandFunc = func(flags *flag.FlagSet, printer *Printer) error

// Command is an executable function in a CLI.
// It should be linked to a [CommandSet] to establish a tree of commands available to the user.
type Command struct {
	CommandSet
	flags      *flag.FlagSet
	exec       CommandFunc
	key        string
	parent     string
	shortUsage string
	printer    *Printer
	aliases    []string
}

func cleanseKey(key string) string {
	return keyCleansePattern.ReplaceAllString(strings.ToLower(key), "")
}

func newCommand(key, parent, shortUsage string, printer *Printer) *Command {
	key = cleanseKey(key)
	fs := flag.NewFlagSet(key, flag.ContinueOnError)
	fs.BoolP("help", "h", false, "Prints this usage information")
	fs.SetInterspersed(false)
	cmd := &Command{flags: fs, key: key, parent: parent, shortUsage: shortUsage, printer: printer}
	if len(parent) > 0 {
		cmd.CommandSet.parent = strings.Join([]string{parent, key}, " ")
	} else {
		cmd.CommandSet.parent = key
	}
	cmd.Usage("").Does(func(flags *flag.FlagSet, _ *Printer) error {
		if flags.Usage == nil {
			cmd.Usage("")
		}
		flags.Usage()
		return nil
	})
	return cmd
}

// Does specifies the [CommandFunc] that should be executed by this [Command].
func (c *Command) Does(commandFunc CommandFunc) *Command {
	if commandFunc == nil {
		return c
	}
	c.exec = commandFunc
	return c
}

// Parent retrieves the parent [Command] name.
func (c *Command) Parent() string {
	return c.parent
}

// CommandPath returns the reference chain for this [Command].
func (c *Command) CommandPath() string {
	return fmt.Sprintf("%s %s", c.parent, c.key)
}

// Flags returns the [flag.FlagSet] for this [Command].
func (c *Command) Flags() *flag.FlagSet {
	return c.flags
}

// Usage allows specifying a longer description of the [Command] that will be output when a [HelpPatterns] flag is passed.
//
// The short description, flag usages, and sub-command usages will be appended to this description.
func (c *Command) Usage(format string, args ...any) *Command {
	text := fmt.Sprintf(format, args...)
	if len(c.Parent()) > 0 && len(text) > 0 {
		text = c.Parent() + " " + text
	}
	if len(text) > 0 {
		text = `USAGE:
` + text
	}
	c.flags.Usage = func() {
		var buf strings.Builder
		if len(text) == 0 {
			buf.WriteString("\n" + c.shortUsage)
		} else {
			if !strings.HasSuffix(text, "\n") {
				text += "\n"
			}
			buf.WriteString(fmt.Sprintf(`%s

%s`, c.shortUsage, text))
		}
		buf.WriteString("\nFLAGS\n")
		buf.WriteString(c.flags.FlagUsages())
		if len(c.CommandSet.commands) > 0 {
			buf.WriteString("\nCOMMANDS\n")
			buf.WriteString(c.CommandUsages())
		}
		fmt.Print(buf.String())
	}
	return c
}

// Exec executes the command with given arguments, parsing flags.
func (c *Command) Exec(args []string) error {
	if err := c.CommandSet.Exec(args); err != nil {
		if !errors.Is(err, ErrUnknownCommand) {
			return err
		}
	} else {
		return nil
	}
	if err := c.flags.Parse(args); err != nil {
		return err
	}
	if val, _ := c.flags.GetBool("help"); val {
		if c.flags.Usage == nil {
			c.Usage("")
		}
		c.flags.Usage()
		return nil
	}
	if err := runGlobalPreExec(); err != nil {
		return err
	}
	return c.exec(c.flags, c.Printer())
}

// CommandSet is a group of [Command].
type CommandSet struct {
	commands map[string]*Command
	aliases  map[string]*Command
	printer  *Printer
	parent   string
}

// NewCommandSet is used to set up a top level [CommandSet] as the root of a CLI's command structure.
//
// Note: the parent(s) passed to this function will be used to populate sub-command usage information.
// So they should only contain the commands used to invoke this [CommandSet].
func NewCommandSet(parent ...string) *CommandSet {
	var _parent string
	if len(parent) > 0 {
		_parent = strings.Join(parent, " ")
	}
	return &CommandSet{printer: NewPrinter(), parent: _parent}
}

// Parent retrieves the parent [CommandSet] name.
func (s *CommandSet) Parent() string {
	return s.parent
}

// AddCommand adds a sub-command to this [CommandSet].
// The key parameter will be cleansed to remove spaces, and normalize to lower-case.
// Aliases may be added as a way to support shorter variants of the same [Command].
func (s *CommandSet) AddCommand(key, shortUsage string, aliases ...string) *Command {
	key = cleanseKey(key)
	cmd := newCommand(key, s.parent, shortUsage, s.Printer())
	if s.commands == nil {
		s.commands = map[string]*Command{}
	}
	s.commands[key] = cmd
	if len(aliases) > 0 {
		_aliases := make([]string, 0, len(aliases))
		for _, alias := range aliases {
			alias = cleanseKey(alias)
			if len(alias) == 0 {
				continue
			}
			if s.aliases == nil {
				s.aliases = map[string]*Command{}
			}
			s.aliases[alias] = cmd
			_aliases = append(_aliases, alias)
		}
		slices.Sort(_aliases)
		cmd.aliases = _aliases
	}
	return cmd
}

// Printer returns the cached [Printer] for this [CommandSet].
func (s *CommandSet) Printer() *Printer {
	if s.printer == nil {
		s.printer = NewPrinter()
	}
	return s.printer
}

// Exec executes this [CommandSet].
// It's expected that the first 1+ arguments include the key/alias for a sub-command.
func (s *CommandSet) Exec(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: no arguments", ErrUnknownCommand)
	}
	key := strings.ToLower(args[0])
	cmd, ok := s.commands[key]
	if !ok {
		cmd, ok = s.aliases[key]
		if !ok {
			return fmt.Errorf("%w: %s", ErrUnknownCommand, args[0])
		}
	}
	return cmd.Exec(args[1:])
}

// RespondUsage will print usage information with the given [Printer] if one of [HelpPatterns] is given as the first argument.
// If usage information was printed, then true will be returned.
func (s *CommandSet) RespondUsage(format string, vals ...any) bool {
	args := os.Args[1:]
	if len(args) == 0 {
		return false
	}
	if slices.Contains(HelpPatterns, args[0]) {
		text := fmt.Sprintf(format, vals...)
		if len(text) > 0 {
			text = strings.TrimSuffix("\n\n"+text, "\n")
		}
		usage := fmt.Sprintf(`%s%s

COMMANDS:
%s`, s.parent, text, s.CommandUsages())
		s.printer.Print(usage)
		return true
	}
	return false
}

// CommandUsages returns a string including the usage information for sub-commands in this [CommandSet].
//
// The sub-command keys will be sorted alphabetically before output.
func (s *CommandSet) CommandUsages() string {
	var (
		buf         strings.Builder
		cmds        []*Command
		keys        = make([]string, len(s.commands))
		withAliases = make([]string, len(s.commands))
		maxLen      int
		i           int
	)
	for key := range s.commands {
		keys[i] = key
		withAliases[i] = key
		i++
	}
	slices.Sort(keys)
	slices.Sort(withAliases)

	cmds = make([]*Command, len(keys))
	for i, key := range keys {
		cmd := s.commands[key]
		cmds[i] = cmd
		if len(cmd.aliases) > 0 {
			withAliases[i] = strings.Join(append([]string{key}, cmd.aliases...), ", ")
		}
		l := len(withAliases[i])
		if l > maxLen {
			maxLen = l
		}
	}
	fmtStr := fmt.Sprintf("  %%-%ds\t%%s\n", maxLen)
	for i, cmd := range cmds {
		buf.WriteString(fmt.Sprintf(fmtStr, withAliases[i], cmd.shortUsage))
	}
	return buf.String()
}
