package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type Snapshot struct {
	*BaseWidget
}

func NewSnapshot(db *database.Service) common.Widget {
	resourceType := "snapshots"
	listHeaders := []string{"id", "name", "path"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &Snapshot{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (Snapshot) API(rest *VMSRest) VastResourceAPI {
	return rest.Snapshots
}
