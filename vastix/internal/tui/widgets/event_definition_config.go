package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type EventDefinitionConfig struct {
	*BaseWidget
}

func NewEventDefinitionConfig(db *database.Service) common.Widget {
	resourceType := "eventdefinitionconfigs"
	listHeaders := []string{"id", "critical_value", "info_value", "syslog_host", "syslog_port", "syslog_protocol", "webhook_method", "audit_logs_retention"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPatch, "eventdefinitionconfigs/{id}", "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &EventDefinitionConfig{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (EventDefinitionConfig) API(rest *VMSRest) VastResourceAPI {
	return rest.EventDefinitionConfigs
}

func (w EventDefinitionConfig) GetAllowedNavigatorModes() []common.NavigatorMode {
	return []common.NavigatorMode{
		common.NavigatorModeList,
		common.NavigatorModeDetails,
	}
}
