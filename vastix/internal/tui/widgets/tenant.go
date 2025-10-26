package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type Tenant struct {
	*BaseWidget
}

func NewTenant(db *database.Service) common.Widget {
	resourceType := "tenants"
	listHeaders := []string{"id", "name", "client_ip_ranges"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &Tenant{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (Tenant) API(rest *VMSRest) VastResourceAPI {
	return rest.Tenants
}
