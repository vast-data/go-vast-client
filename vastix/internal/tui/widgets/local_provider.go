package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type LocalProvider struct {
	*BaseWidget
}

func NewLocalProvider(db *database.Service) common.Widget {
	resourceType := "localproviders"
	listHeaders := []string{"id", "name"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &LocalProvider{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (LocalProvider) API(rest *VMSRest) VastResourceAPI {
	return rest.LocalProviders
}
