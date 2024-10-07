package cli

import (
	"fmt"
	flag "github.com/spf13/pflag"
)

func ExampleNewCommandSet() {
	// The NewCommandSet function is called to get a top level command set.
	// The string used should be the name used to invoke your CLI, but it could also be os.Args[0].
	tlc := NewCommandSet("my-cli")

	// Sub-commands can be added easily.
	sub := tlc.AddCommand("sub-command", "Shows an example of a sub-command")

	// The flags for a Command or CommandSet can be accessed to set up whatever flags are needed.
	sub.Flags().Bool("do-something", false, "Makes the sub-command do something")

	// Usage hints can be set with the Usage method. No need to mess with the usage function in flags.
	// Parent command references will automatically be prepended to this string.
	// In this case the actual usage string will be 'my-cli sub-command [FLAGS]'.
	sub.Usage("sub-command [FLAGS]")

	// Functionality is defined with the Does method.
	sub.Does(func(flags *flag.FlagSet, _ *Printer) error {
		// Flags are already parsed by the time this function is executed.
		if MustGet(flags.GetBool("do-something")) {
			// Using fmt for the example, but the Printer should be used to communicate with the user.
			fmt.Println("sub-command ran")
		}
		return nil
	})

	// os.Args[1:] should be passed to tlc.Exec
	// Sub-commands will be matched case-insensitive.
	if err := tlc.Exec([]string{"suB-ComMAnd", "--do-something"}); err != nil {
		fmt.Println("Something bad happened!")
	}
	fmt.Println()

	// Help flags are automatically set up for each command.
	_ = tlc.Exec([]string{"sub-command", "-h"})

	// Output:
	// sub-command ran
	//
	// Shows an example of a sub-command
	//
	// USAGE:
	// my-cli sub-command [FLAGS]
	//
	// FLAGS
	//       --do-something   Makes the sub-command do something
	//   -h, --help           Prints this usage information
}
