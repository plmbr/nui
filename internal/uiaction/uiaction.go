// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package uiaction

// Action is a typed SPA control instruction emitted by nui built-in tools.
type Action struct {
	Type   string `json:"type"`
	Target string `json:"target,omitempty"` // navigate: customize | new_session | launch | schedules
	Theme  string `json:"theme,omitempty"`  // set_theme: dark | light
}

const (
	TypeNavigate  = "navigate"
	TypeSetTheme  = "set_theme"
	TypeRefreshUI = "refresh_ui" // soft refresh (e.g. after extension toggle)
)

const (
	TargetCustomize  = "customize"
	TargetNewSession = "new_session"
	TargetLaunch     = "launch"
	TargetSchedules  = "schedules"
)

const (
	ThemeDark  = "dark"
	ThemeLight = "light"
)

// Validate returns an error message if the action is invalid; empty string if ok.
func Validate(a Action) string {
	switch a.Type {
	case TypeNavigate:
		switch a.Target {
		case TargetCustomize, TargetNewSession, TargetLaunch, TargetSchedules:
			return ""
		default:
			return "navigate target must be customize, new_session, launch, or schedules"
		}
	case TypeSetTheme:
		switch a.Theme {
		case ThemeDark, ThemeLight:
			return ""
		default:
			return "set_theme theme must be dark or light"
		}
	case TypeRefreshUI:
		return ""
	default:
		return "unknown action type"
	}
}
