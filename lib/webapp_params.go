package lib

import (
	"encoding/json"
	"strings"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
)

// WebAppInputParams 将 AI 应用详情的输入节点转换为 modelzoo 参数形态，
// 从而复用任务向导与参数校验/强制类型转换逻辑。
func WebAppInputParams(detail *WebAppDetail) []ModelzooInputParam {
	if detail == nil {
		return nil
	}
	params := make([]ModelzooInputParam, 0, len(detail.InputNodes))
	for _, n := range detail.InputNodes {
		params = append(params, WebAppNodeToParam(n))
	}
	return params
}

// WebAppNodeToParam 将单个输入节点映射为 modelzoo 参数。
func WebAppNodeToParam(n WebAppInputNode) ModelzooInputParam {
	return ModelzooInputParam{
		VariableName: n.VariableName,
		VariableType: WebAppNodeVariableType(n),
		FieldName:    n.FieldName,
		FieldType:    n.FieldType,
		FieldValue:   n.FieldValue,
		FieldLabel:   n.FieldLabel,
		FieldTooltip: WebAppFieldTooltip(n),
		FieldOptions: ParseWebAppFieldOptions(n),
	}
}

// WebAppFieldTooltip 返回节点的说明文案（优先 node_name，其次 field_label）。
func WebAppFieldTooltip(n WebAppInputNode) string {
	if n.NodeName != "" {
		return n.NodeName
	}
	return ""
}

// WebAppNodeVariableType 推断输入节点的变量类型：
// 图片/视频节点按 node_type 识别，其余按 field_type 归一化到 modelzoo 类型。
func WebAppNodeVariableType(n WebAppInputNode) string {
	if media := WebAppNodeMediaType(n); media != "" {
		return media
	}
	switch strings.ToLower(n.FieldType) {
	case "float", "number", "num":
		return "number"
	case "int", "integer":
		return "integer"
	case "bool", "boolean":
		return "boolean"
	default:
		return "string"
	}
}

// WebAppNodeMediaType 返回输入节点的媒体类型（image/video），非媒体节点返回空串。
func WebAppNodeMediaType(n WebAppInputNode) string {
	t := strings.ToLower(n.NodeType)
	switch {
	case strings.Contains(t, "loadvideo"):
		return "video"
	case strings.Contains(t, "loadimage"), strings.Contains(t, "load_image"):
		return "image"
	default:
		return ""
	}
}

// IsWebAppMediaNode 判断输入节点是否为图片/视频媒体输入。
func IsWebAppMediaNode(n WebAppInputNode) bool {
	return WebAppNodeMediaType(n) != ""
}

// ParseWebAppFieldOptions 尽力解析输入节点的 FieldOptions（JSON 字符串/对象/数组）。
// 成功时返回枚举选项；无法解析或元数据对象（min/max 等）返回 nil，调用方按普通输入处理。
func ParseWebAppFieldOptions(n WebAppInputNode) *ModelzooFieldOptions {
	raw := strings.TrimSpace(n.FieldOptions)
	if raw == "" {
		return nil
	}
	fo := &ModelzooFieldOptions{}
	if strings.HasPrefix(raw, "[") {
		var values []any
		if err := json.Unmarshal([]byte(raw), &values); err == nil {
			fo.Values = values
			return fo
		}
		return nil
	}
	if !strings.HasPrefix(raw, "{") {
		return nil
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return nil
	}
	enums := webAppExtractEnumLists(obj)
	if len(enums.Values) > 0 {
		fo.Values = enums.Values
	}
	if len(enums.Options) > 0 {
		fo.Options = enums.Options
	}
	if len(enums.Choices) > 0 {
		fo.Choices = enums.Choices
	}
	if len(fo.Values) == 0 && len(fo.Options) == 0 && len(fo.Choices) == 0 {
		return nil
	}
	return fo
}

func webAppExtractEnumLists(obj map[string]any) (out struct {
	Values  []any
	Options []any
	Choices []any
}) {
	if v, ok := obj["values"]; ok {
		out.Values = toAnyList(v)
	}
	if v, ok := obj["options"]; ok {
		out.Options = toAnyList(v)
	}
	if v, ok := obj["choices"]; ok {
		out.Choices = toAnyList(v)
	}
	return out
}

func toAnyList(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	case []string:
		out := make([]any, 0, len(t))
		for _, s := range t {
			out = append(out, s)
		}
		return out
	default:
		return nil
	}
}

// MissingWebAppRequiredParams 返回仍有缺失的必填输入节点（复用 modelzoo 判定逻辑）。
func MissingWebAppRequiredParams(detail *WebAppDetail, params map[string]any) []WebAppInputNode {
	if detail == nil {
		return nil
	}
	var missing []WebAppInputNode
	for _, n := range detail.InputNodes {
		key := n.FieldName
		if key == "" {
			key = n.VariableName
		}
		if key == "" {
			continue
		}
		if n.FieldValue != nil {
			continue
		}
		val, ok := params[key]
		if ok && val != nil && val != "" {
			continue
		}
		missing = append(missing, n)
	}
	return missing
}

// ApplyWebAppFieldDefaults 为未填写的输入节点补入 API 默认值。
func ApplyWebAppFieldDefaults(params map[string]any, detail *WebAppDetail) {
	if detail == nil {
		return
	}
	for _, n := range detail.InputNodes {
		key := n.FieldName
		if key == "" {
			key = n.VariableName
		}
		if key == "" || n.FieldValue == nil {
			continue
		}
		if _, ok := params[key]; !ok {
			params[key] = n.FieldValue
		}
	}
}

// WebAppNodeParamKey 返回输入节点的参数键名。
func WebAppNodeParamKey(n WebAppInputNode) string {
	if n.FieldName != "" {
		return n.FieldName
	}
	return n.VariableName
}

// WebAppFieldLabel 返回本地化参数标签。
func WebAppFieldLabel(n WebAppInputNode) string {
	if n.FieldLabel != "" {
		return i18n.APITranslate("param_label", n.FieldLabel)
	}
	if n.VariableName != "" {
		return n.VariableName
	}
	return n.FieldName
}