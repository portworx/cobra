// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"

	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// genCmd represents the gen command
var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "auto-gen CLI using YAML spec",
	Long: `This command takes a YAML description of CLI as input and
produces boilerplate code for CLI.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fileName := cmd.Flag("file").Value.String()

		// read yaml file
		yb, err := ioutil.ReadFile(fileName)
		if err != nil {
			return err
		}
		commands = make([]*cmdSpec, 0, 0)
		if err := yaml.Unmarshal(yb, &commands); err != nil {
			return err
		}

		// iterate over commands and add these commands
		for _, command := range commands {
			command := command
			add("", command)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(genCmd)
	genCmd.Flags().StringVarP(&yamlSpecFile, "file", "f", "", "YAML spec file path")
}

func add(parent string, cmd *cmdSpec) {
	if cmd == nil {
		return
	}

	var project *Project
	if packageName != "" {
		project = NewProject(packageName)
	} else {
		wd, err := os.Getwd()
		if err != nil {
			er(err)
		}
		project = NewProjectFromPath(wd)
	}

	cmdName := validateCmdName(cmd.Name)
	cmdPath := filepath.Join(project.CmdPath(), cmdName+".go")
	createCmdFileWithAdditionalData(project.License(), cmdPath, parent, cmd)

	fmt.Fprintln((&cobra.Command{}).OutOrStdout(), cmdName, "created at", cmdPath)

	for _, subCmd := range cmd.SubCmd {
		subCmd := subCmd
		add(cmd.Name+"Cmd", subCmd)
	}
}

func createCmdFileWithAdditionalData(license License, path, parent string, cmd *cmdSpec) {
	template := `{{comment .copyright}}
{{if .license}}{{comment .license}}{{end}}

package {{.cmdPackage}}

import (
	"fmt"

	"github.com/spf13/cobra"
)

// {{.cmdName}}Cmd represents the {{.cmdName}} command
var {{.cmdName}}Cmd = &cobra.Command{
	Use:   "{{.cmdName}}",
	Short: "{{.short}}",
	Long: ` + "`" + `{{.long}}` + "`" + `,
	{{ if ne .func "" }}Run: {{.func}},{{ end }}
}

{{ if ne .func "" }}
func {{.func}}(cmd *cobra.Command, args []string) {
		fmt.Println("{{.cmdName}} called")
}
{{ end }}

func init() {
	{{.parentName}}.AddCommand({{.cmdName}}Cmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// {{.cmdName}}Cmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// {{.cmdName}}Cmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	{{ range $key, $value := .boolFlags }}
	{{$.cmdName}}Cmd.Flags().BoolP("{{$value.Long}}", "{{$value.Short}}", false, "{{$value.Use}}")
	{{ end }}
	{{ range $key, $value := .strFlags }}
	{{$.cmdName}}Cmd.Flags().StringP("{{$value.Long}}", "{{$value.Short}}", "", "{{$value.Use}}")
	{{ end }}
	{{ range $key, $value := .intFlags }}
	{{$.cmdName}}Cmd.Flags().IntP("{{$value.Long}}", "{{$value.Short}}", 0, "{{$value.Use}}")
	{{ end }}
}
`

	data := make(map[string]interface{})
	data["copyright"] = copyrightLine()
	data["license"] = license.Header
	data["cmdPackage"] = filepath.Base(filepath.Dir(path)) // last dir of path
	if parent == "" {
		data["parentName"] = parentName
	} else {
		data["parentName"] = parent
	}
	data["cmdName"] = cmd.Name
	data["short"] = cmd.Short
	data["long"] = cmd.Long
	data["func"] = cmd.Func

	if data["short"] == "" {
		data["short"] = "A brief description of your command"
	}

	if data["long"] == "" {
		data["long"] = `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`
	}

	boolFlags := make([]*flagSpec, 0, 0)
	strFlags := make([]*flagSpec, 0, 0)
	intFlags := make([]*flagSpec, 0, 0)
	for _, flag := range cmd.Flags {
		flag := flag
		switch flag.Type {
		case FlagBool:
			boolFlags = append(boolFlags, flag)
		case FlagStr:
			strFlags = append(strFlags, flag)
		case FlagInt:
			intFlags = append(intFlags, flag)
		}
	}

	data["boolFlags"] = boolFlags
	data["intFlags"] = intFlags
	data["strFlags"] = strFlags

	cmdScript, err := executeTemplate(template, data)
	if err != nil {
		er(err)
	}
	err = writeStringToFile(path, cmdScript)
	if err != nil {
		er(err)
	}
}
