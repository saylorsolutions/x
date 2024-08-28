package cli

import (
	flag "github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestCommand_Exec(t *testing.T) {
	cmd := newCommand("test", "", "test command", NewPrinter())
	assert.NoError(t, cmd.Exec(nil))

	executed := false
	cmd.Does(func(flags *flag.FlagSet, _ *Printer) error {
		executed = true
		return nil
	})
	assert.NoError(t, cmd.Exec(nil))
	assert.True(t, executed)
}

func TestCommandSet_Exec(t *testing.T) {
	set := NewCommandSet()
	assert.ErrorIs(t, set.Exec(nil), ErrUnknownCommand)

	cmd := set.AddCommand("test", "test command")
	assert.NoError(t, set.Exec([]string{"test"}))

	executed := false
	cmd.Does(func(flags *flag.FlagSet, _ *Printer) error {
		executed = true
		return nil
	})
	assert.NoError(t, set.Exec([]string{"test"}))
	assert.True(t, executed)

	assert.ErrorIs(t, set.Exec([]string{"Does", "not", "exist"}), ErrUnknownCommand)
}

func TestCommand_AddSubCommand(t *testing.T) {
	cmdExecuted := 0
	subExecuted := 0
	cmd := testCommandWithSubcommand(t, &cmdExecuted, &subExecuted)

	assert.NoError(t, cmd.Exec([]string{"-h"}))

	cmd = testCommandWithSubcommand(t, &cmdExecuted, &subExecuted)
	assert.NoError(t, cmd.Exec([]string{"blah"}), "Should execute test without error")
	assert.Equal(t, 1, cmdExecuted)
	assert.Equal(t, 0, subExecuted)

	cmd = testCommandWithSubcommand(t, &cmdExecuted, &subExecuted)
	assert.NoError(t, cmd.Exec([]string{"SUB"}))
	assert.Equal(t, 1, cmdExecuted)
	assert.Equal(t, 1, subExecuted)
}

func TestCommandSet_AddCommand_Aliases(t *testing.T) {
	cmdExecuted := 0
	subExecuted := 0
	cmd := testCommandSet(t, &cmdExecuted, &subExecuted)
	assert.NoError(t, cmd.Exec([]string{"test", "a"}), "Should execute test without error")
	assert.Equal(t, 0, cmdExecuted)
	assert.Equal(t, 1, subExecuted)

	cmd = testCommandSet(t, &cmdExecuted, &subExecuted)
	assert.NoError(t, cmd.Exec([]string{"test", "b"}), "Should execute test without error")
	assert.Equal(t, 0, cmdExecuted)
	assert.Equal(t, 2, subExecuted)
}

func TestPrinter_RespondUsage(t *testing.T) {
	cmdExecuted := 0
	subExecuted := 0
	cmd := testCommandSet(t, &cmdExecuted, &subExecuted)
	tmp := os.Args
	t.Cleanup(func() {
		os.Args = tmp
	})
	os.Args = []string{"command", HelpPatterns[0], "something", "else"}
	responded := cmd.RespondUsage("Printed usage")
	assert.True(t, responded, "Should have responded with cmd usage")
}

func testCommandSet(t *testing.T, cmdExecuted, subExecuted *int) *CommandSet {
	set := NewCommandSet("commands")
	cmd := set.AddCommand("test", "test command", "t")
	cmd.Flags().String("message", "", "Sets a message")
	cmd.Does(func(_ *flag.FlagSet, _ *Printer) error {
		*cmdExecuted++
		return nil
	})

	sub := cmd.AddCommand("sub", "test subcommand", "a", "b")
	assert.Equal(t, "commands test", sub.parent)
	sub.Does(func(flags *flag.FlagSet, _ *Printer) error {
		*subExecuted++
		return nil
	})
	return set
}

func testCommandWithSubcommand(t *testing.T, cmdExecuted, subExecuted *int) *Command {
	cmd := newCommand("test", "", "test command", NewPrinter()).Does(func(flags *flag.FlagSet, _ *Printer) error {
		*cmdExecuted++
		return nil
	})
	cmd.Flags().String("message", "", "Sets a message")

	sub := cmd.AddCommand("sub", "test subcommand", "a", "b")
	assert.Equal(t, "test", sub.parent)
	sub.Does(func(flags *flag.FlagSet, _ *Printer) error {
		*subExecuted++
		return nil
	})
	return cmd
}
