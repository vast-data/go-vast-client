package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type Group struct {
	*BaseWidget
}

func NewGroup(db *database.Service) common.Widget {
	resourceType := "groups"
	listHeaders := []string{"id", "name", "gid", "sid"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &Group{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (Group) API(rest *VMSRest) VastResourceAPI {
	return rest.Groups
}
