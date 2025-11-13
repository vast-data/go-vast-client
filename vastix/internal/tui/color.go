package tui

// This file re-exports colors from the centralized colors package
// for backwards compatibility with existing TUI code.
// New code should import "vastix/internal/colors" directly.

import "vastix/internal/colors"

// Re-export all colors for backwards compatibility
const (
	Black           = colors.Black
	DarkRed         = colors.BrightRed
	Red             = colors.Red
	Purple          = colors.Purple
	Orange          = colors.Orange
	BurntOrange     = colors.BurntOrange
	Yellow          = colors.Yellow
	Green           = colors.Green
	Turquoise       = colors.Turquoise
	DarkGreen       = colors.DarkGreen
	LightGreen      = colors.LightGreen
	GreenBlue       = colors.GreenBlue
	DeepBlue        = colors.DeepBlue
	LightBlue       = colors.LightBlue
	LightishBlue    = colors.LightishBlue
	Blue            = colors.Blue
	DarkBlue        = colors.VeryDarkBlue
	Violet          = colors.Violet
	Grey            = colors.Grey
	LightGrey       = colors.LightGrey
	LighterGrey     = colors.LighterGrey
	EvenLighterGrey = colors.EvenLighterGrey
	DarkGrey        = colors.DarkGrey
	White           = colors.White
	OffWhite        = colors.OffWhite
	HotPink         = colors.HotPink
)

var (
	DebugLogLevel                = colors.DebugLogLevel
	InfoLogLevel                 = colors.InfoLogLevel
	ErrorLogLevel                = colors.ErrorLogLevel
	WarnLogLevel                 = colors.WarnLogLevel
	LogRecordAttributeKey        = colors.LogRecordAttributeKey
	HelpKey                      = colors.HelpKey
	HelpDesc                     = colors.HelpDesc
	InactivePreviewBorder        = colors.InactivePreviewBorder
	CurrentBackground            = colors.CurrentBackground
	CurrentForeground            = colors.CurrentForeground
	SelectedBackground           = colors.SelectedBackground
	SelectedForeground           = colors.SelectedForeground
	CurrentAndSelectedBackground = colors.CurrentAndSelectedBackground
	CurrentAndSelectedForeground = colors.CurrentAndSelectedForeground
	TitleColor                   = colors.TitleColor
	GroupReportBackgroundColor   = colors.GroupReportBackgroundColor
	TaskSummaryBackgroundColor   = colors.TaskSummaryBackgroundColor
	ScrollPercentageBackground   = colors.ScrollPercentageBackground
)
