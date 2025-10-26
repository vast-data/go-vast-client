package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type Volume struct {
	*BaseWidget
}

func NewVolume(db *database.Service) common.Widget {
	resourceType := "volumes"
	listHeaders := []string{"id", "name", "size", "tenant_name", "subsystem_name", "qos_policy"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &Volume{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (Volume) API(rest *VMSRest) VastResourceAPI {
	return rest.Volumes
}
