package common

type NavigatorMode int

const (
	NavigatorModeList NavigatorMode = iota
	NavigatorModeCreate
	NavigatorModeDelete
	NavigatorModeDetails
	NavigatorModeExtra // This is for extra navigators that don't fit the standard modes
)

func (n NavigatorMode) String() string {
	switch n {
	case NavigatorModeList:
		return "list"
	case NavigatorModeCreate:
		return "create"
	case NavigatorModeDelete:
		return "delete"
	case NavigatorModeDetails:
		return "details"
	case NavigatorModeExtra:
		return "extra"
	default:
		return "unknown"
	}
}

type ExtraNavigatorMode int

const (
	ExtraNavigatorModeList ExtraNavigatorMode = iota
	ExtraNavigatorModeCreate
	ExtraNavigatorModeDelete
	ExtraNavigatorModeDetails
	ExtraNavigatorModePrompt
)

func (n ExtraNavigatorMode) String() string {
	switch n {
	case ExtraNavigatorModeList:
		return "list"
	case ExtraNavigatorModeCreate:
		return "create"
	case ExtraNavigatorModeDelete:
		return "delete"
	case ExtraNavigatorModeDetails:
		return "details"
	case ExtraNavigatorModePrompt:
		return "prompt"
	default:
		return "unknown"
	}
}
