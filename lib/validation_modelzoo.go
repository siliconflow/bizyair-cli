package lib

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
)

type ModelzooValidationRule interface {
	Evaluate(raw string) error
}

func DeriveModelzooRules(p ModelzooInputParam) []ModelzooValidationRule {
	var rules []ModelzooValidationRule

	if p.Required {
		rules = append(rules, modelzooRequiredRule{label: p.FieldLabel})
	}

	switch p.VariableType {
	case "number", "float", "integer":
		rules = append(rules, modelzooNumberRule{label: p.FieldLabel, vtype: p.VariableType})
	case "image", "video":
		rules = append(rules, modelzooURLRule{label: p.FieldLabel})
	case "boolean":
		rules = append(rules, modelzooBooleanRule{label: p.FieldLabel})
	}

	// 媒体（图片/视频）由用户填写 URL 或上传，枚举选项仅是预设建议，
	// 不能作为可选范围校验；其余类型的枚举选项才作为必选列表约束。
	media := p.VariableType == "image" || p.VariableType == "video"
	if p.FieldOptions != nil && !media {
		rules = append(rules, modelzooRangeRules(p)...)
		ev := p.FieldOptions.EnumValues()
		if len(ev) > 0 {
			rules = append(rules, modelzooEnumRule{label: p.FieldLabel, options: ev})
		}
	}

	return rules
}

func modelzooRangeRules(p ModelzooInputParam) []ModelzooValidationRule {
	var rules []ModelzooValidationRule
	fo := p.FieldOptions
	if fo == nil {
		return nil
	}

	if fo.MinPixels != nil || fo.MaxPixels != nil {
		rules = append(rules, modelzooRangeRule{
			label: p.FieldLabel,
			min:   intPtrToFloat64(fo.MinPixels),
			max:   intPtrToFloat64(fo.MaxPixels),
		})
	}

	if fo.MinAspectRatio != nil || fo.MaxAspectRatio != nil {
		rules = append(rules, modelzooRangeRule{
			label: p.FieldLabel,
			min:   fo.MinAspectRatio,
			max:   fo.MaxAspectRatio,
		})
	}
	return rules
}

func intPtrToFloat64(i *int) *float64 {
	if i == nil {
		return nil
	}
	f := float64(*i)
	return &f
}

type modelzooRequiredRule struct {
	label string
}

func (r modelzooRequiredRule) Evaluate(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return i18n.NewError("tui.task_model.param_required",
			map[string]any{"Name": r.label}, nil)
	}
	return nil
}

type modelzooNumberRule struct {
	label string
	vtype string
}

func (r modelzooNumberRule) Evaluate(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	val, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return i18n.NewError("tui.task_model.invalid_number",
			map[string]any{"Name": r.label, "Value": raw, "Cause": err}, nil)
	}
	if math.IsNaN(val) || math.IsInf(val, 0) {
		return i18n.NewError("tui.task_model.not_finite",
			map[string]any{"Name": r.label, "Value": raw}, nil)
	}
	if r.vtype == "integer" {
		if _, err := strconv.ParseInt(raw, 10, 64); err != nil {
			return i18n.NewError("tui.task_model.invalid_number",
				map[string]any{"Name": r.label, "Value": raw, "Cause": err}, nil)
		}
	}
	return nil
}

type modelzooRangeRule struct {
	label string
	min   *float64
	max   *float64
}

func (r modelzooRangeRule) Evaluate(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	val, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil
	}
	if r.min != nil && val < *r.min {
		return i18n.NewError("tui.task_model.range_error",
			map[string]any{"Name": r.label, "Value": raw, "Min": modelzooFormatFloat(*r.min), "Max": modelzooFormatFloatOrNone(r.max)}, nil)
	}
	if r.max != nil && val > *r.max {
		return i18n.NewError("tui.task_model.range_error",
			map[string]any{"Name": r.label, "Value": raw, "Min": modelzooFormatFloatOrNone(r.min), "Max": modelzooFormatFloat(*r.max)}, nil)
	}
	return nil
}

type modelzooEnumRule struct {
	label   string
	options []any
}

func (r modelzooEnumRule) Evaluate(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	for _, o := range r.options {
		if fmt.Sprintf("%v", o) == raw {
			return nil
		}
	}
	optStrs := make([]string, 0, len(r.options))
	for _, o := range r.options {
		optStrs = append(optStrs, fmt.Sprintf("%v", o))
	}
	return i18n.NewError("tui.task_model.enum_error",
		map[string]any{"Name": r.label, "Value": raw, "Options": strings.Join(optStrs, "、")}, nil)
}

type modelzooURLRule struct {
	label string
}

func (r modelzooURLRule) Evaluate(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	if !IsHTTPURL(raw) {
		return i18n.NewError("tui.task_model.url_invalid",
			map[string]any{"Type": r.label}, nil)
	}
	return nil
}

type modelzooBooleanRule struct {
	label string
}

func (r modelzooBooleanRule) Evaluate(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	if _, err := strconv.ParseBool(raw); err != nil {
		return i18n.NewError("tui.task_model.invalid_number",
			map[string]any{"Name": r.label, "Value": raw, "Cause": err}, nil)
	}
	return nil
}

func modelzooFormatFloat(v float64) string {
	if v == float64(int64(v)) {
		return fmt.Sprintf("%.0f", v)
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", v), "0"), ".")
}

func modelzooFormatFloatOrNone(p *float64) string {
	if p == nil {
		return "∞"
	}
	return modelzooFormatFloat(*p)
}

func ValidateModelzooParam(rules []ModelzooValidationRule, raw string) error {
	for _, r := range rules {
		if err := r.Evaluate(raw); err != nil {
			return err
		}
	}
	return nil
}
