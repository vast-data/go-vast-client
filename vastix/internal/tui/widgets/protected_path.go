package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type ProtectedPath struct {
	*BaseWidget
}

func NewProtectedPath(db *database.Service) common.Widget {
	resourceType := "protectedpaths"
	listHeaders := []string{"id", "name", "protection_policy_name"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &ProtectedPath{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (ProtectedPath) API(rest *VMSRest) VastResourceAPI {
	return rest.ProtectedPaths
}
