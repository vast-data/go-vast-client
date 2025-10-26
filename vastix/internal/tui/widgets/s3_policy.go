package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type S3Policy struct {
	*BaseWidget
}

func NewS3Policy(db *database.Service) common.Widget {
	resourceType := "s3policies"
	listHeaders := []string{"id", "name", "enabled", "tenant_name"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &S3Policy{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (S3Policy) API(rest *VMSRest) VastResourceAPI {
	return rest.S3Policies
}
