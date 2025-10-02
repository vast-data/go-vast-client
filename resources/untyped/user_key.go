package untyped

import (
	"context"
	"fmt"
	"net/http"

	"github.com/vast-data/go-vast-client/core"
)

type UserKey struct {
	*core.VastResource
}

func (uk *UserKey) CreateKeyWithContext(ctx context.Context, userId int64) (core.Record, error) {
	path := core.BuildResourcePathWithID("users", userId, "access_keys")
	return core.Request[core.Record](ctx, uk, http.MethodPost, path, nil, nil)
}

func (uk *UserKey) CreateKey(userId int64) (core.Record, error) {
	return uk.CreateKeyWithContext(uk.Rest.GetCtx(), userId)
}

func (uk *UserKey) EnableKeyWithContext(ctx context.Context, userId int64, accessKey string) (core.EmptyRecord, error) {
	path := fmt.Sprintf(uk.GetResourcePath(), userId)
	params := core.Params{"access_key": accessKey, "enabled": true}
	return core.Request[core.EmptyRecord](ctx, uk, http.MethodPatch, path, nil, params)
}

func (uk *UserKey) EnableKey(userId int64, accessKey string) (core.EmptyRecord, error) {
	return uk.EnableKeyWithContext(uk.Rest.GetCtx(), userId, accessKey)
}

func (uk *UserKey) DisableKeyWithContext(ctx context.Context, userId int64, accessKey string) (core.EmptyRecord, error) {
	path := fmt.Sprintf(uk.GetResourcePath(), userId)
	params := core.Params{"access_key": accessKey, "enabled": false}
	return core.Request[core.EmptyRecord](ctx, uk, http.MethodPatch, path, nil, params)
}

func (uk *UserKey) DisableKey(userId int64, accessKey string) (core.EmptyRecord, error) {
	return uk.DisableKeyWithContext(uk.Rest.GetCtx(), userId, accessKey)
}

func (uk *UserKey) DeleteKeyWithContext(ctx context.Context, userId int64, accessKey string) (core.EmptyRecord, error) {
	path := fmt.Sprintf(uk.GetResourceType(), userId)
	return core.Request[core.EmptyRecord](ctx, uk, http.MethodDelete, path, nil, core.Params{"access_key": accessKey})
}

func (uk *UserKey) DeleteKey(userId int64, accessKey string) (core.EmptyRecord, error) {
	return uk.DeleteKeyWithContext(uk.Rest.GetCtx(), userId, accessKey)
}
