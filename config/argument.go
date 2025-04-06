package config

import (
	"github.com/urfave/cli/v2"
)

type Argument struct {
	CmdType    string // command type
	Verbose    bool   // print verbose log
	BaseDomain string // request domain
	ApiKey     string // api key
	Path       string // local path to upload
	Type       string // type of the file to upload
	Name       string // name of the model
	ExtName    string // extension name of the model
	ShowFiles  bool   // show files
	FilePath   string // file path
	FormatTree bool   // format tree
	Overwrite  bool   // overwrite model
	Public     bool   // only show public model
	// Host			string
	// Port			string
	ModelVersion 	string
	Versions    string // model versions info
	BaseModel   string
	CoverUrls   string
	Intro       string
	Current		int
	PageSize	int
}

func NewArgument() *Argument {
	return &Argument{}
}

// Parse initializes a new argument based on its own information
func (arg *Argument) Parse(c *cli.Context, cmd string) (*Argument, error) {
	// v2 cli cannot put the StringSlice flag to struct, so we need to parse it here
	arg.parseStringSlice(c)
	args := arg.Fork()
	args.CmdType = cmd

	return args, nil
}

func (arg *Argument) parseStringSlice(c *cli.Context) {
	// parse string slice
}

// Fork can copy its own parameters to a new argument
func (arg *Argument) Fork() *Argument {
	args := NewArgument()
	*args = *arg
	return args
}
