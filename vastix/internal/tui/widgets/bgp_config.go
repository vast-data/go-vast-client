package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type BGPConfig struct {
	*BaseWidget
}

func NewBGPConfig(db *database.Service) common.Widget {
	resourceType := "bgpconfigs"
	listHeaders := []string{"id"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &BGPConfig{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (BGPConfig) API(rest *VMSRest) VastResourceAPI {
	return rest.BGPConfigs
}

func (*BGPConfig) GetNotAllowedNavigatorModes() []common.NavigatorMode {
	return []common.NavigatorMode{
		common.NavigatorModeCreate,
	}
}
