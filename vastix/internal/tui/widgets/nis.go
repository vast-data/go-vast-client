package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type NIS struct {
	*BaseWidget
}

func NewNIS(db *database.Service) common.Widget {
	resourceType := "nis"
	listHeaders := []string{"id", "name", "domain_name"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &NIS{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (NIS) API(rest *VMSRest) VastResourceAPI {
	return rest.Nis
}
