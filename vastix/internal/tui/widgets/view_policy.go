package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type ViewPolicy struct {
	*BaseWidget
}

func NewViewPolicy(db *database.Service) common.Widget {
	resourceType := "viewpolicies"
	listHeaders := []string{"id", "name", "access_flavor", "flavor"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &ViewPolicy{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (ViewPolicy) API(rest *VMSRest) VastResourceAPI {
	return rest.ViewPolicies
}
