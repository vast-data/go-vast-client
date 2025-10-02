package untyped

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/vast-data/go-vast-client/core"
)

// +apityped:details:GET=users
// +apityped:upsert:POST=users
// +apiuntyped:extraMethod:PATCH=/users/{id}/tenant_data/
// +apiuntyped:extraMethod:GET=/users/{id}/tenant_data/
// +apityped:extraMethod:PATCH=/users/{id}/tenant_data/
// +apityped:extraMethod:GET=/users/{id}/tenant_data/
type User struct {
	*core.VastResource
}

func (u *User) CopyWithContext(ctx context.Context, params core.Params) error {
	path := fmt.Sprintf("%s/copy", u.GetResourcePath())
	result, err := core.Request[core.Record](ctx, u, http.MethodPost, path, nil, params)
	if err != nil {
		return err
	}
	task := asyncResultFromRecord(ctx, result, u.Rest)
	if _, err := task.Wait(3 * time.Minute); err != nil {
		return fmt.Errorf("failed to copy users: %w", err)
	}
	return nil
}

func (u *User) Copy(params core.Params) error {
	return u.CopyWithContext(u.Rest.GetCtx(), params)
}
