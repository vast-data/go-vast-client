package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type BlockHostMapping struct {
	*BaseWidget
}

func NewBlockHostMapping(db *database.Service) common.Widget {
	resourceType := "blockhostvolumes"
	listHeaders := []string{"id", "volume", "block_host"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPatch, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &BlockHostMapping{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (BlockHostMapping) API(rest *VMSRest) VastResourceAPI {
	return rest.BlockHostMappings
}

func (w BlockHostMapping) GetNotAllowedNavigatorModes() []common.NavigatorMode {
	return []common.NavigatorMode{
		common.NavigatorModeCreate,
	}
}
