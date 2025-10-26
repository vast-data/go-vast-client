package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type LDAP struct {
	*BaseWidget
}

func NewLDAP(db *database.Service) common.Widget {
	resourceType := "ldaps"
	listHeaders := []string{"id", "domain_name", "binddn"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &LDAP{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (LDAP) API(rest *VMSRest) VastResourceAPI {
	return rest.Ldaps
}
