package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type View struct {
	*BaseWidget
}

func NewView(db *database.Service) common.Widget {
	resourceType := "views"
	listHeaders := []string{"id", "name", "path", "policy", "protocols", "tenant_name"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &View{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (View) API(rest *VMSRest) VastResourceAPI {
	return rest.Views
}
