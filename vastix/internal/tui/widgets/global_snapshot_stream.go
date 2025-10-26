package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type GlobalSnapshotStream struct {
	*BaseWidget
}

func NewGlobalSnapshotStream(db *database.Service) common.Widget {
	resourceType := "globalsnapstreams"
	listHeaders := []string{"id", "name", "loanee_root_path"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &GlobalSnapshotStream{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (GlobalSnapshotStream) API(rest *VMSRest) VastResourceAPI {
	return rest.GlobalSnapshotStreams
}
