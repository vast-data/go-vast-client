package widgets

import (
	"net/http"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
)

type S3ReplicationPeer struct {
	*BaseWidget
}

func NewS3ReplicationPeer(db *database.Service) common.Widget {
	resourceType := "replicationtargets"
	listHeaders := []string{"id", "name"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &S3ReplicationPeer{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (S3ReplicationPeer) API(rest *VMSRest) VastResourceAPI {
	return rest.S3replicationPeers
}
