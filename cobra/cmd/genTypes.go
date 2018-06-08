package cmd

const (
	FlagBool = "bool"
	FlagStr  = "str"
	FlagInt  = "int"
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
	Name   string
	Short  string
	Long   string
	Func   string
	Flags  []*flagSpec
	SubCmd []*cmdSpec
}

var (
	yamlSpecFile string
	commands     []*cmdSpec
)
