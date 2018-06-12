package cmd

const (
	FlagBool     = "bool"
	FlagStr      = "str"
	FlagInt      = "int"
	FlagUint     = "uint"
	FlagStrSlice = "str-slice"
	FlagIntSlice = "int-slice"
)

type flagSpec struct {
	Type       string
	Short      string
	Name       string
	Use        string
	Default    string
	Hidden     bool
	Persistent bool
}

type cmdSpec struct {
	Name    string
	Short   string
	Long    string
	Func    string
	Imports string
	Aliases []string
	Hidden  bool
	Flags   []*flagSpec
	SubCmd  []*cmdSpec
}

var (
	yamlSpecFile string
	commands     []*cmdSpec
)
