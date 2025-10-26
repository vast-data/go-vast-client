package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type ReplicationPeer struct {
	*BaseWidget
}

func NewReplicationPeer(db *database.Service) common.Widget {
	resourceType := "nativereplicationremotetargets"
	listHeaders := []string{"id"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &ReplicationPeer{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (ReplicationPeer) API(rest *VMSRest) VastResourceAPI {
	return rest.ReplicationPeers
}
