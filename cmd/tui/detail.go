package tui

import (
	"github.com/charmbracelet/lipgloss"
)

func detailKeyStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#22D3EE")).
		Bold(true).
		Padding(0, 1)
}

func sectionCard(title string, content string, width int) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6B7280")).
		Padding(1, 2).
		Width(width)
	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FBBF24")).
		Bold(true).
		Render(title)
	return style.Render(lipgloss.JoinVertical(lipgloss.Left, header, content))
}

func statusColor(status string) lipgloss.Color {
	switch status {
	case "active", "ready", "completed", "success":
		return lipgloss.Color("#22C55E")
	case "pending", "processing", "uploading":
		return lipgloss.Color("#FBBF24")
	case "failed", "error", "canceled":
		return lipgloss.Color("#EF4444")
	case "public":
		return lipgloss.Color("#22C55E")
	case "private":
		return lipgloss.Color("#6B7280")
	default:
		return lipgloss.Color("#9CA3AF")
	}
}

func statusBadge(status string) string {
	c := statusColor(status)
	badge := lipgloss.NewStyle().
		Foreground(c).
		Bold(true).
		Padding(0, 1).
		Render("● " + status)
	return badge
}

func typeBadge(modelType string) string {
	var c lipgloss.Color
	switch modelType {
	case "Checkpoint":
		c = lipgloss.Color("#8B5CF6")
	case "LoRA":
		c = lipgloss.Color("#EC4899")
	case "Controlnet":
		c = lipgloss.Color("#06B6D4")
	case "VAE":
		c = lipgloss.Color("#84CC16")
	case "UNet":
		c = lipgloss.Color("#F97316")
	case "Upscaler":
		c = lipgloss.Color("#EAB308")
	case "Detection":
		c = lipgloss.Color("#14B8A6")
	default:
		c = lipgloss.Color("#6B7280")
	}
	badge := lipgloss.NewStyle().
		Foreground(c).
		Bold(true).
		Padding(0, 1).
		Render(modelType)
	return badge
}
