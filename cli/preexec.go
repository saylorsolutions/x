package cli

import "sync"

// PreExec is a function that may run before execution of a [Command]
type PreExec func() error

var (
	preExecMux    sync.Mutex
	globalPreExec []PreExec
)

// AddGlobalPreExec registers a function that will be executed right before a [Command] runs.
// If an error is returned from a [PreExec], then the [Command] will not be executed, and the error will be returned from Exec instead.
// Note that no [PreExec] commands will be executed for calling the top level [CommandSet], since it just prints usage.
//
// Passing a nil [PreExec] function to this function will panic.
func AddGlobalPreExec(fn PreExec) {
	if fn == nil {
		panic("nil pre-exec function")
	}
	preExecMux.Lock()
	defer preExecMux.Unlock()
	globalPreExec = append(globalPreExec, fn)
}

func runGlobalPreExec() error {
	preExecMux.Lock()
	defer preExecMux.Unlock()
	for _, fn := range globalPreExec {
		err := fn()
		if err != nil {
			return err
		}
	}
	return nil
}
