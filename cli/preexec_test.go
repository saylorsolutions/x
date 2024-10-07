package cli

import (
	flag "github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
)

func TestAddGlobalPreExec(t *testing.T) {
	preExecMux.Lock()
	globalPreExec = nil
	preExecMux.Unlock()
	t.Cleanup(func() {
		preExecMux.Lock()
		globalPreExec = nil
		preExecMux.Unlock()
	})

	var preExecRuns int
	AddGlobalPreExec(func() error {
		preExecRuns++
		return nil
	})
	tlc := NewCommandSet("base")
	tlc.Printer().Redirect(io.Discard)
	testCmd := tlc.AddCommand("test", "Runs the test sub-command").Does(func(flags *flag.FlagSet, out *Printer) error {
		return nil
	})
	testCmd.AddCommand("two", "Runs the test two sub-command").Does(func(flags *flag.FlagSet, out *Printer) error {
		return nil
	})

	assert.NoError(t, tlc.Exec([]string{"test"}))
	assert.Equal(t, 1, preExecRuns, "Pre-exec should be run once here")
	assert.NoError(t, tlc.Exec([]string{"test", "two"}))
	assert.Equal(t, 2, preExecRuns, "Pre-exec should be run again, and only before running 'two'")
}
