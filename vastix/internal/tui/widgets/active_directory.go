package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type ActiveDirectory struct {
	*BaseWidget
}

func NewActiveDirectory(db *database.Service) common.Widget {
	resourceType := "activedirectory"
	listHeaders := []string{"id", "ldap_id", "domain_name", "machine_account_name"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &ActiveDirectory{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (ActiveDirectory) API(rest *VMSRest) VastResourceAPI {
	return rest.ActiveDirectories
}
