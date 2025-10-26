package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type Quota struct {
	*BaseWidget
}

func NewQuota(db *database.Service) common.Widget {
	resourceType := "quotas"
	listHeaders := []string{"id", "name", "path", "tenant_name"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &Quota{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (Quota) API(rest *VMSRest) VastResourceAPI {
	return rest.Quotas
}

func (Quota) GetNotAllowedNavigatorModes() []common.NavigatorMode {
	return []common.NavigatorMode{
		common.NavigatorModeCreate,
	}
}
