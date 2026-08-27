package cmd

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
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

	fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.modelzoo.run.endpoint_label", nil), detail.Endpoint)
	fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.modelzoo.run.name_label", nil), detail.DisplayName)
	if detail.Description != "" {
		fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.modelzoo.run.desc_label", nil), detail.Description)
	}

	params, err := buildRunParams(c, detail.InputParams)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	if len(params) == 0 && hasRequiredParams(detail.InputParams) {
		fmt.Fprintln(os.Stdout, i18n.T("cli.modelzoo.run.params_hint", nil))
		for _, p := range detail.InputParams {
			req := ""
			if p.Required {
				req = i18n.T("cli.modelzoo.param_required", nil)
			}
			fmt.Fprintln(os.Stdout, i18n.T("cli.modelzoo.run.param_hint_line", map[string]any{
				"Name":     p.ParamKey(),
				"Label":    p.FieldLabel,
				"Type":     p.VariableType,
				"Required": req,
			}))
		}
		return nil
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
				for _, u := range extractOutputURLs(statusResult.Status.Outputs) {
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

func hasRequiredParams(inputParams []lib.ModelzooInputParam) bool {
	for _, p := range inputParams {
		if p.Required {
			return true
		}
	}
	return false
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

func extractOutputURLs(outputs any) []string {
	var urls []string
	extractURLsRecursive(reflect.ValueOf(outputs), &urls)
	return urls
}

func extractURLsRecursive(v reflect.Value, urls *[]string) {
	if !v.IsValid() {
		return
	}

	switch v.Kind() {
	case reflect.String:
		s := v.String()
		if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
			*urls = append(*urls, s)
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			extractURLsRecursive(v.MapIndex(key), urls)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			extractURLsRecursive(v.Index(i), urls)
		}
	case reflect.Interface:
		extractURLsRecursive(v.Elem(), urls)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			extractURLsRecursive(v.Field(i), urls)
		}
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
