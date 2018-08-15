# Buffalo Plugin Template

Tiny one-file program which shows interaction between [Buffalo](https://gobuffalo.io) and its [plugin](https://gobuffalo.io/en/docs/plugins). It generates/destroys one command plugins stored in [GOPATH](https://github.com/golang/go/wiki/GOPATH).

### Prerequisites

It requires:

* [Golang](https://golang.org/doc/install)
* [Buffalo](https://github.com/gobuffalo/buffalo)

## Installation

```
$ cd buffalo-plugin-template source directory
$ go build
$ go install

# Let's check if Buffalo recognizes this plugin:
$ buffalo generate --help
Available Commands:
...
plugin-template     [PLUGIN] [--output (gohome|stdout)] <buffalo_command> <plugin_name> Generates a new plugin.
```

## Usage

```
# This command will create new plugin at: 
# ${GOPATH}/src/my/plugin/subdir/buffalo-my-plugin
$ buffalo generate plugin-template generate my/plugin/subdir/buffalo-my-plugin

$ cd ${GOPATH}/src/my/plugin/subdir/buffalo-my-plugin
$ go build
$ go install

# Now newly generated plugin is accessible from Buffalo.
# Let's see available plugins for "buffalo generate" command:
$ buffalo generate --help
Available Commands:
...
buffalo-my-plugin   [PLUGIN] Here is command description

# Let's edit new plugin source code to customize it to your requirements.
$ vi ${GOPATH}/src/my/plugin/subdir/buffalo-my-plugin/buffalo-my-plugin.go
```

