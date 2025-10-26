package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type KafkaBroker struct {
	*BaseWidget
}

func NewKafkaBroker(db *database.Service) common.Widget {
	resourceType := "kafkabrokers"
	listHeaders := []string{"id"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &KafkaBroker{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (KafkaBroker) API(rest *VMSRest) VastResourceAPI {
	return rest.KafkaBrokers
}
