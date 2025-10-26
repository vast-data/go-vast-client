package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type ProtectionPolicy struct {
	*BaseWidget
}

func NewProtectionPolicy(db *database.Service) common.Widget {
	resourceType := "protectionpolicies"
	listHeaders := []string{"id", "name", "internal", "is_local", "is_on_schedule", "is_sync_replication"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &ProtectionPolicy{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (ProtectionPolicy) API(rest *VMSRest) VastResourceAPI {
	return rest.ProtectionPolicies
}
