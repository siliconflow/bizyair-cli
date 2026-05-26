package cmd

import (
	"github.com/siliconflow/bizyair-cli/config"
	"github.com/urfave/cli/v2"
)

func parseArgument(c *cli.Context, cmd string) *config.Argument {
	args := globalArgs.Fork()
	args.CmdType = cmd

	args.ModelVersion = c.StringSlice("version")
	args.Intro = c.StringSlice("intro")
	args.IntroPath = c.StringSlice("intro-path")
	args.Path = c.StringSlice("path")
	args.CoverUrls = c.StringSlice("cover")
	args.BaseModel = c.StringSlice("base")
	args.VersionPublic = c.StringSlice("public")

	return args
}
