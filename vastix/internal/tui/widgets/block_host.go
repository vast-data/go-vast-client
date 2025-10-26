package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type BlockHost struct {
	*BaseWidget
}

func NewBlockHost(db *database.Service) common.Widget {
	resourceType := "blockhosts"
	listHeaders := []string{"id", "name", "nqn", "tenant_name"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &BlockHost{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (BlockHost) API(rest *VMSRest) VastResourceAPI {
	return rest.BlockHosts
}
