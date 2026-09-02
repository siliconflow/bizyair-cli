package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

func AppRun(c *cli.Context) error {
	args := parseArgument(c, meta.CmdApp)
	setLogVerbose(args.Verbose)
	logArguments(args)

	arg := c.Args().First()
	if arg == "" {
		return cli.Exit(i18n.NewError("cli.app.id_required", nil, nil), meta.LoadError)
	}

	_, client, err := ResolveClient(args)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	ref, err := resolveWebAppVersion(c, client, args, arg)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	detailResult := actions.GetAIAppDetail(c.Context, client, ref.VersionID)
	if detailResult.Error != nil {
		return cli.Exit(detailResult.Error, meta.ServerError)
	}
	detail := detailResult.Detail
	if detail == nil {
		return cli.Exit(i18n.NewError("cli.app.not_found", map[string]any{"ID": arg}, nil), meta.ServerError)
	}

	params, err := buildAppRunParams(c, detail)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}
	lib.ApplyWebAppFieldDefaults(params, detail)

	taskResult := actions.CreateWebAppTask(c.Context, client, lib.WebAppTaskCreateReq{
		WebAppId:    detail.Id,
		InputValues: params,
	})
	if taskResult.Error != nil {
		return cli.Exit(taskResult.Error, meta.ServerError)
	}

	start := time.Now()
	fmt.Fprintf(os.Stdout, "%s %s\n", gray(i18n.T("cli.app.run.app_label", nil)), cyan(detail.Name))
	if taskResult.TaskID > 0 {
		fmt.Fprintf(os.Stdout, "%s %d\n", gray(i18n.T("cli.app.run.request_id_label", nil)), taskResult.TaskID)
	}
	if stdoutIsTTY() {
		fmt.Fprintln(os.Stdout)
	}

	spinner := []string{"|", "/", "-", "\\"}
	si := 0
	for {
		time.Sleep(500 * time.Millisecond)
		statusResult := actions.GetWebAppTaskStatus(c.Context, client, taskResult.TaskID)
		if statusResult.Error != nil {
			clearLine()
			return cli.Exit(statusResult.Error, meta.ServerError)
		}
		if statusResult.Status == nil {
			continue
		}
		status := statusResult.Status.Status
		switch status {
		case lib.TaskStatusSuccess:
			clearLine()
			printAppRunSuccess()
			if requestID := statusResult.Status.RequestID; requestID != "" {
				outputsResult := actions.GetWebAppTaskOutputs(c.Context, client, requestID)
				if outputsResult.Error == nil {
					for _, o := range outputsResult.Outputs {
						if o.ObjectURL != "" {
							fmt.Fprintln(os.Stdout, o.ObjectURL)
						}
					}
				}
			}
			return nil
		case lib.TaskStatusFailed, lib.TaskStatusCancelled:
			clearLine()
			printRunFailure(status, time.Since(start))
			return nil
		default:
			printRunPolling(spinner[si%len(spinner)], lib.WebAppTaskStatusName(status), time.Since(start))
			si++
		}
	}
}

// printAppRunSuccess 输出任务成功的头两行。

func printAppRunSuccess() {
	fmt.Fprintf(os.Stdout, "%s %s\n", green(i18n.T("cli.icon.success", nil)), green(bold(lib.WebAppTaskStatusName(lib.TaskStatusSuccess))))
}

// buildAppRunParams 从 CLI 参数收集 AI 应用运行输入：
// --param 传普通参数，--image 传图片/视频 URL 参数（URL 需以 http(s) 开头）。

func buildAppRunParams(c *cli.Context, detail *lib.WebAppDetail) (map[string]any, error) {
	params := make(map[string]any)

	nodeMediaMap := make(map[string]bool)
	nodeTypeMap := make(map[string]string)
	for _, n := range detail.InputNodes {
		key := lib.WebAppNodeParamKey(n)
		if lib.IsWebAppMediaNode(n) {
			nodeMediaMap[key] = true
		}
		nodeTypeMap[key] = lib.WebAppNodeVariableType(n)
	}

	for _, kv := range c.StringSlice("param") {
		idx := strings.Index(kv, "=")
		if idx < 1 {
			return nil, fmt.Errorf("%s", i18n.T("cli.app.run.error.param_format", map[string]any{"Param": kv}))
		}
		key := kv[:idx]
		val := kv[idx+1:]
		if nodeMediaMap[key] && !lib.IsHTTPURL(val) {
			return nil, fmt.Errorf("%s", i18n.T("cli.app.run.error.url_invalid", map[string]any{"Key": key}))
		}
		vtype := nodeTypeMap[key]
		if vtype == "" {
			vtype = "string"
		}
		coerced, err := lib.CoerceModelzooParamValue(vtype, val)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		params[key] = coerced
	}

	for _, kv := range c.StringSlice("image") {
		idx := strings.Index(kv, "=")
		if idx < 1 {
			return nil, fmt.Errorf("%s", i18n.T("cli.app.run.error.param_format", map[string]any{"Param": kv}))
		}
		key := kv[:idx]
		urlVal := kv[idx+1:]
		if !lib.IsHTTPURL(urlVal) {
			return nil, fmt.Errorf("%s", i18n.T("cli.app.run.error.url_invalid", map[string]any{"Key": key}))
		}
		params[key] = urlVal
	}

	return params, nil
}
