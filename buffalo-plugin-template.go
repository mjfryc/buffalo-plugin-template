// Buffalo plugin which can generate other buffalo plugins.
// See --help for details.
// @date   2018-08-05
// @author Michal Jaron <mjfryc@gmail.com>

package main

import (
	"encoding/json"
	"fmt"
	"go/build"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"github.com/gobuffalo/buffalo/plugins"
)

func getGoHome() string {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	return gopath
}

// buffaloCommandsToJSONString() converts plugins.Commands to JSON string
// which will be returned by "available" command, so buffalo is able to determine
// options provided by this plugin.
// On error causes Fatal error.
func buffaloCommandsToJSONString(buffaloCommands plugins.Commands) string {
	commandsJSON, err := json.Marshal(buffaloCommands)
	if err != nil {
		log.Fatal("Cannot encode to JSON ", err)
	}

	return string(commandsJSON)
}

// ArgumentIterator describes number of agument from command line.
// Allows to iterate over command line arguments.
type ArgumentIterator int

func (argumentIterator ArgumentIterator) getArgument(argumentDescription string) string {
	if len(os.Args) <= int(argumentIterator) {
		log.Fatal("Failed to read argument: \""+argumentDescription,
			"\" expected in command line arguments at position: "+argumentIterator.getIndexString(),
			"\n all arguments (counting them from 0):",
			"\n", os.Args,
			"\nSee help:\n"+helpToString())
	}

	return os.Args[int(argumentIterator)]
}
func (argumentIterator ArgumentIterator) hasNextArgument() bool {
	return len(os.Args) > int(argumentIterator)
}
func (argumentIterator ArgumentIterator) getIndexString() string {
	var itString = strconv.Itoa(int(argumentIterator))
	return itString
}
func (argumentIterator *ArgumentIterator) getNextArgument(argumentDescription string) string {
	var argumentValue = argumentIterator.getArgument(argumentDescription)
	*argumentIterator++
	return argumentValue
}
func (argumentIterator ArgumentIterator) seeNextArgument(argumentDescription string) string {
	var argumentValue = argumentIterator.getArgument(argumentDescription)
	return argumentValue
}

func helpToString() string {
	var helpString = `    Commands:
      available Prints commands for buffalo in json format. See: https://gobuffalo.io/en/docs/plugins#writing-a-plugin
      (buffalo generate plugin-template | buffalo-plugin-template plugin-template-generate)
		[ --output (gohome|stdout) ] <buffalo_command> <plugin_name> Generates a new plugin.
	  parameters:
        <buffalo_command> is one of commands available in buffalo. Type buffalo --help to list buffalo commands.
        <plugin_name>     tells what's [subdiredtory/]name of your plugin. It has to start with "buffalo-"
                          e.g. myNamespace/MyPluginName will generate plugin at: $GOPATH/src/myNamespace/MyPluginName
      (buffalo destroy plugin-template | buffalo-plugin-template)
          plugin-template-destroy <plugin_name>
     
      Optional flags:
      --help    Prints this help.
      --output  (gohome|stdout) [default: gohome]`

	return helpString
}

func printHelp() {
	fmt.Println(helpToString())
}

var generationOutput = "gohome"

func setOutput(outputValue string) {
	log.Println("Setting output to: " + outputValue)
	if "gohome" == outputValue || "stdout" == outputValue {
		generationOutput = outputValue
	} else {
		log.Fatal("Invalid value of argument \"output\": \"" + outputValue + "\"")
	}
}

func determineAbsolutePluginPath(pluginNameWithPath string) string {
	var sep = string(os.PathSeparator)
	var generationFileDir, err = filepath.Abs(getGoHome() + sep + "src" + sep + pluginNameWithPath)
	if err != nil {
		panic(err)
	}
	return generationFileDir
}

func destroyPluginTemplate(pluginNameWithPath string) {
	var absolutePluginDir = determineAbsolutePluginPath(pluginNameWithPath)
	log.Println("Destroying buffalo plugin at directory: [" + absolutePluginDir + "]")

	stat, err := os.Stat(absolutePluginDir)
	if err != nil {
		log.Fatal(err)
	}
	if stat.IsDir() {
		err := os.RemoveAll(absolutePluginDir)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Fatal("Path is not a diretory or doesn't exist.")
	}
}

func determineHandlerName(pluginName string) string {
	if !strings.HasPrefix(pluginName, "buffalo-") {
		log.Fatal("Buffalo plugin name is incorrect, it should start with \"buffalo-\": [" + pluginName + "]")
	}
	var pluginNameWithoutBuffaloPrefix = pluginName[len("buffalo-"):len(pluginName)]
	var pluginNameCamelCase = string("")
	var nextCamelCase = true
	for _, characterEntry := range pluginNameWithoutBuffaloPrefix {
		if unicode.IsLetter(characterEntry) || unicode.IsNumber(characterEntry) {
			if nextCamelCase {
				pluginNameCamelCase += string(unicode.ToUpper(characterEntry))
				nextCamelCase = false
			} else {
				pluginNameCamelCase += string(characterEntry)
			}
		} else {
			nextCamelCase = true
		}
	}

	if len(pluginNameCamelCase) == 0 {
		log.Fatal("Failed to convert plugin name to camel case golang identifier: [" + pluginName + "]")
	}

	return "handle" + pluginNameCamelCase
}

func generatePluginTemplate(buffaloCommand string, pluginNameWithPath string) {
	var pluginName = filepath.Base(pluginNameWithPath)

	var templateParams = TemplateParams{
		BuffaloCommand:            buffaloCommand,
		PluginName:                pluginName,
		BuffaloCommandHandlerName: determineHandlerName(pluginName),
	}

	tmpl, err := template.New("PluginTemplate").Parse(templateBody)
	if err != nil {
		panic(err)
	}

	var outputInterface io.Writer
	if "stdout" == generationOutput {
		outputInterface = os.Stdout
		log.Println("Generating buffalo plugin at: [" + generationOutput + "]")
	} else if "gohome" == generationOutput {
		var sep = string(os.PathSeparator)
		var absolutePluginDir = determineAbsolutePluginPath(pluginNameWithPath)
		var generationFilePath = absolutePluginDir + sep + pluginName + ".go"
		os.MkdirAll(absolutePluginDir, os.ModePerm)
		outputInterface, err = os.Create(generationFilePath)
		if err != nil {
			panic(err)
		}
		log.Println("Generating buffalo plugin at: [" + absolutePluginDir + "]")
	} else {
		panic("Unexpected generation output name: [" + generationOutput + "]")
	}

	err = tmpl.Execute(outputInterface, templateParams)
	if err != nil {
		panic(err)
	}
}

func processGeneratePluginTemplate(argumentIterator *ArgumentIterator) {
	for argumentIterator.hasNextArgument() {
		var flagOrTemplateGeneratorCommand = argumentIterator.seeNextArgument("Flag or buffalo generate plugin-template's parameter")
		switch flagOrTemplateGeneratorCommand {
		case "--output":
			argumentIterator.getNextArgument("--output argument consume")
			var outputValue = argumentIterator.getNextArgument("Output value: one of (gohome|stdout)")
			setOutput(outputValue)
		default:
			goto generation
		}
	}
generation:
	generatePluginTemplate(argumentIterator.getNextArgument("<buffalo_command> For which buffalo command is this plugin for."), argumentIterator.getNextArgument("<plugin_name> Generated plugin name."))
}

func processDestroyPluginTemplate(argumentIterator *ArgumentIterator) {
	destroyPluginTemplate(argumentIterator.getNextArgument("<plugin_name> Generated plugin name."))
}

func main() {
	if len(os.Args) <= 1 {
		printHelp()
		return
	}

	var buffaloCommands = plugins.Commands{
		plugins.Command{
			Name:           "plugin-template",
			UseCommand:     "plugin-template-generate",
			BuffaloCommand: "generate",
			Description:    "[--output (gohome|stdout)] <buffalo_command> <plugin_name> Generates a new plugin.",
		},
		plugins.Command{
			Name:           "plugin-template",
			UseCommand:     "plugin-template-destroy",
			BuffaloCommand: "destroy",
			Description:    "<plugin_name> Destroys plugin previously generated by \"buffalo generate plugin-template\" command",
		},
	}

	//<buffalo_command> is one of commands available in buffalo. Type buffalo --help to list buffalo commands.
	//<plugin_name>     tells what's [subdiredtory/]name of your plugin. It has to start with "buffalo-"
	//e.g. myNamespace/MyPluginName will generate plugin at: $GOPATH/src/myNamespace/MyPluginName`,

	// Starting from 2nd argument (with index 1)
	for argumentIterator := ArgumentIterator(1); argumentIterator.hasNextArgument(); {
		var flagOrBuffaloCommand = argumentIterator.getNextArgument("Flag or buffalo root command")
		switch flagOrBuffaloCommand {
		case "available":
			fmt.Println(buffaloCommandsToJSONString(buffaloCommands))
		case "--help":
			printHelp()
			return
		case "plugin-template-generate":
			processGeneratePluginTemplate(&argumentIterator)
		case "plugin-template-destroy":
			processDestroyPluginTemplate(&argumentIterator)
		default:
			fmt.Println("Unrecognized option: " + flagOrBuffaloCommand)
			fmt.Println("All passed arguments:")
			fmt.Println(os.Args)
			printHelp()
			return
		}
	}
}

// TemplateParams are used to fill templateBody.
type TemplateParams struct {
	BuffaloCommand            string
	PluginName                string
	BuffaloCommandHandlerName string
}

var templateBody = `
package main

import (
    "encoding/json"
    "fmt"
    "log"
	"os"
	
	"github.com/gobuffalo/buffalo/plugins"
)

// buffaloCommandsToJSONString() converts plugins.Commands to JSON string
// which will be returned by "available" command, so buffalo is able to determine
// options provided by this plugin.
// On error causes Fatal error.
func buffaloCommandsToJSONString(buffaloCommands plugins.Commands) string {
	commandsJSON, err := json.Marshal(buffaloCommands)
	if err != nil {
		log.Fatal("Cannot encode to JSON ", err)
	}

	return string(commandsJSON)
}

func printHelp() {
    fmt.Println("buffalo-plugin-template options:")
    fmt.Println("  --help    Prints this help.")
    fmt.Println("  available Prints commands for buffalo in json format. See: https://gobuffalo.io/en/docs/plugins#writing-a-plugin")
}

// handleBuffalo{{.BuffaloCommand}}{{.PluginName}}() describes how to handle command:
// buffalo {{.BuffaloCommand}} {{.PluginName}}
func {{.BuffaloCommandHandlerName}}() {
    fmt.Println("Handling: buffalo {{.BuffaloCommand}} {{.PluginName}}. TODO: Write your implementation here.")
}

func main() {
    if len(os.Args) < 2 {
        printHelp()
        return
    }

	var buffaloCommands = plugins.Commands {
		// Add your commands here
		// About commands:
		// Let's imagine following plugins.Command in plugin named "buffalo-my-plugin":
		//     plugins.Command { Name: "my-command-name", UseCommand: "my-command-use", BuffaloCommand: "generate", }
		// 
		// When calling such buffalo plugin, e.g:
		//     buffalo generate my-command-name <my-arguments-here-if-needed>
		// 
		// Buffalo will call:
		//     buffalo-my-plugin my-command-use <my-arguments-here-if-needed>
		//
		// UseCommand is useful for cases where there are different BuffaloCommands with the same name, e.g:
		//     buffalo generate my-command-name
		//     buffalo destroy my-command-name
		// Above commands should have different UseCommands.
		plugins.Command {
			Name:           "{{.PluginName}}",
			UseCommand:     "{{.PluginName}}", // buffalo will change Name to UseCommand while calling this plugin.
			BuffaloCommand: "{{.BuffaloCommand}}",
			Description:    "Here is command description",
		},
	}

    switch os.Args[1] {
    case "available":
        fmt.Println(buffaloCommandsToJSONString(buffaloCommands))
    case "--help":
		printHelp()

    // Process your commands here:
    case "{{.PluginName}}": // Value passed in UseCommand.
	    {{.BuffaloCommandHandlerName}}()
	default:
		log.Fatal("Unexpected argument: [", os.Args[1], "], all arguments: ", os.Args)
        printHelp()
    }
}
`
