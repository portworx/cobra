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
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	// cmdNames is a bookkeeping map to keep track of names that might be already in use.
	// Such collisions need to be prevented for uniqueness of filenames.
	cmdNames map[string]bool

	// commandBuffer stores exec command stubs that get dumped in cli package for the
	// commands that do not have a valid action listed against them
	commandBuffer bytes.Buffer

	// commandWriter wraps commandBuffer
	commandWriter *bufio.Writer

	// symbolBuffer stores symbol table for use in conjunction with cflags.Provider interface.
	// It dumps symbol table that provide a way to lookup flagnames during access.
	symbolBuffer bytes.Buffer

	// symbolWriter wraps symbolBuffer
	symbolWriter *bufio.Writer
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

		commandWriter.Flush()
		symbolWriter.Flush()
		f := make(map[string]interface{})
		f["functions"] = string(commandBuffer.Bytes())
		f["constants"] = string(symbolBuffer.Bytes())

		if t, err := ioutil.ReadFile("templates/pxFunctions.tmpl"); err != nil {
			return err
		} else {
			if b, err := executeTemplate(string(t), f); err != nil {
				return err
			} else {
				outFile := filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "portworx", "porx", "px", "cli", "exec.go")
				if err := ioutil.WriteFile(outFile, []byte(b), 0644); err != nil {
					return err
				}
			}
		}

		if t, err := ioutil.ReadFile("templates/pxSymbols.tmpl"); err != nil {
			return err
		} else {
			if b, err := executeTemplate(string(t), f); err != nil {
				return err
			} else {
				outFile := filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "portworx", "porx", "px", "cli", "symbols.go")
				if err := ioutil.WriteFile(outFile, []byte(b), 0644); err != nil {
					return err
				}
			}
		}

		return nil
	},
}

func init() {
	// initialize global vars
	cmdNames = make(map[string]bool)
	commandWriter = bufio.NewWriter(&commandBuffer)
	symbolWriter = bufio.NewWriter(&symbolBuffer)

	rootCmd.AddCommand(genCmd)
	genCmd.Flags().StringVarP(&yamlSpecFile, "file", "f", "", "YAML spec file path")
}

// add is a recursive func that sifts through YAML spec and generates one go file per command.
// All commands are dumped in the same folder. Nesting of commands is handled by linking pointers.
func add(parent, keyPath string, cmd *cmdSpec) error {
	// recursion exits on nil
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
	commandName := cmdName
	// loop till we find a command name that is not taken
	for i := 0; ; i++ {
		if _, ok := cmdNames[commandName]; !ok {
			break
		} else {
			commandName = fmt.Sprintf("%s%d", cmdName, i)
		}
	}
	cmdName = commandName

	// make entry of this command name so future command names could be checked
	cmdNames[cmdName] = true

	// use the generated name as varName for this command
	cmd.varName = cmdName

	cmdPath := filepath.Join(project.CmdPath(), "auto_"+cmdName+".go")
	if err := os.RemoveAll(cmdPath); err != nil {
		return err
	}

	if err := createCmdFileWithAdditionalData(project.License(), cmdPath, parent, keyPath, cmd); err != nil {
		return err
	}

	fmt.Fprintln((&cobra.Command{}).OutOrStdout(), cmdName, "created at", cmdPath)

	// run recursion for subcommands
	for _, subCmd := range cmd.SubCmd {
		subCmd := subCmd
		if err := add(cmd.varName+"Cmd", filepath.Join(keyPath, cmd.Name), subCmd); err != nil {
			return err
		}
	}
	return nil
}

func createCmdFileWithAdditionalData(license License, path, parent, keyPath string, cmd *cmdSpec) error {
	data := make(map[string]interface{})

	// parent is the var name of the parent command struct to link against.
	// there is always a parent available.
	if parent == "" {
		// if empty string is passed, it implies we need to link against "root"
		data["parentName"] = parentName
	} else {
		data["parentName"] = parent
	}

	data["copyright"] = copyrightLine()
	data["license"] = license.Header
	data["cmdPackage"] = filepath.Base(filepath.Dir(path)) // last dir of path
	data["cmdName"] = cmd.Name
	data["cmdVarName"] = cmd.varName
	data["imports"] = cmd.Imports
	data["aliases"] = cmd.Aliases
	data["hidden"] = cmd.Hidden

	if cmd.Short == "" {
		data["short"] = cmd.Name
	} else {
		data["short"] = cmd.Short
	}

	if cmd.Long == "" {
		data["long"] = cmd.Name
	} else {
		data["long"] = cmd.Long
	}

	if keyPath != "/" {
		data["keyPath"] = keyPath
	} else {
		data["keyPath"] = ""
	}

	keys := strings.Split(strings.Replace(data["keyPath"].(string), "/", "-", -1), "-")
	execFunc := "xec"
	for _, key := range keys {
		key := strings.TrimSpace(key)
		if len(key) > 0 {
			if len(key) == 1 {
				key = strings.ToUpper(string(key[0]))
				execFunc = execFunc + key
			} else {
				key = strings.ToUpper(string(key[0])) + key[1:]
				execFunc = execFunc + key
			}
		}
	}
	data["func"] = "cli.E" + execFunc + formatInput(cmd.Name)
	data["funcInCli"] = "E" + execFunc + formatInput(cmd.Name)
	data["localFunc"] = "e" + execFunc + formatInput(cmd.Name)

	if len(cmd.Func) > 0 {
		data["func"] = cmd.Func
	}

	boolFlags := make([]*flagSpec, 0, 0)
	strFlags := make([]*flagSpec, 0, 0)
	intFlags := make([]*flagSpec, 0, 0)
	uintFlags := make([]*flagSpec, 0, 0)
	strSliceFlags := make([]*flagSpec, 0, 0)
	intSliceFlag := make([]*flagSpec, 0, 0)
	for _, flag := range cmd.Flags {
		flag := flag
		s := fmt.Sprintf("%s%s = \"%s\"\n",
			strings.Replace(data["localFunc"].(string), "exec", "flag", -1),
			formatInput(flag.Name), flag.Name)
		symbolWriter.Write([]byte(s))
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
			return fmt.Errorf("invalid flag type. Valid types: %s, %s, %s, %s, %s, %s",
				FlagBool, FlagStr, FlagInt, FlagUint, FlagStrSlice, FlagIntSlice)
		}
	}

	data["boolFlags"] = boolFlags
	data["intFlags"] = intFlags
	data["uintFlags"] = uintFlags
	data["strFlags"] = strFlags
	data["strSliceFlag"] = strSliceFlags
	data["intSliceFlag"] = intSliceFlag

	// dump a go file for new command
	if t, err := ioutil.ReadFile("templates/pxCommand.tmpl"); err != nil {
		return err
	} else {
		if b, err := executeTemplate(string(t), data); err != nil {
			er(err)
		} else {
			if err = writeStringToFile(path, b); err != nil {
				er(err)
			}
		}
	}

	// dump a stub for exec func if user does not provide one in YAML
	if len(cmd.Func) == 0 {
		if t, err := ioutil.ReadFile("templates/pxFunction.tmpl"); err != nil {
			return err
		} else {
			if b, err := executeTemplate(string(t), data); err != nil {
				er(err)
			} else {
				commandWriter.Write([]byte(b))
			}
		}
	}

	return nil
}

// formatInput is a helper func
func formatInput(x string) string {
	if len(x) == 0 {
		return x
	}

	if len(x) == 1 {
		return strings.ToUpper(x)
	}

	x = strings.Replace(x, "_", "-", -1)
	out := ""
	for _, key := range strings.Split(x, "-") {
		if len(key) == 1 {
			key = strings.ToUpper(key)
		} else {
			key = strings.ToUpper(string(key[0])) + key[1:]
		}
		out = out + key
	}
	return out
}
