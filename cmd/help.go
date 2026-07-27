package cmd

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/urfave/cli/v2"
)

var defaultFlagStringer = cli.FlagStringer

func localizedHelpCommand() *cli.Command {
	return &cli.Command{
		Name:      "help",
		Usage:     i18n.T("help.command_summary"),
		ArgsUsage: i18n.T("help.command_args"),
		Action: func(c *cli.Context) error {
			if !c.Args().Present() {
				return cli.ShowAppHelp(c)
			}
			name := c.Args().First()
			if c.App.Command(name) == nil {
				return cli.Exit(i18n.NewError("cli.error.unknown_command", map[string]any{"Command": name}, nil), 3)
			}
			arguments := append([]string{c.App.Name}, c.Args().Slice()...)
			arguments = append(arguments, "--help")
			return c.App.RunContext(c.Context, arguments)
		},
	}
}

func localizeUsageHandlers(commands []*cli.Command) {
	for _, command := range commands {
		command.HideHelpCommand = true
		command.OnUsageError = localizedUsageError
		if len(command.Subcommands) > 0 {
			command.CustomHelpTemplate = localizedSubcommandHelpTemplate()
		} else {
			command.CustomHelpTemplate = localizedCommandHelpTemplate()
		}
		localizeUsageHandlers(command.Subcommands)
	}
}

func localizedUsageError(c *cli.Context, err error, isSubcommand bool) error {
	if errors.Is(err, flag.ErrHelp) {
		if isSubcommand {
			template := cli.CommandHelpTemplate
			if c.Command != nil && c.Command.CustomHelpTemplate != "" {
				template = c.Command.CustomHelpTemplate
			}
			cli.HelpPrinter(c.App.Writer, template, c.Command)
			return nil
		}
		return cli.ShowAppHelp(c)
	}
	detail := err.Error()
	if prefix := "flag provided but not defined: "; strings.HasPrefix(detail, prefix) {
		return i18n.NewError("cli.error.flag_unknown", map[string]any{"Flag": strings.TrimPrefix(detail, prefix)}, err)
	}
	if prefix := "flag needs an argument: "; strings.HasPrefix(detail, prefix) {
		return i18n.NewError("cli.error.flag_value_required", map[string]any{"Flag": strings.TrimPrefix(detail, prefix)}, err)
	}
	return i18n.NewError("cli.error.incorrect_usage", map[string]any{"Detail": detail}, err)
}

func configureCLIHelp() {
	cli.HelpFlag = &cli.BoolFlag{
		Name:               "help",
		Aliases:            []string{"h"},
		Usage:              i18n.T("common.help"),
		DisableDefaultText: true,
	}
	cli.VersionFlag = &cli.BoolFlag{
		Name:               "version",
		Aliases:            []string{"v"},
		Usage:              i18n.T("common.version"),
		DisableDefaultText: true,
	}
	cli.FlagStringer = func(flag cli.Flag) string {
		text := defaultFlagStringer(flag)
		if i18n.CurrentLanguage() == i18n.SimplifiedChinese {
			text = strings.ReplaceAll(text, " value", " 值")
			text = strings.ReplaceAll(text, "(default: ", "(默认值：")
		}
		return text
	}
	cli.AppHelpTemplate = localizedAppHelpTemplate()
	cli.CommandHelpTemplate = localizedCommandHelpTemplate()
	cli.SubcommandHelpTemplate = localizedSubcommandHelpTemplate()
}

func localizedAppHelpTemplate() string {
	return fmt.Sprintf(`%s:
   {{template "helpNameTemplate" .}}

%s:
   {{if .UsageText}}{{wrap .UsageText 3}}{{else}}{{.HelpName}} {{if .VisibleFlags}}%s{{end}}{{if .Commands}} %s %s{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}{{if .Args}}%s{{end}}{{end}}{{end}}{{if .Version}}{{if not .HideVersion}}

%s:
   {{.Version}}{{end}}{{end}}{{if .Description}}

%s:
   {{template "descriptionTemplate" .}}{{end}}{{if .VisibleCommands}}

%s:{{template "visibleCommandCategoryTemplate" .}}{{end}}{{if .VisibleFlagCategories}}

%s:{{template "visibleFlagCategoryTemplate" .}}{{else if .VisibleFlags}}

%s:{{template "visibleFlagTemplate" .}}{{end}}
`,
		i18n.T("help.name"), i18n.T("help.usage"), i18n.T("help.global_options_placeholder"),
		i18n.T("help.command_placeholder"), i18n.T("help.command_options_placeholder"),
		i18n.T("help.arguments_placeholder"), i18n.T("help.version"), i18n.T("help.summary"),
		i18n.T("help.commands"), i18n.T("help.global_options"), i18n.T("help.global_options"))
}

func localizedCommandHelpTemplate() string {
	return fmt.Sprintf(`%s:
   {{template "helpNameTemplate" .}}

%s:
   {{if .UsageText}}{{wrap .UsageText 3}}{{else}}{{.HelpName}}{{if .VisibleFlags}} %s{{end}}{{if .ArgsUsage}} {{.ArgsUsage}}{{else}}{{if .Args}} %s{{end}}{{end}}{{end}}{{if .Category}}

%s:
   {{.Category}}{{end}}{{if .Description}}

%s:
   {{template "descriptionTemplate" .}}{{end}}{{if .VisibleFlagCategories}}

%s:{{template "visibleFlagCategoryTemplate" .}}{{else if .VisibleFlags}}

%s:{{template "visibleFlagTemplate" .}}{{end}}
`, i18n.T("help.name"), i18n.T("help.usage"), i18n.T("help.command_options_placeholder"),
		i18n.T("help.arguments_placeholder"), i18n.T("help.category"), i18n.T("help.summary"),
		i18n.T("help.options"), i18n.T("help.options"))
}

func localizedSubcommandHelpTemplate() string {
	return fmt.Sprintf(`%s:
   {{template "helpNameTemplate" .}}

%s:
   {{if .UsageText}}{{wrap .UsageText 3}}{{else}}{{.HelpName}} {{if .VisibleFlags}}%s{{end}} %s %s{{end}}{{if .Description}}

%s:
   {{template "descriptionTemplate" .}}{{end}}{{if .VisibleCommands}}

%s:{{template "visibleCommandCategoryTemplate" .}}{{end}}{{if .VisibleFlagCategories}}

%s:{{template "visibleFlagCategoryTemplate" .}}{{else if .VisibleFlags}}

%s:{{template "visibleFlagTemplate" .}}{{end}}
`, i18n.T("help.name"), i18n.T("help.usage"), i18n.T("help.command_options_placeholder"),
		i18n.T("help.command_placeholder"), i18n.T("help.command_options_placeholder"),
		i18n.T("help.summary"), i18n.T("help.commands"), i18n.T("help.options"), i18n.T("help.options"))
}
