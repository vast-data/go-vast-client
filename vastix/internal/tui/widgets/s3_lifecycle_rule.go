package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type S3LifeCycleRule struct {
	*BaseWidget
}

func NewS3LifeCycleRule(db *database.Service) common.Widget {
	resourceType := "s3lifecyclerules"
	listHeaders := []string{"id"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &S3LifeCycleRule{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (S3LifeCycleRule) API(rest *VMSRest) VastResourceAPI {
	return rest.S3LifeCycleRules
}

func (S3LifeCycleRule) GetNotAllowedNavigatorModes() []common.NavigatorMode {
	return []common.NavigatorMode{
		common.NavigatorModeCreate,
	}
}
