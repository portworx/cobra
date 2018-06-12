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

	"strconv"
	"strings"

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
			if err := add("", "/", command); err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(genCmd)
	genCmd.Flags().StringVarP(&yamlSpecFile, "file", "f", "", "YAML spec file path")
}

func add(parent, keyPath string, cmd *cmdSpec) error {
	if cmd == nil {
		return nil
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
	if err := createCmdFileWithAdditionalData(project.License(), cmdPath, parent, keyPath, cmd); err != nil {
		return err
	}

	fmt.Fprintln((&cobra.Command{}).OutOrStdout(), cmdName, "created at", cmdPath)

	for _, subCmd := range cmd.SubCmd {
		subCmd := subCmd
		if err := add(cmd.Name+"Cmd", filepath.Join(keyPath, cmd.Name), subCmd); err != nil {
			return err
		}
	}
	return nil
}

func createCmdFileWithAdditionalData(license License, path, parent, keyPath string, cmd *cmdSpec) error {
	template := `{{comment .copyright}}
{{if .license}}{{comment .license}}{{end}}

// this file is auto-generated. Please DO NOT EDIT

// package {{.cmdPackage}} has CLI command implementations
package {{.cmdPackage}}

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/portworx/porx/px/cli/cflags"
	{{ if ne .imports "" }}"{{.imports}}"{{ end }}
)

// all content below this line is auto-generated. Please DO NOT EDIT

// {{.cmdVarName}}Cmd represents the {{.cmdName}} command
var {{.cmdVarName}}Cmd = &cobra.Command{
	Use:   "{{.cmdName}}",
	Short: "{{.short}}",
	Long: ` + "`" + `{{.long}}` + "`" + `,
	Aliases: []string{ {{ range $key, $value := .aliases }}"{{ $value }}",{{ end }} },
	Hidden: {{.hidden}},
	RunE: {{.cmdVarName}}Func,
}

func {{.cmdVarName}}Func(cmd *cobra.Command, args []string) error {
	{{ range $key, $value := .boolFlags -}}
		{{- if eq $value.Persistent true -}}
			vp.BindPFlag("/persistent/{{$value.Name}}", cmd.PersistentFlags().Lookup("{{$value.Name}}"));
			vp.SetDefault("/persistent/{{$value.Name}}", {{$value.Default}});
		{{- else -}}
			vp.BindPFlag("{{$.keyPath}}/local/{{$value.Name}}", cmd.Flags().Lookup("{{$value.Name}}"));
			vp.SetDefault("{{$.keyPath}}/local/{{$value.Name}}", {{$value.Default}});
		{{- end }}
	{{- end }}
	{{- range $key, $value := .strFlags -}}
		{{- if eq $value.Persistent true -}}
			vp.BindPFlag("/persistent/{{$value.Name}}", cmd.PersistentFlags().Lookup("{{$value.Name}}"));
			vp.SetDefault("/persistent/{{$value.Name}}", "{{$value.Default}}");
		{{- else -}}
			vp.BindPFlag("{{$.keyPath}}/local/{{$value.Name}}", cmd.Flags().Lookup("{{$value.Name}}"));
			vp.SetDefault("{{$.keyPath}}/local/{{$value.Name}}", "{{$value.Default}}");
		{{- end }}
	{{- end }}
	{{- range $key, $value := .intFlags -}}
		{{- if eq $value.Persistent true -}}
			vp.BindPFlag("/persistent/{{$value.Name}}", cmd.PersistentFlags().Lookup("{{$value.Name}}"));
			vp.SetDefault("/persistent/{{$value.Name}}", {{$value.Default}});
		{{- else -}}
			vp.BindPFlag("{{$.keyPath}}/local/{{$value.Name}}", cmd.Flags().Lookup("{{$value.Name}}"));
			vp.SetDefault("{{$.keyPath}}/local/{{$value.Name}}", {{$value.Default}});
		{{- end }}
	{{- end }}
	{{- range $key, $value := .uintFlags -}}
		{{- if eq $value.Persistent true -}}
			vp.BindPFlag("/persistent/{{$value.Name}}", cmd.PersistentFlags().Lookup("{{$value.Name}}"));
			vp.SetDefault("/persistent/{{$value.Name}}", {{$value.Default}});
		{{- else -}}
			vp.BindPFlag("{{$.keyPath}}/local/{{$value.Name}}", cmd.Flags().Lookup("{{$value.Name}}"));
			vp.SetDefault("{{$.keyPath}}/local/{{$value.Name}}", {{$value.Default}});
		{{- end }}
	{{- end }}
	{{- range $key, $value := .strSliceFlags -}}
		{{- if eq $value.Persistent true -}}
			vp.BindPFlag("/persistent/{{$value.Name}}", cmd.PersistentFlags().Lookup("{{$value.Name}}"));
			vp.SetDefault("/persistent/{{$value.Name}}", nil);
		{{- else -}}
			vp.BindPFlag("{{$.keyPath}}/local/{{$value.Name}}", cmd.Flags().Lookup("{{$value.Name}}"));
			vp.SetDefault("{{$.keyPath}}/local/{{$value.Name}}", nil);
		{{- end }}
	{{- end }}
	{{- range $key, $value := .intSliceFlags -}}
		{{- if eq $value.Persistent true -}}
			vp.BindPFlag("/persistent/{{$value.Name}}", cmd.PersistentFlags().Lookup("{{$value.Name}}"));
			vp.SetDefault("/persistent/{{$value.Name}}", nil);
		{{- else -}}
			vp.BindPFlag("{{$.keyPath}}/local/{{$value.Name}}", cmd.Flags().Lookup("{{$value.Name}}"));
			vp.SetDefault("{{$.keyPath}}/local/{{$value.Name}}", nil);
		{{- end }}
	{{- end }}

		provider, err := cflags.NewViperProvider(cmd, vp, "{{$.keyPath}}/local")
		if err != nil {
			return err
		}
		
		{{ if eq .func "" -}}
			_ = provider
			// enter your exec func here
			// return yourExecFunc(provider)
			fmt.Println("{{.cmdName}} called")
			return nil
		{{- else -}}
			return {{.func}}(provider)
		{{- end }}
}

func init() {
	{{.parentName}}.AddCommand({{.cmdVarName}}Cmd)

	// these flags are auto-generated, please DO NOT EDIT


	{{ range $key, $value := .boolFlags -}}
		{{- if eq $value.Persistent true -}}
			{{- if eq $value.Short "" -}}
				{{$.cmdVarName}}Cmd.PersistentFlags().Bool("{{$value.Name}}", {{$value.Default}}, "{{$value.Use}}");
			{{- else -}}
				{{$.cmdVarName}}Cmd.PersistentFlags().BoolP("{{$value.Name}}", "{{$value.Short}}", {{$value.Default}}, "{{$value.Use}}");
			{{- end }}
			{{- if eq $value.Hidden true -}}
				{{$.cmdVarName}}Cmd.PersistentFlags().MarkHidden("{{$value.Name}}");
			{{- end }}
		{{- else -}}
			{{- if eq $value.Short "" -}}
				{{$.cmdVarName}}Cmd.Flags().Bool("{{$value.Name}}", {{$value.Default}}, "{{$value.Use}}");
			{{- else -}}
				{{$.cmdVarName}}Cmd.Flags().BoolP("{{$value.Name}}", "{{$value.Short}}", {{$value.Default}}, "{{$value.Use}}");
			{{- end }}
			{{- if eq $value.Hidden true -}}
				{{$.cmdVarName}}Cmd.Flags().MarkHidden("{{$value.Name}}");
			{{- end }}
		{{- end }}
	{{- end }}
	{{- range $key, $value := .strFlags -}}
		{{- if eq $value.Persistent true -}}
			{{- if eq $value.Short "" -}}
				{{$.cmdVarName}}Cmd.PersistentFlags().String("{{$value.Name}}", "{{$value.Default}}", "{{$value.Use}}");
			{{- else -}}
				{{$.cmdVarName}}Cmd.PersistentFlags().StringP("{{$value.Name}}", "{{$value.Short}}", "{{$value.Default}}", "{{$value.Use}}");
			{{- end }}
			{{- if eq $value.Hidden true -}}
				{{$.cmdVarName}}Cmd.PersistentFlags().MarkHidden("{{$value.Name}}");
			{{- end }}
		{{- else -}}
			{{- if eq $value.Short "" -}}
				{{$.cmdVarName}}Cmd.Flags().String("{{$value.Name}}", "{{$value.Default}}", "{{$value.Use}}");
			{{- else -}}
				{{$.cmdVarName}}Cmd.Flags().StringP("{{$value.Name}}", "{{$value.Short}}", "{{$value.Default}}", "{{$value.Use}}");
			{{- end }}
			{{- if eq $value.Hidden true -}}
				{{$.cmdVarName}}Cmd.Flags().MarkHidden("{{$value.Name}}");
			{{- end }}
		{{- end }}
	{{- end }}
	{{- range $key, $value := .intFlags -}}
		{{- if eq $value.Persistent true -}}
			{{- if eq $value.Short "" -}}
				{{$.cmdVarName}}Cmd.PersistentFlags().Int("{{$value.Name}}", {{$value.Default}}, "{{$value.Use}}");
			{{- else -}}
				{{$.cmdVarName}}Cmd.PersistentFlags().IntP("{{$value.Name}}", "{{$value.Short}}", {{$value.Default}}, "{{$value.Use}}");
			{{- end }}
			{{- if eq $value.Hidden true -}}
				{{$.cmdVarName}}Cmd.PersistentFlags().MarkHidden("{{$value.Name}}");
			{{- end }}
		{{- else -}}
			{{- if eq $value.Short "" -}}
				{{$.cmdVarName}}Cmd.Flags().Int("{{$value.Name}}", {{$value.Default}}, "{{$value.Use}}");
			{{- else -}}
				{{$.cmdVarName}}Cmd.Flags().IntP("{{$value.Name}}", "{{$value.Short}}", {{$value.Default}}, "{{$value.Use}}");
			{{- end }}
			{{- if eq $value.Hidden true -}}
				{{$.cmdVarName}}Cmd.Flags().MarkHidden("{{$value.Name}}");
			{{- end }}
		{{- end }}
	{{- end }}
	{{- range $key, $value := .uintFlags -}}
		{{- if eq $value.Persistent true -}}
			{{- if eq $value.Short "" -}}
				{{$.cmdVarName}}Cmd.PersistentFlags().Uint("{{$value.Name}}", {{$value.Default}}, "{{$value.Use}}");
			{{- else -}}
				{{$.cmdVarName}}Cmd.PersistentFlags().UintP("{{$value.Name}}", "{{$value.Short}}", {{$value.Default}}, "{{$value.Use}}");
			{{- end }}
			{{- if eq $value.Hidden true -}}
				{{$.cmdVarName}}Cmd.PersistentFlags().MarkHidden("{{$value.Name}}");
			{{- end }}
		{{- else -}}
			{{- if eq $value.Short "" -}}
				{{$.cmdVarName}}Cmd.Flags().Uint("{{$value.Name}}", {{$value.Default}}, "{{$value.Use}}");
			{{- else -}}
				{{$.cmdVarName}}Cmd.Flags().UintP("{{$value.Name}}", "{{$value.Short}}", {{$value.Default}}, "{{$value.Use}}");
			{{- end }}
			{{- if eq $value.Hidden true -}}
				{{$.cmdVarName}}Cmd.Flags().MarkHidden("{{$value.Name}}");
			{{- end }}
		{{- end }}
	{{- end }}
	{{- range $key, $value := .strSliceFlags -}}
		{{- if eq $value.Persistent true -}}
			{{- if eq $value.Short "" -}}
				{{$.cmdVarName}}Cmd.PersistentFlags().StringSlice("{{$value.Name}}", nil, "{{$value.Use}}");
			{{- else -}}
				{{$.cmdVarName}}Cmd.PersistentFlags().StringSliceP("{{$value.Name}}", "{{$value.Short}}", nil, "{{$value.Use}}");
			{{- end }}
			{{- if eq $value.Hidden true -}}
				{{$.cmdVarName}}Cmd.PersistentFlags().MarkHidden("{{$value.Name}}");
			{{- end }}
		{{- else -}}
			{{- if eq $value.Short "" -}}
				{{$.cmdVarName}}Cmd.Flags().StringSlice("{{$value.Name}}", nil, "{{$value.Use}}");
			{{- else -}}
				{{$.cmdVarName}}Cmd.Flags().StringSliceP("{{$value.Name}}", "{{$value.Short}}", nil, "{{$value.Use}}");
			{{- end }}
			{{- if eq $value.Hidden true -}}
				{{$.cmdVarName}}Cmd.Flags().MarkHidden("{{$value.Name}}");
			{{- end }}
		{{- end }}
	{{- end }}
	{{- range $key, $value := .intSliceFlags -}}
		{{- if eq $value.Persistent true -}}
			{{- if eq $value.Short "" -}}
				{{$.cmdVarName}}Cmd.PersistentFlags().IntSlice("{{$value.Name}}", nil, "{{$value.Use}}");
			{{- else -}}
				{{$.cmdVarName}}Cmd.PersistentFlags().IntSliceP("{{$value.Name}}", "{{$value.Short}}", nil, "{{$value.Use}}");
			{{- end }}
			{{- if eq $value.Hidden true -}}
				{{$.cmdVarName}}Cmd.PersistentFlags().MarkHidden("{{$value.Name}}");
			{{- end }}
		{{- else -}}
			{{- if eq $value.Short "" -}}
				{{$.cmdVarName}}Cmd.Flags().IntSlice("{{$value.Name}}", nil, "{{$value.Use}}");
			{{- else -}}
				{{$.cmdVarName}}Cmd.Flags().IntSliceP("{{$value.Name}}", "{{$value.Short}}", nil, "{{$value.Use}}");
			{{- end }}
			{{- if eq $value.Hidden true -}}
				{{$.cmdVarName}}Cmd.Flags().MarkHidden("{{$value.Name}}");
			{{- end }}
		{{- end }}
	{{- end }}
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
	data["cmdVarName"] = validateCmdName(cmd.Name)
	data["short"] = cmd.Short
	data["long"] = cmd.Long
	data["func"] = cmd.Func
	data["imports"] = cmd.Imports
	data["aliases"] = cmd.Aliases
	data["hidden"] = cmd.Hidden

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
	uintFlags := make([]*flagSpec, 0, 0)
	strSliceFlags := make([]*flagSpec, 0, 0)
	intSliceFlag := make([]*flagSpec, 0, 0)
	for _, flag := range cmd.Flags {
		flag := flag
		switch flag.Type {
		case FlagBool:
			if flag.Default == "" {
				flag.Default = "false"
			} else {
				switch strings.ToLower(flag.Default) {
				case "false":
					flag.Default = "false"
				case "true":
					flag.Default = "true"
				default:
					return fmt.Errorf("error parsing YAML, invalid default value for bool type %s", flag.Name)
				}
			}
			boolFlags = append(boolFlags, flag)
		case FlagStr:
			strFlags = append(strFlags, flag)
		case FlagInt:
			if _, err := strconv.ParseInt(flag.Default, 10, 32); err != nil {
				return fmt.Errorf("error parsing YAML, invalid default value for int type %s", flag.Name)
			}
			intFlags = append(intFlags, flag)
		case FlagUint:
			if _, err := strconv.ParseUint(flag.Default, 10, 32); err != nil {
				return fmt.Errorf("error parsing YAML, invalid default value for uint type %s", flag.Name)
			}
			uintFlags = append(uintFlags, flag)
		case FlagStrSlice:
			if flag.Default != "" {
				return fmt.Errorf("error parsing YAML, default value not supported for string slice type %s", flag.Name)
			}
			strSliceFlags = append(strSliceFlags, flag)
		case FlagIntSlice:
			if flag.Default != "" {
				return fmt.Errorf("error parsing YAML, default value not supported for int slice type %s", flag.Name)
			}
			intSliceFlag = append(intSliceFlag, flag)
		default:
			return fmt.Errorf("invalid flag type. Valid types: %s, %s, %s, %s, %s",
				FlagBool, FlagStr, FlagInt, FlagStrSlice, FlagIntSlice)
		}
	}

	if keyPath != "/" {
		data["keyPath"] = keyPath
	} else {
		data["keyPath"] = ""
	}

	data["boolFlags"] = boolFlags
	data["intFlags"] = intFlags
	data["uintFlags"] = uintFlags
	data["strFlags"] = strFlags
	data["strSliceFlag"] = strSliceFlags
	data["intSliceFlag"] = intSliceFlag

	cmdScript, err := executeTemplate(template, data)
	if err != nil {
		er(err)
	}
	err = writeStringToFile(path, cmdScript)
	if err != nil {
		er(err)
	}
	return nil
}
