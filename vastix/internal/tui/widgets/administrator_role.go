package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type AdministratorRole struct {
	*BaseWidget
}

func NewAdministratorRole(db *database.Service) common.Widget {
	resourceType := "roles"
	listHeaders := []string{"id", "name", "tenant"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &AdministratorRole{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (AdministratorRole) API(rest *VMSRest) VastResourceAPI {
	return rest.Roles
}
