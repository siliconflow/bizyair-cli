package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
	"github.com/siliconflow/bizyair-cli/lib/format"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

func ModelzooRun(c *cli.Context) error {
	args := parseArgument(c, meta.CmdRun)
	setLogVerbose(args.Verbose)
	logArguments(args)

	endpoint := c.Args().First()
	if endpoint == "" {
		return cli.Exit(i18n.NewError("cli.modelzoo.endpoint_required", nil, nil), meta.LoadError)
	}

	_, client, err := ResolveClient(args)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	detailResult := actions.GetEndpointDetail(c.Context, client, endpoint)
	if detailResult.Error != nil {
		return cli.Exit(detailResult.Error, meta.ServerError)
	}

	detail := detailResult.Detail
	if detail == nil {
		return cli.Exit(i18n.NewError("cli.modelzoo.detail_not_found", map[string]any{"Endpoint": endpoint}, nil), meta.ServerError)
	}

	params, err := buildRunParams(c, detail.InputParams)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	applyFieldDefaults(params, detail.InputParams)
	if missing := missingRequiredParams(detail.InputParams, params); len(missing) > 0 {
		return cli.Exit(missingParamsError(missing), meta.LoadError)
	}

	taskResult := actions.CreateTask(c.Context, client, endpoint, params)
	if taskResult.Error != nil {
		return cli.Exit(taskResult.Error, meta.ServerError)
	}

	requestID := taskResult.RequestID
	fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.modelzoo.run.request_id_label", nil), requestID)
	fmt.Fprintln(os.Stdout, i18n.T("cli.modelzoo.run.polling_status", nil))

	for {
		time.Sleep(2 * time.Second)
		statusResult := actions.GetTaskStatus(c.Context, client, requestID)
		if statusResult.Error != nil {
			return cli.Exit(statusResult.Error, meta.ServerError)
		}

		status := statusResult.Status.Status
		switch status {
		case lib.TaskStatusSuccess:
			fmt.Fprintf(os.Stdout, "\n%s: %s\n", i18n.T("cli.modelzoo.run.status_label", nil), statusDisplayName(status))
			if statusResult.Status.Outputs != nil {
				for _, u := range format.ExtractOutputURLs(statusResult.Status.Outputs) {
					fmt.Fprintln(os.Stdout, u)
				}
			}
			return nil
		case lib.TaskStatusFailed, lib.TaskStatusCancelled:
			fmt.Fprintf(os.Stdout, "\n%s: %s\n", i18n.T("cli.modelzoo.run.status_label", nil), statusDisplayName(status))
			return nil
		default:
			fmt.Fprintf(os.Stdout, "%s ", statusDisplayName(status))
		}
	}
}

func buildRunParams(c *cli.Context, inputParams []lib.ModelzooInputParam) (map[string]any, error) {
	params := make(map[string]any)

	paramTypeMap := make(map[string]string)
	for _, p := range inputParams {
		paramTypeMap[p.ParamKey()] = p.VariableType
	}

	for _, kv := range c.StringSlice("param") {
		idx := strings.Index(kv, "=")
		if idx < 1 {
			return nil, fmt.Errorf("%s", i18n.T("cli.modelzoo.run.error.param_format", map[string]any{"Param": kv}))
		}
		key := kv[:idx]
		val := kv[idx+1:]

		vtype := paramTypeMap[key]
		switch vtype {
		case "number", "float", "integer":
			f, err := strconv.ParseFloat(val, 64)
			if err == nil {
				params[key] = f
			} else {
				params[key] = val
			}
		case "boolean":
			b, err := strconv.ParseBool(val)
			if err != nil {
				return nil, fmt.Errorf("%s", i18n.T("cli.modelzoo.run.error.boolean_invalid", map[string]any{"Key": key, "Value": val}))
			}
			params[key] = b
		default:
			params[key] = val
		}
	}

	for _, kv := range c.StringSlice("image") {
		idx := strings.Index(kv, "=")
		if idx < 1 {
			return nil, fmt.Errorf("%s", i18n.T("cli.modelzoo.run.error.param_format", map[string]any{"Param": kv}))
		}
		key := kv[:idx]
		urlVal := kv[idx+1:]

		if !strings.HasPrefix(urlVal, "http://") && !strings.HasPrefix(urlVal, "https://") {
			return nil, fmt.Errorf("%s", i18n.T("cli.modelzoo.run.error.url_invalid", map[string]any{"Key": key}))
		}
		params[key] = urlVal
	}

	return params, nil
}

// applyFieldDefaults 为未填写的参数补入 API 默认值（对齐 TUI 的 applyFieldDefaults）。
func applyFieldDefaults(params map[string]any, inputParams []lib.ModelzooInputParam) {
	for _, p := range inputParams {
		key := p.ParamKey()
		if _, ok := params[key]; !ok && p.FieldValue != nil {
			params[key] = p.FieldValue
		}
	}
}

// missingRequiredParams 返回仍未填写的必填参数。
func missingRequiredParams(inputParams []lib.ModelzooInputParam, params map[string]any) []lib.ModelzooInputParam {
	missing := make([]lib.ModelzooInputParam, 0)
	for _, p := range inputParams {
		if !p.Required {
			continue
		}
		key := p.ParamKey()
		val, ok := params[key]
		if ok && val != nil && val != "" {
			continue
		}
		missing = append(missing, p)
	}
	return missing
}

// missingParamsError 构建缺失必填参数的错误信息（含参数明细与用法提示）。
func missingParamsError(missing []lib.ModelzooInputParam) error {
	names := make([]string, 0, len(missing))
	for _, p := range missing {
		names = append(names, p.ParamKey())
	}
	var b strings.Builder
	b.WriteString(i18n.T("cli.modelzoo.run.error.missing_params", map[string]any{"Names": strings.Join(names, ", ")}))
	b.WriteString("\n")
	for _, p := range missing {
		def := ""
		if p.FieldValue != nil {
			def = i18n.T("cli.modelzoo.run.error.default_value", map[string]any{"Value": fmt.Sprintf("%v", p.FieldValue)})
		}
		b.WriteString(i18n.T("cli.modelzoo.run.error.missing_param_line", map[string]any{
			"Name":    p.ParamKey(),
			"Label":   p.FieldLabel,
			"Type":    variableTypeDisplayName(p.VariableType),
			"Default": def,
		}))
		b.WriteString("\n")
	}
	b.WriteString(i18n.T("cli.modelzoo.run.error.usage_hint", nil))
	return fmt.Errorf("%s", b.String())
}

func statusDisplayName(status string) string {
	switch status {
	case lib.TaskStatusSuccess:
		return i18n.T("cli.modelzoo.run.status_success", nil)
	case lib.TaskStatusFailed:
		return i18n.T("cli.modelzoo.run.status_failed", nil)
	case lib.TaskStatusRunning:
		return i18n.T("cli.modelzoo.run.status_running", nil)
	case lib.TaskStatusQueued:
		return i18n.T("cli.modelzoo.run.status_queued", nil)
	case lib.TaskStatusCancelled:
		return i18n.T("cli.modelzoo.run.status_cancelled", nil)
	case lib.TaskStatusTransferring:
		return i18n.T("cli.modelzoo.run.status_transferring", nil)
	default:
		return status
	}
}

func variableTypeDisplayName(vt string) string {
	switch vt {
	case "number", "float", "integer":
		return i18n.T("cli.modelzoo.run.type_number", nil)
	case "boolean":
		return i18n.T("cli.modelzoo.run.type_boolean", nil)
	case "image", "video":
		return i18n.T("cli.modelzoo.run.type_media", nil)
	case "string", "text":
		return i18n.T("cli.modelzoo.run.type_string", nil)
	default:
		return vt
	}
}
