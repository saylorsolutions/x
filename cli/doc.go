/*
Package cli provides an opinionated package for how a CLI with sub-commands can be structured.

There are a few reasonable (IMHO) policies for how this operates.

  - User-visible output should go to STDERR by default. This is supported with a configurable [Printer].
  - This package uses [pflag] for posix style flags.
  - Flags should NOT be interspersed by default. This makes flag and argument parsing much more consistent and predictable, but can be overridden.
  - Global flags are often confusing and not necessary. Flags apply to the command at hand, while global state may be configured through other means.
  - Sub-command aliases are often very convenient, so they're supported as additional, optional parameters to [CommandSet.AddCommand].

# Invocation

Invoking a CLI with sub-commands can always follow this form:

	CLI_NAME [SUB-COMMAND...] [FLAGS...] [ARGS...]

This consistency helps to build muscle memory for frequent CLI use, and a predictable user experience.
Just calling CLI_NAME will print usage information for the tool.

# Usage by default

Usage information can be incredibly helpful for understanding a tool's purpose and expectations.
That's why the '-h' and '--help' flags are set up by default, with input from the developer with the [Command.Usage] method.

Flag usage and sub-command usage is included in a usage template along with developer-provided usage information.

To display usage information from the root [CommandSet]'s perspective, use [CommandSet.RespondUsage].
This method will return true if the user requested root command usage.

NOTE: Commands will NOT respond with usage by default if an error is returned.

# Prioritizing Dev UX

Developers want nice things too, especially with tooling they rely on.
This is the motivation for interactive mode.

If your CLI calls [CommandSet.RespondInteractive], then you're enabling the use of the [InteractiveFlag] (which can be changed) to enter this mode.
This method will block for interactions and return true if the user requested interactive mode.

If you want to work with a nested sub-command the [UseCommand] can be used to push that string of sub-commands to an invocation stack.
Use the [BackCommand] to pop the invocation stack and go back to where you were.

To exit interactive mode, use one of the [InteractiveQuitCommands] at the prompt.

For more robust interactivity, I can recommend [tview] as a great tool for full TUI support.
It's easy to use, and quick to get productive.
I haven't tried many alternatives because this works well for me. YMMV.

[pflag]: https://github.com/spf13/pflag
[tview]: https://github.com/rivo/tview
*/
package cli
