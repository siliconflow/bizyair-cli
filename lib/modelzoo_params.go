package lib

import (
	"strconv"
	"strings"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
)

// ApplyModelzooFieldDefaults 为未填写的参数补入 API 默认值，CLI 与 TUI 共用。
func ApplyModelzooFieldDefaults(params map[string]any, inputParams []ModelzooInputParam) {
	for _, p := range inputParams {
		key := p.ParamKey()
		if _, ok := params[key]; !ok && p.FieldValue != nil {
			params[key] = p.FieldValue
		}
	}
}

// MissingModelzooRequiredParams 返回仍未满足的必填参数：
// 已填有值或 API 提供了默认值的参数视为已满足。
func MissingModelzooRequiredParams(inputParams []ModelzooInputParam, params map[string]any) []ModelzooInputParam {
	missing := make([]ModelzooInputParam, 0)
	for _, p := range inputParams {
		if !p.Required {
			continue
		}
		if p.FieldValue != nil {
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

// CoerceModelzooParamValue 按参数类型解析并转换用户输入值。
// number/float → float64，integer → int64，boolean → bool，其余原样返回字符串。
// 解析失败时返回原始字符串与错误，供调用方决定行为。
func CoerceModelzooParamValue(vtype, raw string) (any, error) {
	switch vtype {
	case "number", "float":
		f, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err != nil {
			return raw, err
		}
		return f, nil
	case "integer":
		i, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
		if err != nil {
			return raw, err
		}
		return i, nil
	case "boolean":
		b, err := strconv.ParseBool(strings.TrimSpace(raw))
		if err != nil {
			return raw, err
		}
		return b, nil
	default:
		return raw, nil
	}
}

// ModelzooStatusName 返回任务状态的本地化显示名。
func ModelzooStatusName(status string) string {
	switch status {
	case TaskStatusSuccess:
		return i18n.T("common.modelzoo.task_status_success", nil)
	case TaskStatusFailed:
		return i18n.T("common.modelzoo.task_status_failed", nil)
	case TaskStatusRunning:
		return i18n.T("common.modelzoo.task_status_running", nil)
	case TaskStatusQueued:
		return i18n.T("common.modelzoo.task_status_queued", nil)
	case TaskStatusCancelled:
		return i18n.T("common.modelzoo.task_status_cancelled", nil)
	case TaskStatusTransferring:
		return i18n.T("common.modelzoo.task_status_transferring", nil)
	default:
		return status
	}
}