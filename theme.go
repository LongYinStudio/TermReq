package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
)

const (
	colorBg           = "#1A1B26"
	colorSurface      = "#1A1B26"
	colorSurface2     = "#1A1B26"
	colorSurface3     = "#1A1B26"
	colorBorder       = "#3B4261"
	colorBorderSoft   = "#2D334A"
	colorAccent       = "#F6C177"
	colorAccent2      = "#7BDFF2"
	colorText         = "#E5E9F0"
	colorMuted        = "#8796A8"
	colorError        = "#F7768E"
	colorWarn         = "#E0AF68"
	colorSuccess      = "#A7D46F"
	colorInfo         = "#7BDFF2"
	colorMethodGet    = "#A7D46F"
	colorMethodPost   = "#7BDFF2"
	colorMethodPut    = "#E0AF68"
	colorMethodPatch  = "#F6C177"
	colorMethodDelete = "#F7768E"
)

type uiStyles struct {
	app              lipgloss.Style
	headerBar        lipgloss.Style
	headerBadge      lipgloss.Style
	headerSubtle     lipgloss.Style
	headerLabel      lipgloss.Style
	headerValue      lipgloss.Style
	messageBar       lipgloss.Style
	messageInfo      lipgloss.Style
	messageSuccess   lipgloss.Style
	messageError     lipgloss.Style
	panel            lipgloss.Style
	panelActive      lipgloss.Style
	panelTitle       lipgloss.Style
	panelTitleActive lipgloss.Style
	methodChip       lipgloss.Style
	methodSelected   lipgloss.Style
	label            lipgloss.Style
	labelActive      lipgloss.Style
	section          lipgloss.Style
	sectionRule      lipgloss.Style
	sectionMeta      lipgloss.Style
	field            lipgloss.Style
	fieldActive      lipgloss.Style
	fieldSelected    lipgloss.Style
	textareaWrap     lipgloss.Style
	textareaActive   lipgloss.Style
	tableHeader      lipgloss.Style
	rowIndex         lipgloss.Style
	rowIndexActive   lipgloss.Style
	hint             lipgloss.Style
	metricLabel      lipgloss.Style
	metricValue      lipgloss.Style
	status2xx        lipgloss.Style
	status3xx        lipgloss.Style
	status4xx        lipgloss.Style
	status5xx        lipgloss.Style
	text             lipgloss.Style
	muted            lipgloss.Style
	helpBar          lipgloss.Style
}

var styles = newUIStyles()

func newUIStyles() uiStyles {
	return uiStyles{
		app: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorText)),
		headerBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorText)).
			Padding(0, 1),
		headerBadge: lipgloss.NewStyle().
			Background(lipgloss.Color(colorAccent)).
			Foreground(lipgloss.Color(colorBg)).
			Bold(true).
			Padding(0, 1),
		headerSubtle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMuted)).
			Italic(true),
		headerLabel: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMuted)).
			Bold(true),
		headerValue: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorAccent2)).
			Bold(true),
		messageBar: lipgloss.NewStyle().
			Padding(0, 1),
		messageInfo: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorInfo)).
			Bold(true),
		messageSuccess: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorSuccess)).
			Bold(true),
		messageError: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorError)).
			Bold(true),
		panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorBorderSoft)).
			Padding(0, 1),
		panelActive: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorBorder)).
			Padding(0, 1),
		panelTitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorText)).
			Bold(true).
			Padding(0, 1),
		panelTitleActive: lipgloss.NewStyle().
			Background(lipgloss.Color(colorAccent)).
			Foreground(lipgloss.Color(colorBg)).
			Bold(true).
			Padding(0, 1),
		methodChip: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMuted)).
			Padding(0, 1),
		methodSelected: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBg)).
			Background(lipgloss.Color(colorAccent)).
			Bold(true).
			Padding(0, 1),
		label: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMuted)).
			Bold(true).
			Width(9),
		labelActive: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorAccent)).
			Bold(true).
			Width(9),
		section: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorAccent2)).
			Bold(true),
		sectionRule: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBorder)).
			Faint(true),
		sectionMeta: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMuted)),
		field: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorText)).
			Padding(0, 1),
		fieldActive: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorText)).
			Padding(0, 1),
		fieldSelected: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorText)).
			Padding(0, 1),
		textareaWrap:   lipgloss.NewStyle(),
		textareaActive: lipgloss.NewStyle(),
		tableHeader: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMuted)).
			Bold(true),
		rowIndex: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMuted)),
		rowIndexActive: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorAccent)).
			Bold(true),
		hint: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMuted)),
		metricLabel: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMuted)).
			Bold(true),
		metricValue: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorText)).
			Bold(true),
		status2xx: lipgloss.NewStyle().Foreground(lipgloss.Color(colorSuccess)).Bold(true),
		status3xx: lipgloss.NewStyle().Foreground(lipgloss.Color(colorWarn)).Bold(true),
		status4xx: lipgloss.NewStyle().Foreground(lipgloss.Color("#FF9E64")).Bold(true),
		status5xx: lipgloss.NewStyle().Foreground(lipgloss.Color(colorError)).Bold(true),
		text:      lipgloss.NewStyle().Foreground(lipgloss.Color(colorText)),
		muted:     lipgloss.NewStyle().Foreground(lipgloss.Color(colorMuted)),
		helpBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorText)).
			Padding(0, 1),
	}
}

func newHelpModel() help.Model {
	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().
		Background(lipgloss.Color(colorAccent2)).
		Foreground(lipgloss.Color(colorBg)).
		Bold(true)
	h.Styles.ShortDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorText))
	h.Styles.ShortSeparator = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorMuted))
	h.Styles.Ellipsis = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorMuted))
	return h
}

func methodLipgloss(method string) lipgloss.Style {
	switch method {
	case "GET":
		return styles.methodSelected.Foreground(lipgloss.Color(colorBg)).Background(lipgloss.Color(colorMethodGet))
	case "POST":
		return styles.methodSelected.Foreground(lipgloss.Color(colorBg)).Background(lipgloss.Color(colorMethodPost))
	case "PUT":
		return styles.methodSelected.Foreground(lipgloss.Color(colorBg)).Background(lipgloss.Color(colorMethodPut))
	case "PATCH":
		return styles.methodSelected.Foreground(lipgloss.Color(colorBg)).Background(lipgloss.Color(colorMethodPatch))
	case "DELETE":
		return styles.methodSelected.Foreground(lipgloss.Color(colorBg)).Background(lipgloss.Color(colorMethodDelete))
	default:
		return styles.methodSelected
	}
}

func messageLipgloss(level string) lipgloss.Style {
	switch level {
	case "success":
		return styles.messageSuccess
	case "error":
		return styles.messageError
	default:
		return styles.messageInfo
	}
}

func statusLipgloss(status string) lipgloss.Style {
	if status == "" {
		return styles.muted
	}
	switch status[0] {
	case '2':
		return styles.status2xx
	case '3':
		return styles.status3xx
	case '4':
		return styles.status4xx
	case '5':
		return styles.status5xx
	default:
		return styles.text
	}
}

func sectionLine(title string, width int) string {
	return sectionLineMeta(title, "", width)
}

func sectionLineMeta(title, meta string, width int) string {
	labelView := styles.section.Render(strings.ToUpper(title))
	metaView := ""
	if meta != "" {
		metaView = styles.sectionMeta.Render(meta)
	}

	usedWidth := lipgloss.Width(labelView)
	if metaView != "" {
		usedWidth += lipgloss.Width(metaView) + 2
	}

	if width <= usedWidth+1 {
		if metaView == "" {
			return labelView
		}
		return lipgloss.JoinHorizontal(lipgloss.Top, labelView, " ", metaView)
	}

	ruleWidth := width - usedWidth - 1
	parts := []string{
		labelView,
		" ",
		styles.sectionRule.Render(strings.Repeat("─", ruleWidth)),
	}
	if metaView != "" {
		parts = append(parts, " ", metaView)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}
