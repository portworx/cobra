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
	Name       string
	Type       string
	Short      string
	Use        string
	Default    string
	Hidden     bool
	Persistent bool
	Required   bool
	Func       string
}

type flagStub struct {
	Key        string
	Name       string
	VarName    string
	Type       string
	Persistent bool
}

type cmdSpec struct {
	Name           string
	varName        string
	Short          string
	Long           string
	Func           string
	InputInterface bool
	Imports        string
	Aliases        []string
	Hidden         bool
	Flags          []*flagSpec
	SubCmd         []*cmdSpec
}

var (
	yamlSpecFile string
	commands     []*cmdSpec
)
