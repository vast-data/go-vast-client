package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type EncryptionGroup struct {
	*BaseWidget
}

func NewEncryptionGroup(db *database.Service) common.Widget {
	resourceType := "encryptiongroups"
	listHeaders := []string{"id", "crn", "state"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &EncryptionGroup{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (EncryptionGroup) API(rest *VMSRest) VastResourceAPI {
	return rest.EncryptionGroups
}

func (*EncryptionGroup) GetNotAllowedNavigatorModes() []common.NavigatorMode {
	return []common.NavigatorMode{
		common.NavigatorModeCreate,
	}
}
