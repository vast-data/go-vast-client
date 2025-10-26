package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type DNS struct {
	*BaseWidget
}

func NewDNS(db *database.Service) common.Widget {
	resourceType := "dns"
	listHeaders := []string{"id", "name", "domain_suffix", "vip"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &DNS{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (DNS) API(rest *VMSRest) VastResourceAPI {
	return rest.Dns
}
