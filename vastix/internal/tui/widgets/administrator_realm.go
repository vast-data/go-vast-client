package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type AdministratorRealm struct {
	*BaseWidget
}

func NewAdministratorRealm(db *database.Service) common.Widget {
	resourceType := "realms"
	listHeaders := []string{"id", "name"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &AdministratorRealm{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (AdministratorRealm) API(rest *VMSRest) VastResourceAPI {
	return rest.Realms
}
