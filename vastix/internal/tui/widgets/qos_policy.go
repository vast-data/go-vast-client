package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type QosPolicy struct {
	*BaseWidget
}

func NewQosPolicy(db *database.Service) common.Widget {
	resourceType := "qospolicies"
	listHeaders := []string{"id", "name", "policy_type", "is_default", "tenant_name"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &QosPolicy{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (QosPolicy) API(rest *VMSRest) VastResourceAPI {
	return rest.QosPolicies
}

func (QosPolicy) GetNotAllowedNavigatorModes() []common.NavigatorMode {
	return []common.NavigatorMode{
		common.NavigatorModeCreate,
	}
}
