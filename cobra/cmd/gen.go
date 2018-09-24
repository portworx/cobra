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
	"log"
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

	// execFolder is the path of dir where this executable is running.
	// It is used to locate template files relative to this path.
	execFolder string

	// commandBuffer stores exec command stubs that get dumped in cli package for the
	// commands that do not have a valid action listed against them.
	commandBuffer bytes.Buffer

	// commandWriter wraps commandBuffer.
	commandWriter *bufio.Writer

	// configBuffer stores yaml config file that could be used as input for cli being built.
	configBuffer bytes.Buffer

	// configWriter wraps configBuffer.
	configWriter *bufio.Writer

	// structBuffer stores new structs for flag values
	structBuffer bytes.Buffer

	// structWriter wraps structBuffer
	structWriter *bufio.Writer

	// symbolBuffer stores symbol table for use in conjunction with cflags.Provider interface.
	// It dumps symbol table that provide a way to lookup flagnames during access.
	symbolBuffer bytes.Buffer

	// symbolWriter wraps symbolBuffer.
	symbolWriter *bufio.Writer

	// testBuffer stores yaml config file that could be used as input for cli being built.
	testBuffer bytes.Buffer

	// testWriter wraps configBuffer.
	testWriter *bufio.Writer

	// typeList simply holds list of declared types are strings
	typeList []string
)

// genCmd represents the gen command.
var genCmd = &cobra.Command{
	Use:     "gen",
	Example: "cobra gen -f file1.yaml [file2.yaml [file3.yaml]]",
	Short:   "auto-gen CLI using YAML spec",
	Long: `This command takes a YAML description of CLI as input and
produces boilerplate code for CLI.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		yamlfiles := make([]string, 0, 0)
		if cmd.Flag("file").Changed {
			fileName := cmd.Flag("file").Value.String()
			yamlfiles = append(yamlfiles, fileName)
		}
		yamlfiles = append(yamlfiles, args...)

		commands = make([]*cmdSpec, 0, 0)

		for _, fileName := range yamlfiles {
			// read yaml file
			yb, err := ioutil.ReadFile(fileName)
			if err != nil {
				return err
			}
			cmds := make([]*cmdSpec, 0, 0)
			if err := yaml.Unmarshal(yb, &cmds); err != nil {
				return err
			}

			commands = append(commands, cmds...)
		}

		for _, c := range commands {
			if c.Name == "cluster" {
				for _, d := range c.SubCmd {
					if d.Name == "config" {
						setInputInterface(d)
						break
					}
				}
				break
			}
		}

		// remove all auto-generated files from command path
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
		files, err := ioutil.ReadDir(project.CmdPath())
		if err != nil {
			return err
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			fileName := file.Name()
			if len(fileName) > 5 && fileName[:5] == "auto_" {
				if err := os.RemoveAll(filepath.Join(project.CmdPath(), fileName)); err != nil {
					return err
				} else {
					fmt.Println("removed:", filepath.Join(project.CmdPath(), fileName))
				}
			}
		}

		// iterate over commands and add these commands
		for _, command := range commands {
			command := command
			if err := add("", "/", command); err != nil {
				return err
			}
		}

		commandWriter.Flush()
		configWriter.Flush()
		symbolWriter.Flush()
		structWriter.Flush()
		testWriter.Flush()
		f := make(map[string]interface{})
		f["functions"] = string(commandBuffer.Bytes())
		f["constants"] = string(symbolBuffer.Bytes())
		f["structs"] = string(structBuffer.Bytes())
		f["testFuncs"] = string(testBuffer.Bytes())
		f["typeList"] = typeList

		if t, err := ioutil.ReadFile(filepath.Join(execFolder, "templates", "pxFunctions.tmpl")); err != nil {
			return err
		} else {
			if b, err := executeTemplate(string(t), f); err != nil {
				return err
			} else {
				outFile := filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "portworx", "porx", "px", "cli", "execStubs.go")
				if err := ioutil.WriteFile(outFile, []byte(b), 0644); err != nil {
					return err
				}
			}
		}

		if t, err := ioutil.ReadFile(filepath.Join(execFolder, "templates", "pxStructs.tmpl")); err != nil {
			return err
		} else {
			if b, err := executeTemplate(string(t), f); err != nil {
				return err
			} else {
				outFile := filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "portworx", "porx", "px", "cli", "types.go")
				if err := ioutil.WriteFile(outFile, []byte(b), 0644); err != nil {
					return err
				}
			}
		}

		if err := os.MkdirAll("test", 0755); err != nil {
			return err
		}
		if t, err := ioutil.ReadFile(filepath.Join(execFolder, "templates", "pxTests.tmpl")); err != nil {
			return err
		} else {
			if b, err := executeTemplate(string(t), f); err != nil {
				return err
			} else {
				outFile := filepath.Join("test", "pxctl_test.go")
				if err := ioutil.WriteFile(outFile, []byte(b), 0644); err != nil {
					return err
				}
			}
		}

		return ioutil.WriteFile("config.yaml", configBuffer.Bytes(), 0644)
	},
}

func setInputInterface(cmd *cmdSpec) {
	cmd.InputInterface = true
	if len(cmd.SubCmd) > 0 {
		for _, c := range cmd.SubCmd {
			setInputInterface(c)
		}
	}
}

func init() {
	// initialize global vars
	cmdNames = make(map[string]bool)
	commandWriter = bufio.NewWriter(&commandBuffer)
	symbolWriter = bufio.NewWriter(&symbolBuffer)
	structWriter = bufio.NewWriter(&structBuffer)
	configWriter = bufio.NewWriter(&configBuffer)
	testWriter = bufio.NewWriter(&testBuffer)

	// init executable path
	if execPath, err := os.Executable(); err != nil {
		log.Fatal(err)
	} else {
		execFolder, _ = filepath.Split(execPath)
	}

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
	data["example"] = cmd.Example

	if cmd.Short == "" {
		data["short"] = cmd.Name
	} else {
		data["short"] = cmd.Short
	}

	if cmd.Long == "" {
		data["long"] = cmd.Short
	} else {
		data["long"] = cmd.Long
	}

	if keyPath != "/" {
		data["keyPath"] = keyPath
	} else {
		data["keyPath"] = ""
	}

	data["inputInterface"] = cmd.InputInterface
	data["argsRequired"] = cmd.ArgsRequired

	// build standard ut's
	execCommands := make([]testSpec, 0, 0)
	execCommand := strings.Split(strings.TrimLeft(filepath.Join(keyPath, cmd.Name), "/"), "/")
	execCommands = append(execCommands, testSpec{CommandArgs: execCommand})
	for _, alias := range cmd.Aliases {
		execCommand := strings.Split(strings.TrimLeft(filepath.Join(keyPath, alias), "/"), "/")
		execCommands = append(execCommands, testSpec{CommandArgs: execCommand})
	}

	keys := strings.Split(strings.Replace(data["keyPath"].(string), "/", "-", -1), "-")
	execFunc := ""
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
	data["func"] = "cli.Exec" + execFunc + formatInputUp(cmd.Name)
	data["funcInCli"] = "Exec" + execFunc + formatInputUp(cmd.Name)
	data["localFunc"] = "exec" + execFunc + formatInputUp(cmd.Name)
	data["localStruct"] = "Flags" + execFunc + formatInputUp(cmd.Name)
	typeList = append(typeList, data["localStruct"].(string))

	if len(cmd.Func) > 0 {
		data["func"] = cmd.Func
	}

	boolFlags := make([]*flagSpec, 0, 0)
	strFlags := make([]*flagSpec, 0, 0)
	intFlags := make([]*flagSpec, 0, 0)
	uintFlags := make([]*flagSpec, 0, 0)
	strSliceFlags := make([]*flagSpec, 0, 0)
	intSliceFlags := make([]*flagSpec, 0, 0)

	boolStubs := make([]*flagStub, 0, 0)
	strStubs := make([]*flagStub, 0, 0)
	intStubs := make([]*flagStub, 0, 0)
	uintStubs := make([]*flagStub, 0, 0)
	strSliceStubs := make([]*flagStub, 0, 0)
	intSliceStubs := make([]*flagStub, 0, 0)
	for _, flag := range cmd.Flags {
		flag := flag
		if flag.Required {
			flag.Use = "(Required) " + flag.Use
			for i := range execCommands {
				if len(flag.ValidValues) > 0 {
					execCommands[i].CommandArgs = append(execCommands[i].CommandArgs,
						"--"+flag.Name, flag.ValidValues[0])
				} else if len(flag.ValidRange) > 0 {
					execCommands[i].CommandArgs = append(execCommands[i].CommandArgs,
						"--"+flag.Name, flag.ValidRange[0])

				} else {
					execCommands[i].CommandArgs = append(execCommands[i].CommandArgs,
						"--"+flag.Name, "0")
				}
			}
		}
		s := fmt.Sprintf("%s%s = \"%s\"\n",
			strings.Replace(data["localFunc"].(string), "exec", "flag", -1),
			formatInputUp(flag.Name), flag.Name)
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
			stub := new(flagStub)
			stub.Type = FlagBool
			stub.Name = formatInputUp(flag.Name)
			stub.OriginalName = flag.Name
			stub.Key = flag.Name
			stub.Persistent = flag.Persistent
			stub.VarName = fmt.Sprintf("%s%s",
				strings.Replace(data["localFunc"].(string), "exec", "flag", -1),
				formatInputUp(flag.Name))
			if len(flag.ValidValues) > 0 || len(flag.ValidRange) > 0 || len(flag.ValidatorFunc) > 0 {
				return fmt.Errorf("bool flag cannot have validation checks. pl. fix yaml")
			}
			boolStubs = append(boolStubs, stub)
			boolFlags = append(boolFlags, flag)
		case FlagStr:
			stub := new(flagStub)
			stub.Type = FlagStr
			stub.Name = formatInputUp(flag.Name)
			stub.OriginalName = flag.Name
			stub.Key = flag.Name
			stub.Persistent = flag.Persistent
			stub.VarName = fmt.Sprintf("%s%s",
				strings.Replace(data["localFunc"].(string), "exec", "flag", -1),
				formatInputUp(flag.Name))
			stub.ValidValues = flag.ValidValues
			stub.ValidatorFunc = flag.ValidatorFunc
			validationChecks := 0
			if len(flag.ValidValues) > 0 {
				validationChecks += 1
			}
			if len(flag.ValidRange) > 0 {
				validationChecks += 1
			}
			if len(flag.ValidatorFunc) > 0 {
				validationChecks += 1
			}

			if validationChecks > 1 {
				return fmt.Errorf("please have only one validation method in: %s", flag.Name)
			}

			if len(flag.ValidRange) > 0 {
				return fmt.Errorf("str flag cannot have valid ranges. pl. fix yaml")
			}

			if len(flag.ValidValues) > 0 {
				flag.Use = fmt.Sprintf("%s (Valid Values: %v)", flag.Use, flag.ValidValues)
			}

			strStubs = append(strStubs, stub)
			strFlags = append(strFlags, flag)
		case FlagInt:
			if _, err := strconv.ParseInt(flag.Default, 10, 32); err != nil {
				return fmt.Errorf("error parsing YAML, invalid default value for int type %s", flag.Name)
			}
			stub := new(flagStub)
			stub.Type = FlagInt
			stub.Name = formatInputUp(flag.Name)
			stub.OriginalName = flag.Name
			stub.Key = flag.Name
			stub.Persistent = flag.Persistent
			stub.VarName = fmt.Sprintf("%s%s",
				strings.Replace(data["localFunc"].(string), "exec", "flag", -1),
				formatInputUp(flag.Name))

			if len(flag.ValidRange) > 0 {
				if len(flag.ValidRange) != 2 {
					return fmt.Errorf("range can only contain two values")
				} else {
					if flag.ValidRange[0] > flag.ValidRange[1] {
						return fmt.Errorf("range values not in ascending order %v", flag.ValidRange)
					}
				}
			}

			if err := validateSliceForAType(flag.ValidValues, FlagInt); err != nil {
				return fmt.Errorf("%s %s: %v", flag.Name, "valid values are not valid", err)
			}

			if err := validateSliceForAType(flag.ValidRange, FlagInt); err != nil {
				return fmt.Errorf("%s %s: %v", flag.Name, "valid values are not valid", err)
			}

			stub.ValidValues = flag.ValidValues
			stub.ValidRange = flag.ValidRange
			stub.ValidatorFunc = flag.ValidatorFunc

			validationChecks := 0
			if len(flag.ValidValues) > 0 {
				validationChecks += 1
			}
			if len(flag.ValidRange) > 0 {
				validationChecks += 1
			}
			if len(flag.ValidatorFunc) > 0 {
				validationChecks += 1
			}

			if validationChecks > 1 {
				return fmt.Errorf("please have only one validation method in: %s", flag.Name)
			}

			if len(flag.ValidValues) > 0 {
				flag.Use = fmt.Sprintf("%s (Valid Values: %v)", flag.Use, flag.ValidValues)
			}

			if len(flag.ValidRange) > 0 {
				flag.Use = fmt.Sprintf("%s (Valid Range: %v)", flag.Use, flag.ValidRange)
			}
			intStubs = append(intStubs, stub)
			intFlags = append(intFlags, flag)
		case FlagUint:
			if _, err := strconv.ParseUint(flag.Default, 10, 32); err != nil {
				return fmt.Errorf("error parsing YAML, invalid default value for uint type %s", flag.Name)
			}
			stub := new(flagStub)
			stub.Type = FlagUint
			stub.Name = formatInputUp(flag.Name)
			stub.OriginalName = flag.Name
			stub.Key = flag.Name
			stub.Persistent = flag.Persistent
			stub.VarName = fmt.Sprintf("%s%s",
				strings.Replace(data["localFunc"].(string), "exec", "flag", -1),
				formatInputUp(flag.Name))
			if len(flag.ValidRange) > 0 && len(flag.ValidValues) > 0 {
				return fmt.Errorf("enter either valid values or valid range, not both")
			}
			if len(flag.ValidRange) > 0 {
				if len(flag.ValidRange) != 2 {
					return fmt.Errorf("range can only contain two values")
				} else {
					if flag.ValidRange[0] > flag.ValidRange[1] {
						return fmt.Errorf("range values not in ascending order: %v", flag.ValidRange)
					}
				}
			}

			if err := validateSliceForAType(flag.ValidValues, FlagUint); err != nil {
				return fmt.Errorf("%s %s: %v", flag.Name, "valid values are not valid", err)
			}

			if err := validateSliceForAType(flag.ValidRange, FlagUint); err != nil {
				return fmt.Errorf("%s %s: %v", flag.Name, "valid values are not valid", err)
			}

			stub.ValidValues = flag.ValidValues
			stub.ValidRange = flag.ValidRange
			stub.ValidatorFunc = flag.ValidatorFunc

			validationChecks := 0
			if len(flag.ValidValues) > 0 {
				validationChecks += 1
			}
			if len(flag.ValidRange) > 0 {
				validationChecks += 1
			}
			if len(flag.ValidatorFunc) > 0 {
				validationChecks += 1
			}

			if validationChecks > 1 {
				return fmt.Errorf("please have only one validation method in: %s", flag.Name)
			}

			if len(flag.ValidValues) > 0 {
				flag.Use = fmt.Sprintf("%s (Valid Values: %v)", flag.Use, flag.ValidValues)
			}

			if len(flag.ValidRange) > 0 {
				flag.Use = fmt.Sprintf("%s (Valid Range: %v)", flag.Use, flag.ValidRange)
			}

			uintStubs = append(uintStubs, stub)
			uintFlags = append(uintFlags, flag)
		case FlagStrSlice:
			if flag.Default != "" {
				return fmt.Errorf("error parsing YAML, default value not supported for string slice type %s", flag.Name)
			}
			stub := new(flagStub)
			stub.Type = FlagStrSlice
			stub.Name = formatInputUp(flag.Name)
			stub.OriginalName = flag.Name
			stub.Key = flag.Name
			stub.Persistent = flag.Persistent
			stub.VarName = fmt.Sprintf("%s%s",
				strings.Replace(data["localFunc"].(string), "exec", "flag", -1),
				formatInputUp(flag.Name))

			if len(flag.ValidValues) > 0 || len(flag.ValidRange) > 0 || len(flag.ValidatorFunc) > 0 {
				return fmt.Errorf("validators are not supported in string slice flags: %s", flag.Name)
			}

			strSliceStubs = append(strSliceStubs, stub)
			strSliceFlags = append(strSliceFlags, flag)
		case FlagIntSlice:
			if flag.Default != "" {
				return fmt.Errorf("error parsing YAML, default value not supported for int slice type %s", flag.Name)
			}
			stub := new(flagStub)
			stub.Type = FlagIntSlice
			stub.Name = formatInputUp(flag.Name)
			stub.OriginalName = flag.Name
			stub.Key = flag.Name
			stub.Persistent = flag.Persistent
			stub.VarName = fmt.Sprintf("%s%s",
				strings.Replace(data["localFunc"].(string), "exec", "flag", -1),
				formatInputUp(flag.Name))

			if len(flag.ValidValues) > 0 || len(flag.ValidRange) > 0 || len(flag.ValidatorFunc) > 0 {
				return fmt.Errorf("validators are not supported in string int flags: %s", flag.Name)
			}

			intSliceStubs = append(intSliceStubs, stub)
			intSliceFlags = append(intSliceFlags, flag)
		default:
			return fmt.Errorf("invalid flag type. Valid types: %s, %s, %s, %s, %s, %s",
				FlagBool, FlagStr, FlagInt, FlagUint, FlagStrSlice, FlagIntSlice)
		}
	}

	// add argument
	n := len(execCommands)
	if cmd.ArgsRequired {
		execCommands = append(execCommands, execCommands...)
	}

	if cmd.ArgsRequired {
		for i := range execCommands {
			if i < n {
				execCommands[i].CommandArgs = append(execCommands[i].CommandArgs, "myArg")
			} else {
				execCommands[i].ExpectedToFail = true
			}
		}
	}

	for i := range cmd.Tests {
		execCommand = strings.Split(strings.TrimLeft(filepath.Join(keyPath, cmd.Name), "/"), "/")
		cmd.Tests[i].CommandArgs = append(execCommand, cmd.Tests[i].CommandArgs...)
	}

	// now copy user defined ut's
	execCommands = append(execCommands, cmd.Tests...)
	data["execCommands"] = execCommands
	data["boolFlags"] = boolFlags
	data["intFlags"] = intFlags
	data["uintFlags"] = uintFlags
	data["strFlags"] = strFlags
	data["strSliceFlags"] = strSliceFlags
	data["intSliceFlags"] = intSliceFlags

	data["boolStubs"] = boolStubs
	data["intStubs"] = intStubs
	data["uintStubs"] = uintStubs
	data["strStubs"] = strStubs
	data["strSliceStubs"] = strSliceStubs
	data["intSliceStubs"] = intSliceStubs

	// dump a go file for new command
	if t, err := ioutil.ReadFile(filepath.Join(execFolder, "templates", "pxStruct.tmpl")); err != nil {
		return err
	} else {
		if b, err := executeTemplate(string(t), data); err != nil {
			er(err)
		} else {
			structWriter.Write([]byte(b))
		}
	}

	if t, err := ioutil.ReadFile(filepath.Join(execFolder, "templates", "pxData.tmpl")); err != nil {
		return err
	} else {
		if b, err := executeTemplate(string(t), data); err != nil {
			er(err)
		} else {
			data["dataStubs"] = b
		}
	}

	// dump a go file for new command
	if t, err := ioutil.ReadFile(filepath.Join(execFolder, "templates", "pxCommand.tmpl")); err != nil {
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

	// dump config file for new command
	if t, err := ioutil.ReadFile(filepath.Join(execFolder, "templates", "config.tmpl")); err != nil {
		return err
	} else {
		if b, err := executeTemplate(string(t), data); err != nil {
			er(err)
		} else {
			configWriter.Write([]byte(b))
		}
	}

	if t, err := ioutil.ReadFile(filepath.Join(execFolder, "templates", "pxFlag.tmpl")); err != nil {
		return err
	} else {
		if b, err := executeTemplate(string(t), data); err != nil {
			er(err)
		} else {
			data["flagStubs"] = b
		}
	}

	if t, err := ioutil.ReadFile(filepath.Join(execFolder, "templates", "pxTest.tmpl")); err != nil {
		return err
	} else {
		if b, err := executeTemplate(string(t), data); err != nil {
			er(err)
		} else {
			testWriter.Write([]byte(b))
		}
	}

	// dump a stub for exec func if user does not provide one in YAML
	if len(cmd.Func) == 0 {
		if t, err := ioutil.ReadFile(filepath.Join(execFolder, "templates", "pxFunction.tmpl")); err != nil {
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

// formatInputUp is a helper func
func formatInputUp(x string) string {
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

// formatInputLo is a helper func
func formatInputLo(x string) string {
	if len(x) == 0 {
		return x
	}

	if len(x) == 1 {
		return strings.ToLower(x)
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

	if len(out) == 1 {
		out = strings.ToLower(out)
	} else {
		out = strings.ToLower(string(out[0])) + out[1:]
	}
	return out
}

func validateSliceForAType(values []string, flagType string) error {
	if len(values) == 0 {
		return nil
	}

	switch flagType {
	case FlagInt:
		for _, value := range values {
			if _, err := strconv.ParseInt(value, 10, 64); err != nil {
				return err
			}
		}
	case FlagUint:
		for _, value := range values {
			if _, err := strconv.ParseUint(value, 10, 64); err != nil {
				return err
			}
		}
	default:
		return nil
	}
	return nil
}
