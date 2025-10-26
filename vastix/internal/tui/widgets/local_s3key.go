package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type LocalS3Key struct {
	*BaseWidget
}

func NewLocalS3Key(db *database.Service) common.Widget {
	resourceType := "locals3keys"
	listHeaders := []string{"id", "access_key", "enabled", "creation_time"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &LocalS3Key{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (LocalS3Key) API(rest *VMSRest) VastResourceAPI {
	return rest.LocalS3Keys
}

func (LocalS3Key) GetNotAllowedNavigatorModes() []common.NavigatorMode {
	return []common.NavigatorMode{
		common.NavigatorModeCreate,
		common.NavigatorModeDelete,
	}
}
