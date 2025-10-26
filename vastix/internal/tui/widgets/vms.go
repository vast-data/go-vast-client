package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type Vms struct {
	*BaseWidget
}

func NewVms(db *database.Service) common.Widget {
	resourceType := "vms"
	listHeaders := []string{"id", "name", "mgmt_vip"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{
		NewGenerateAccessKey(db),
	}

	widget := &Vms{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (Vms) API(rest *VMSRest) VastResourceAPI {
	return rest.Vms
}
