package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type EventDefinition struct {
	*BaseWidget
}

func NewEventDefinition(db *database.Service) common.Widget {
	resourceType := "eventdefinitions"
	listHeaders := []string{"id", "name", "event_type", "object_type", "severity"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPatch, "eventdefinitions/{id}", "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &EventDefinition{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (EventDefinition) API(rest *VMSRest) VastResourceAPI {
	return rest.EventDefinitions
}

func (w EventDefinition) GetAllowedNavigatorModes() []common.NavigatorMode {
	return []common.NavigatorMode{
		common.NavigatorModeList,
		common.NavigatorModeDetails,
	}
}
