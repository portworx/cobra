package cmd

var (
	// yamlSpecFile to write to.
	yamlSpecFile string

	// commands is a list of commands that get marshaled as YAML spec.
	commands []*cmdSpec
)

// flag type constants define allowed types for a flag.
const (
	//flag is boolean and takes no value on CLI.
	FlagBool = "bool"

	// flag is a string and takes single string value on CLI.
	FlagStr = "str"

	// flag is an int and takes single int value on CLI.
	FlagInt = "int"

	// flag is an int but we cast to uint in CLI. This will be deprecated.
	FlagUint = "uint"

	// flag can be repeated with different strings as values.
	FlagStrSlice = "str-slice"

	// flag can be repeated with different ints as values.
	FlagIntSlice = "int-slice"
)

// flagSpec defines an individual flag in a command.
type flagSpec struct {
	// Name is name of the command.
	// Please use simple names with all lowecase.
	// Examples: my-command, test etc.
	Name string

	// Type is one of the allowed types for a flag.
	// Allowed types are defined as constants shown above.
	Type string

	// Short is a shorthand for the flag.
	// A shorthand must be single char.
	// Shorthand cannot be something that is already defined globally.
	// Shorthand cannot also be something defined as persistent in parent commands.
	// For instance, for pxctl CLI, -j is already defined globally.
	Short string

	// Use is a single line short description for the flag usage.
	Use string

	// Default value for the flag.
	Default string

	// Hidden indicates if this flag is hidden from CLI view, but still functional.
	Hidden bool

	// Persistent makes a flag globally accessible.
	Persistent bool

	// Required enforces flag value to be entered on CLI.
	Required bool

	// Func is a validation func (not yet implemented)
	Func string
}

type flagStub struct {
	Key        string
	Name       string
	VarName    string
	Type       string
	Persistent bool
}

// cmdSpec defines an individual command.
type cmdSpec struct {
	// Name of the command. Use simple names with no whitespaces.
	// Examples: my-command, test etc.
	Name string

	// varName is internal representation of this command.
	varName string

	// Short is a one-line short description for this command.
	Short string

	// Long is a multi-line long description for this command.
	Long string

	// Func is a registered function that should execute for this command.
	// For consistency all registered functions are defined in pkg cli in exec.go.
	Func string

	// InputInterface indicates if flag access should be via cflags interface.
	// This will be deprecated, so do not use.
	InputInterface bool

	// Imports is the pkg import string for the registered func.
	Imports string

	// Aliases for this command.
	Aliases []string

	// Hidden indicates if the command is hidden from CLI view but functional.
	Hidden bool

	// Flags contains a list of flags associated with this command.
	Flags []*flagSpec

	// SubCmd contains a list of sub-commands for this command.
	SubCmd []*cmdSpec
}
