package lib

import "strings"

// SeriesFromEndpoint 根据端点前缀推断模型所属系列。
// 当 price_table_flat 接口未返回 series 字段时用作兜底，供所有消费者复用。
func SeriesFromEndpoint(endpoint string) string {
	if endpoint == "" {
		return ""
	}
	parts := strings.SplitN(endpoint, "/", 2)
	prefix := parts[0]

	switch {
	case strings.HasPrefix(prefix, "nano-banana"):
		return "nano_banana"
	case strings.HasPrefix(prefix, "gemini-omni"):
		return "gemini_omni"
	case strings.HasPrefix(prefix, "gemini-vision"):
		return "gemini_vision"
	case strings.HasPrefix(prefix, "gemini-"):
		return "gemini"
	case strings.HasPrefix(prefix, "google-veo"):
		return "veo"
	case strings.HasPrefix(prefix, "gpt-image"):
		return "gpt_image"
	case strings.HasPrefix(prefix, "seedream"):
		return "seedream"
	case strings.HasPrefix(prefix, "seedance"):
		return "seedance"
	case strings.HasPrefix(prefix, "dreamactor"):
		return "dream_actor"
	case strings.HasPrefix(prefix, "wan-2-7-image"):
		return "wan_image"
	case strings.HasPrefix(prefix, "wan-image"):
		return "wan_image"
	case strings.HasPrefix(prefix, "wan-video"), strings.HasPrefix(prefix, "wan-"):
		return "wan_video"
	case strings.HasPrefix(prefix, "qwen-image"):
		return "qwen_image"
	case strings.HasPrefix(prefix, "happyhorse"):
		return "happy_horse"
	case strings.HasPrefix(prefix, "hailuo"):
		return "hailuo"
	case strings.HasPrefix(prefix, "kling"):
		return "kling"
	case strings.HasPrefix(prefix, "grok-2-image"), strings.HasPrefix(prefix, "grok-image"):
		return "grok_image"
	case strings.HasPrefix(prefix, "grok-imagine"):
		return "grok_video"
	case strings.HasPrefix(prefix, "vidu"):
		return "vidu"
	case strings.HasPrefix(prefix, "skyreels"):
		return "skyreels"
	case strings.HasPrefix(prefix, "mureka"):
		return "mureka"
	case strings.HasPrefix(prefix, "pixal3d"), strings.HasPrefix(prefix, "tripo3d"):
		return "3d_generation"
	case strings.HasPrefix(prefix, "minimax"):
		return "minimax"
	case strings.HasPrefix(prefix, "siliconflow"):
		return "siliconflow"
	}

	s := strings.TrimSuffix(prefix, "-official")
	s = strings.TrimSuffix(s, "-channel")
	return s
}