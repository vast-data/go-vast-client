package vast_client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/hashicorp/go-version"
)

//  ######################################################
//              FINAL VAST RESOURCES
//  ######################################################

type VastResourceType interface {
	Dummy |
	Version |
	Quota |
	View |
	VipPool |
	User |
	UserKey |
	Snapshot |
	BlockHost |
	Volume |
	VTask |
	BlockHostMapping |
	Cnode |
	QosPolicy |
	Dns |
	ViewPolicy |
	Group |
	Nis |
	Tenant |
	Ldap |
	S3LifeCycleRule |
	ActiveDirectory |
	S3Policy |
	ProtectedPath |
	GlobalSnapshotStream |
	ReplicationPeers |
	ProtectionPolicy |
	S3replicationPeers |
	Realm |
	Role |
	NonLocalUser |
	NonLocalGroup |
	NonLocalUserKey |
	ApiToken |
	KafkaBroker |
	Manager |
	Folder |
	EventDefinition |
	EventDefinitionConfig |
	BGPConfig |
	Vms |
	Topic |
	LocalProvider |
	LocalS3Key |
	EncryptionGroup |
	SamlConfig |
	Kerberos |
	Cluster
}

// ------------------------------------------------------

type Dummy struct {
	*VastResource
}

// ------------------------------------------------------

type OpenAPI struct {
	session RESTSession
}

// FetchSchemaV2 retrieves the Swagger 2.0 (OpenAPI v2) schema from the remote VAST backend.
//
// It performs an authenticated request to the OpenAPI schema endpoint and attempts to unmarshal
// the returned JSON into a structured `openapi2.T` object, which includes metadata about the API,
// available paths, definitions (models), parameters, responses, and security schemes.
//
// The returned object follows the OpenAPI 2.0 specification, with fields like:
//   - Swagger: version (must be "2.0")
//   - Info: API title, version, contact, and license
//   - Host: API hostname (e.g. "domain.com")
//   - BasePath: base path for endpoints (e.g. "/api")
//   - Paths: map of endpoint paths and operations
//   - Definitions: schema definitions for reusable data structures
//   - Parameters, Responses, SecurityDefinitions, etc.
//
// Returns:
//
//	*openapi2.T: structured representation of the OpenAPI 2.0 document
//	error: if fetching or unmarshalling fails
//
// Example usage:
//
//	schema, err := client.FetchSchema(ctx)
//	if err != nil {
//	    log.Fatalf("error loading OpenAPI schema: %v", err)
//	}
//	fmt.Println(schema.Paths["/users/"].Get.Summary)
func (o *OpenAPI) FetchSchemaV2(ctx context.Context) (*openapi2.T, error) {
	record, err := o.session.(*VMSSession).fetchSchema(ctx)
	if err != nil {
		return nil, err
	}

	raw, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}

	var doc openapi2.T
	if err = json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// FetchSchemaV3 retrieves the OpenAPI v3.0 schema for the VAST backend by first
// fetching the Swagger 2.0 (OpenAPI v2) document and converting it to v3 format.
//
// This function is useful when working with tools or generators (e.g., Terraform
// schema generators) that expect OpenAPI 3.0-compliant schemas.
//
// The returned OpenAPI v3 document includes:
//   - Components: schemas, responses, parameters, etc.
//   - Paths: endpoint definitions
//   - Info: metadata about the API
//
// Returns:
//
//	*openapi3.T: the converted OpenAPI v3 schema
//	error: if either fetching the v2 schema or converting to v3 fails
//
// Example:
//
//	doc, err := client.OpenAPI.FetchSchemaV3(ctx)
//	if err != nil {
//	    log.Fatalf("failed to fetch schema: %v", err)
//	}
//	fmt.Println(doc.Components.Schemas["User"].Value.Type)
func (o *OpenAPI) FetchSchemaV3(ctx context.Context) (*openapi3.T, error) {
	schemaV2, err := o.FetchSchemaV2(ctx)
	if err != nil {
		return nil, err
	}

	doc, err := openapi2conv.ToV3(schemaV2)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to OpenAPI v3: %w", err)
	}
	return doc, nil
}

// ------------------------------------------------------

type Version struct {
	*VastResource
}

func (v *Version) GetVersionWithContext(ctx context.Context) (*version.Version, error) {
	if v.rest.cachedVersion != nil {
		return v.rest.cachedVersion, nil
	}
	result, err := v.ListWithContext(ctx, Params{"status": "success"})
	if err != nil {
		return nil, err
	}
	truncatedVersion, _ := sanitizeVersion(result[0]["sys_version"].(string))
	clusterVersion, err := version.NewVersion(truncatedVersion)
	if err != nil {
		return nil, err
	}
	//We only work with core version
	v.rest.cachedVersion = clusterVersion.Core()
	return v.rest.cachedVersion, nil
}

func (v *Version) GetVersion() (*version.Version, error) {
	return v.GetVersionWithContext(v.rest.ctx)
}

func (v *Version) CompareWithWithContext(ctx context.Context, other *version.Version) (int, error) {
	clusterVersion, err := v.GetVersionWithContext(ctx)
	if err != nil {
		return 0, err
	}
	return clusterVersion.Compare(other), nil
}

func (v *Version) CompareWith(other *version.Version) (int, error) {
	return v.CompareWithWithContext(v.rest.ctx, other)
}

// ------------------------------------------------------

type Quota struct {
	*VastResource
}

// ------------------------------------------------------

type View struct {
	*VastResource
}

// ------------------------------------------------------

type VipPool struct {
	*VastResource
}

func (v *VipPool) IpRangeForWithContext(ctx context.Context, name string) ([]string, error) {
	result, err := v.GetWithContext(ctx, Params{"name": name})
	if err != nil {
		return nil, err
	}
	var ipRanges struct {
		IpRanges [][2]string `json:"ip_ranges"`
	}
	if err = result.Fill(&ipRanges); err != nil {
		return nil, err
	}
	return generateIPRange(ipRanges.IpRanges)
}

func (v *VipPool) IpRangeFor(name string) ([]string, error) {
	return v.IpRangeForWithContext(v.rest.ctx, name)
}

// ------------------------------------------------------

type User struct {
	*VastResource
}

// UsersCopyParams represents the parameters for copying users
type UsersCopyParams struct {
	DestinationProviderID int64   `json:"destination_provider_id"`
	TenantID              int64   `json:"tenant_id,omitempty"`
	UserIDs               []int64 `json:"user_ids,omitempty"`
}

func (u *User) CopyWithContext(ctx context.Context, params UsersCopyParams) error {
	// Validate parameters according to OpenAPI spec
	if params.TenantID == 0 && len(params.UserIDs) == 0 {
		return fmt.Errorf("either tenant_id or user_ids must be provided")
	}
	if params.TenantID != 0 && len(params.UserIDs) > 0 {
		return fmt.Errorf("cannot provide both tenant_id and user_ids")
	}

	requestParams := Params{
		"destination_provider_id": params.DestinationProviderID,
	}
	if params.TenantID != 0 {
		requestParams["tenant_id"] = params.TenantID
	}
	if len(params.UserIDs) > 0 {
		requestParams["user_ids"] = params.UserIDs
	}

	path := fmt.Sprintf("%s/copy", u.resourcePath)
	result, err := request[Record](ctx, u, http.MethodPost, path, u.apiVersion, nil, requestParams)
	if err != nil {
		return err
	}
	task := asyncResultFromRecord(ctx, result, u.rest)
	if _, err := task.Wait(3 * time.Minute); err != nil {
		return fmt.Errorf("failed to copy users: %w", err)
	}
	return nil
}

func (u *User) Copy(params UsersCopyParams) error {
	return u.CopyWithContext(u.rest.ctx, params)
}

func (u *User) UpdateTenantDataWithContext(ctx context.Context, userId any, params Params) (Record, error) {
	path := buildResourcePathWithID(u.resourcePath, userId, "tenant_data")

	result, err := request[Record](ctx, u, http.MethodPatch, path, u.apiVersion, nil, params)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (u *User) UpdateTenantData(userId any, params Params) (Record, error) {
	return u.UpdateTenantDataWithContext(u.rest.ctx, userId, params)
}

func (u *User) GetTenantDataWithContext(ctx context.Context, userId any) (Record, error) {
	path := buildResourcePathWithID(u.resourcePath, userId, "tenant_data")

	result, err := request[Record](ctx, u, http.MethodGet, path, u.apiVersion, nil, nil)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (u *User) GetTenantData(userId any) (Record, error) {
	return u.GetTenantDataWithContext(u.rest.ctx, userId)
}

// ------------------------------------------------------

type UserKey struct {
	*VastResource
}

func (uk *UserKey) CreateKeyWithContext(ctx context.Context, userId int64) (Record, error) {
	path := fmt.Sprintf(uk.resourcePath, userId)
	return request[Record](ctx, uk, http.MethodPost, path, uk.apiVersion, nil, nil)
}

func (uk *UserKey) CreateKey(userId int64) (Record, error) {
	return uk.CreateKeyWithContext(uk.rest.ctx, userId)
}

func (uk *UserKey) EnableKeyWithContext(ctx context.Context, userId int64, accessKey string) (EmptyRecord, error) {
	path := fmt.Sprintf(uk.resourcePath, userId)
	params := Params{"access_key": accessKey, "enabled": true}
	return request[EmptyRecord](ctx, uk, http.MethodPatch, path, uk.apiVersion, nil, params)
}

func (uk *UserKey) EnableKey(userId int64, accessKey string) (EmptyRecord, error) {
	return uk.EnableKeyWithContext(uk.rest.ctx, userId, accessKey)
}

func (uk *UserKey) DisableKeyWithContext(ctx context.Context, userId int64, accessKey string) (EmptyRecord, error) {
	path := fmt.Sprintf(uk.resourcePath, userId)
	params := Params{"access_key": accessKey, "enabled": false}
	return request[EmptyRecord](ctx, uk, http.MethodPatch, path, uk.apiVersion, nil, params)
}

func (uk *UserKey) DisableKey(userId int64, accessKey string) (EmptyRecord, error) {
	return uk.DisableKeyWithContext(uk.rest.ctx, userId, accessKey)
}

func (uk *UserKey) DeleteKeyWithContext(ctx context.Context, userId int64, accessKey string) (EmptyRecord, error) {
	path := fmt.Sprintf(uk.resourcePath, userId)
	return request[EmptyRecord](ctx, uk, http.MethodDelete, path, uk.apiVersion, nil, Params{"access_key": accessKey})
}

func (uk *UserKey) DeleteKey(userId int64, accessKey string) (EmptyRecord, error) {
	return uk.DeleteKeyWithContext(uk.rest.ctx, userId, accessKey)
}

// ------------------------------------------------------

type NonLocalUserKey struct {
	*VastResource
}

// ------------------------------------------------------

type Cnode struct {
	*VastResource
}

// ------------------------------------------------------

type QosPolicy struct {
	*VastResource
}

// ------------------------------------------------------

type Dns struct {
	*VastResource
}

// ------------------------------------------------------

type ViewPolicy struct {
	*VastResource
}

// ------------------------------------------------------

type Group struct {
	*VastResource
}

// ------------------------------------------------------

type Nis struct {
	*VastResource
}

// ------------------------------------------------------

type Tenant struct {
	*VastResource
}

func (t *Tenant) RevokeEncryptionGroupWithContext(ctx context.Context, tenantId any) (EmptyRecord, error) {
	path := buildResourcePathWithID(t.resourcePath, tenantId, "revoke_encryption_group")
	return request[EmptyRecord](ctx, t, http.MethodPost, path, t.apiVersion, nil, nil)
}

func (t *Tenant) RevokeEncryptionGroup(tenantId any) (EmptyRecord, error) {
	return t.RevokeEncryptionGroupWithContext(t.rest.ctx, tenantId)
}

func (t *Tenant) DeactivateEncryptionGroupWithContext(ctx context.Context, tenantId any) (EmptyRecord, error) {
	path := buildResourcePathWithID(t.resourcePath, tenantId, "deactivate_encryption_group")
	return request[EmptyRecord](ctx, t, http.MethodPost, path, t.apiVersion, nil, nil)
}

func (t *Tenant) DeactivateEncryptionGroup(tenantId any) (EmptyRecord, error) {
	return t.DeactivateEncryptionGroupWithContext(t.rest.ctx, tenantId)
}

func (t *Tenant) ReinstateEncryptionGroupWithContext(ctx context.Context, tenantId any) (EmptyRecord, error) {
	path := buildResourcePathWithID(t.resourcePath, tenantId, "reinstate_encryption_group")
	return request[EmptyRecord](ctx, t, http.MethodPost, path, t.apiVersion, nil, nil)
}

func (t *Tenant) ReinstateEncryptionGroup(tenantId any) (EmptyRecord, error) {
	return t.ReinstateEncryptionGroupWithContext(t.rest.ctx, tenantId)
}

func (t *Tenant) RotateEncryptionGroupKeyWithContext(ctx context.Context, tenantId any) (EmptyRecord, error) {
	path := buildResourcePathWithID(t.resourcePath, tenantId, "rotate_encryption_group_key")
	return request[EmptyRecord](ctx, t, http.MethodPost, path, t.apiVersion, nil, nil)
}

func (t *Tenant) RotateEncryptionGroupKey(tenantId any) (EmptyRecord, error) {
	return t.RotateEncryptionGroupKeyWithContext(t.rest.ctx, tenantId)
}

func (t *Tenant) GetConfiguredIdPWithContext(ctx context.Context, tenantName string) (Record, error) {
	path := fmt.Sprintf("%s/configured_idp", t.resourcePath)
	params := Params{"name": tenantName}
	return request[Record](ctx, t, http.MethodGet, path, t.apiVersion, params, nil)
}

func (t *Tenant) GetConfiguredIdP(tenantName string) (Record, error) {
	return t.GetConfiguredIdPWithContext(t.rest.ctx, tenantName)
}

func (t *Tenant) GetClientMetricsWithContext(ctx context.Context, tenantId any) (Record, error) {
	path := buildResourcePathWithID(t.resourcePath, tenantId, "client_metrics")
	return request[Record](ctx, t, http.MethodGet, path, t.apiVersion, nil, nil)
}

func (t *Tenant) GetClientMetrics(tenantId any) (Record, error) {
	return t.GetClientMetricsWithContext(t.rest.ctx, tenantId)
}

func (t *Tenant) UpdateClientMetricsWithContext(ctx context.Context, tenantId any, params Params) (Record, error) {
	path := buildResourcePathWithID(t.resourcePath, tenantId, "client_metrics")
	return request[Record](ctx, t, http.MethodPatch, path, t.apiVersion, nil, params)
}

func (t *Tenant) UpdateClientMetrics(tenantId any, params Params) (Record, error) {
	return t.UpdateClientMetricsWithContext(t.rest.ctx, tenantId, params)
}

// ------------------------------------------------------

type Ldap struct {
	*VastResource
}

// ------------------------------------------------------

type S3LifeCycleRule struct {
	*VastResource
}

// ------------------------------------------------------

type ActiveDirectory struct {
	*VastResource
}

// ------------------------------------------------------

type S3Policy struct {
	*VastResource
}

// ------------------------------------------------------

type ProtectedPath struct {
	*VastResource
}

// ------------------------------------------------------

type GlobalSnapshotStream struct {
	*VastResource
}

func (gss *GlobalSnapshotStream) CloneSnapshotWithContext(ctx context.Context, snapId int64, createParams Params) (Record, error) {
	path := fmt.Sprintf("snapshots/%d/clone/", snapId)
	return request[Record](ctx, gss, http.MethodPost, path, gss.apiVersion, nil, createParams)
}

func (gss *GlobalSnapshotStream) CloneSnapshot(snapId int64, createParams Params) (Record, error) {
	return gss.CloneSnapshotWithContext(gss.rest.ctx, snapId, createParams)
}

func (gss *GlobalSnapshotStream) EnsureCloneSnapshotWithContext(ctx context.Context, name string, snapId int64, createParams Params) (Record, error) {
	params := Params{"name": name}
	response, err := gss.GetWithContext(ctx, params)
	if err != nil {
		if IsNotFoundErr(err) {
			createParams["name"] = name
			return gss.CloneSnapshotWithContext(ctx, snapId, createParams)
		}
		return nil, err
	}
	return response, nil
}

func (gss *GlobalSnapshotStream) EnsureCloneSnapshot(name string, snapId int64, createParams Params) (Record, error) {
	return gss.EnsureCloneSnapshotWithContext(gss.rest.ctx, name, snapId, createParams)
}

func (gss *GlobalSnapshotStream) StopCloneSnapshotWithContext(ctx context.Context, gssId int64) (Awaitable, error) {
	path := fmt.Sprintf("%s/%d/stop", gss.resourcePath, gssId)
	record, err := request[Record](ctx, gss, http.MethodPatch, path, gss.apiVersion, nil, nil)
	if err != nil {
		return nil, err
	}
	return asyncResultFromRecord(ctx, record, gss.rest), nil
}

func (gss *GlobalSnapshotStream) StopCloneSnapshot(gssId int64) (Awaitable, error) {
	return gss.StopCloneSnapshotWithContext(gss.rest.ctx, gssId)
}

func (gss *GlobalSnapshotStream) EnsureCloneSnapshotDeletedWithContext(ctx context.Context, searchParams Params) (Renderable, error) {
	response, err := gss.GetWithContext(ctx, searchParams)
	if response != nil {
		type GssContainer struct {
			Id    int64  `json:"id"`
			State string `json:"state,omitempty"`
		}
		gssContainer := GssContainer{}
		if err = response.Fill(&gssContainer); err != nil {
			return nil, err
		}
		if gssContainer.State != "finished" {
			task, err := gss.StopCloneSnapshotWithContext(ctx, gssContainer.Id)
			if err != nil {
				return nil, err
			}
			if _, err = task.Wait(3 * time.Minute); err != nil {
				return nil, err
			}
		}
		if deleteResult, err := gss.DeleteByIdWithContext(ctx, response.RecordID(), nil, Params{"remove_dir": true}); IsApiError(err) {
			if err.(*ApiError).StatusCode == 404 {
				return EmptyRecord{}, nil
			}
		} else {
			return deleteResult, err
		}
	}
	return response, err
}

func (gss *GlobalSnapshotStream) EnsureCloneSnapshotDeleted(searchParams Params) (Renderable, error) {
	return gss.EnsureCloneSnapshotDeletedWithContext(gss.rest.ctx, searchParams)
}

// ------------------------------------------------------

type ReplicationPeers struct {
	*VastResource
}

// ------------------------------------------------------

type ProtectionPolicy struct {
	*VastResource
}

// ------------------------------------------------------

type S3replicationPeers struct {
	*VastResource
}

// ------------------------------------------------------

type Realm struct {
	*VastResource
}

// ------------------------------------------------------

type Role struct {
	*VastResource
}

// ------------------------------------------------------

type Snapshot struct {
	*VastResource
}

// ------------------------------------------------------

type BlockHost struct {
	*VastResource
}

func (bh *BlockHost) EnsureBlockHostWithContext(ctx context.Context, name string, tenantId int, nqn string) (Record, error) {
	params := Params{"name": name, "tenant_id": tenantId}
	blockHost, err := bh.GetWithContext(ctx, params)
	if IsNotFoundErr(err) {
		params.Update(Params{"nqn": nqn, "os_type": "LINUX", "connectivity_type": "tcp"}, false)
		return bh.CreateWithContext(ctx, params)
	} else if err != nil {
		return nil, err
	}
	return blockHost, nil
}

func (bh *BlockHost) EnsureBlockHost(name string, tenantId int, nqn string) (Record, error) {
	return bh.EnsureBlockHostWithContext(bh.rest.ctx, name, tenantId, nqn)
}

// ------------------------------------------------------

type Volume struct {
	*VastResource
}

func (v *Volume) CloneVolumeWithContext(ctx context.Context, snapId, targetSubsystemId int64, targetVolumePath string) (Record, error) {
	body := Params{"target_subsystem_id": targetSubsystemId, "target_volume_path": targetVolumePath}
	path := fmt.Sprintf("snapshots/%d/clone_volume/", snapId)
	return request[Record](ctx, v, http.MethodPost, path, v.apiVersion, nil, body)
}

func (v *Volume) CloneVolume(snapId, targetSubsystemId int64, targetVolumePath string) (Record, error) {
	return v.CloneVolumeWithContext(v.rest.ctx, snapId, targetSubsystemId, targetVolumePath)
}

// ------------------------------------------------------

type VTask struct {
	*VastResource
}

// nextBackoff returns the next polling interval using additive backoff strategy.
//
// It increases the current interval by 250ms up to a given max value.
//
// Parameters:
//   - current: the current polling interval.
//   - max: the maximum allowed interval.
//
// Returns:
//   - time.Duration: the next interval to wait before polling again.
func nextBackoff(current, max time.Duration) time.Duration {
	next := current + 250*time.Millisecond
	if next > max {
		return max
	}
	return next
}

// WaitTaskWithContext polls the task status until it completes, fails, or the context expires.
//
// It starts with a 500ms polling interval and increases it slightly after each attempt,
// using exponential-style backoff (capped at 5 seconds). This reduces the load on the API
// during long-running tasks.
//
// Task states:
//   - "completed" → returns the task Record.
//   - "running"   → continues polling.
//   - any other state → considered failure, and returns the last message from the task.
//
// If the context deadline is exceeded or canceled, the method returns an error with context cause.
//
// Parameters:
//   - ctx: context with optional timeout or cancellation.
//   - taskId: unique identifier of the task to wait for.
//
// Returns:
//   - Record: the completed task record, if successful.
//   - error: if the task failed, context expired, or an API error occurred.
func (t *VTask) WaitTaskWithContext(ctx context.Context, taskId int64) (Record, error) {
	if t == nil {
		return nil, fmt.Errorf("VTask is nil")
	}

	baseInterval := 500 * time.Millisecond
	maxInterval := 5 * time.Second
	currentInterval := baseInterval

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for task %d: %w", taskId, ctx.Err())

		default:
			task, err := t.GetByIdWithContext(ctx, taskId)
			if err != nil {
				return nil, err
			}

			state := strings.ToLower(fmt.Sprintf("%v", task["state"]))
			switch state {
			case "completed":
				return task, nil
			case "running":
				// backoff
				time.Sleep(currentInterval)
				currentInterval = nextBackoff(currentInterval, maxInterval)
			default:
				rawMessages := task["messages"]
				messages, ok := rawMessages.([]interface{})
				if !ok || len(messages) == 0 {
					return nil, fmt.Errorf("task %s failed with ID %d: no messages or unexpected format", task.RecordName(), task.RecordID())
				}
				lastMsg := fmt.Sprintf("%v", messages[len(messages)-1])
				return nil, fmt.Errorf("task %s failed with ID %d: %s", task.RecordName(), task.RecordID(), lastMsg)
			}
		}
	}
}

func (t *VTask) WaitTask(taskId int64, timeout time.Duration) (Record, error) {
	ctx, cancel := context.WithTimeout(t.rest.ctx, timeout)
	defer cancel()
	return t.WaitTaskWithContext(ctx, taskId)
}

// ------------------------------------------------------

type BlockHostMapping struct {
	*VastResource
}

func (bhm *BlockHostMapping) MapWithContext(ctx context.Context, hostId, volumeId int64) (Record, error) {
	body := Params{
		"pairs_to_add": []Params{
			{
				"host_id":   hostId,
				"volume_id": volumeId,
			},
		},
	}
	return bhm.bulkPatchAndWait(ctx, body)
}

func (bhm *BlockHostMapping) Map(hostId, volumeId int64) (Record, error) {
	return bhm.MapWithContext(bhm.rest.ctx, hostId, volumeId)
}

func (bhm *BlockHostMapping) UnMapWithContext(ctx context.Context, hostId, volumeId int64) (Record, error) {
	body := Params{
		"pairs_to_remove": []Params{
			{
				"host_id":   hostId,
				"volume_id": volumeId,
			},
		},
	}
	return bhm.bulkPatchAndWait(ctx, body)
}

func (bhm *BlockHostMapping) UnMap(hostId, volumeId int64) (Record, error) {
	return bhm.UnMapWithContext(bhm.rest.ctx, hostId, volumeId)
}

func (bhm *BlockHostMapping) EnsureMapWithContext(ctx context.Context, hostId, volumeId int64) (Record, error) {
	result, err := bhm.GetWithContext(ctx, Params{"volume__id": volumeId, "block_host__id": hostId})
	if IsNotFoundErr(err) {
		return bhm.MapWithContext(ctx, hostId, volumeId)
	}
	return result, err
}

func (bhm *BlockHostMapping) EnsureMap(hostId, volumeId int64) (Record, error) {
	return bhm.EnsureMapWithContext(bhm.rest.ctx, hostId, volumeId)
}

func (bhm *BlockHostMapping) EnsureUnmapWithContext(ctx context.Context, hostId, volumeId int64) (Record, error) {
	result, err := bhm.GetWithContext(ctx, Params{"volume__id": volumeId, "block_host__id": hostId})
	if result != nil {
		return bhm.UnMapWithContext(ctx, hostId, volumeId)
	}
	return result, err
}

func (bhm *BlockHostMapping) EnsureUnmap(hostId, volumeId int64) (Record, error) {
	return bhm.EnsureUnmapWithContext(bhm.rest.ctx, hostId, volumeId)
}

func (bhm *BlockHostMapping) bulkPatchAndWait(ctx context.Context, body Params) (Record, error) {
	path := fmt.Sprintf("%s/bulk", bhm.resourcePath)
	record, err := request[Record](ctx, bhm, http.MethodPatch, path, bhm.apiVersion, nil, body)
	if err != nil {
		return nil, err
	}
	task := asyncResultFromRecord(ctx, record, bhm.rest)
	return task.Wait(1 * time.Minute)
}

// ------------------------------------------------------

type NonLocalUser struct {
	*VastResource
}

func (u *NonLocalUser) UpdateNonLocalUserWithContext(ctx context.Context, data Params) (Record, error) {
	// This function is used to update a non-local user with the given data.
	// Note: non-local user has no ID so we cannot use standard UpdateWithContext.
	return request[Record](ctx, u, http.MethodPatch, u.resourcePath, u.apiVersion, nil, data)
}

func (u *NonLocalUser) UpdateNonLocalUser(data Params) (Record, error) {
	return u.UpdateNonLocalUserWithContext(u.rest.ctx, data)
}

// ------------------------------------------------------

type NonLocalGroup struct {
	*VastResource
}

func (g *NonLocalGroup) UpdateNonLocalGroupWithContext(ctx context.Context, data Params) (Record, error) {
	// This function is used to update a non-local group with the given data.
	// Note: non-local group has no ID so we cannot use standard UpdateWithContext.
	return request[Record](ctx, g, http.MethodPatch, g.resourcePath, g.apiVersion, nil, data)
}

func (g *NonLocalGroup) UpdateNonLocalGroup(data Params) (Record, error) {
	return g.UpdateNonLocalGroupWithContext(g.rest.ctx, data)
}

// ------------------------------------------------------

type ApiToken struct {
	*VastResource
}

func (a *ApiToken) RevokeWithContext(ctx context.Context, tokenId string) (EmptyRecord, error) {
	path := fmt.Sprintf("%s/%s/revoke", a.resourcePath, tokenId)
	return request[EmptyRecord](ctx, a, http.MethodPatch, path, a.apiVersion, nil, nil)
}

func (a *ApiToken) Revoke(tokenId string) (EmptyRecord, error) {
	return a.RevokeWithContext(a.rest.ctx, tokenId)
}

// ------------------------------------------------------

type KafkaBroker struct {
	*VastResource
}

// ------------------------------------------------------

type Manager struct {
	*VastResource
}

// ------------------------------------------------------

type Folder struct {
	*VastResource
}

func (f *Folder) CreateFolderWithContext(ctx context.Context, data Params) (Record, error) {
	path := fmt.Sprintf("%s/create_folder", f.resourcePath)
	return request[Record](ctx, f, http.MethodPost, path, f.apiVersion, nil, data)
}

func (f *Folder) CreateFolder(data Params) (Record, error) {
	return f.CreateFolderWithContext(f.rest.ctx, data)
}

func (f *Folder) ModifyFolderWithContext(ctx context.Context, data Params) (Record, error) {
	path := fmt.Sprintf("%s/modify_folder", f.resourcePath)
	return request[Record](ctx, f, http.MethodPatch, path, f.apiVersion, nil, data)
}

func (f *Folder) ModifyFolder(data Params) (Record, error) {
	return f.ModifyFolderWithContext(f.rest.ctx, data)
}

func (f *Folder) DeleteFolderWithContext(ctx context.Context, data Params) (EmptyRecord, error) {
	path := fmt.Sprintf("%s/delete_folder", f.resourcePath)
	return request[EmptyRecord](ctx, f, http.MethodDelete, path, f.apiVersion, nil, data)
}

func (f *Folder) DeleteFolder(data Params) (EmptyRecord, error) {
	return f.DeleteFolderWithContext(f.rest.ctx, data)
}

func (f *Folder) StatPathWithContext(ctx context.Context, data Params) (Record, error) {
	path := fmt.Sprintf("%s/stat_path", f.resourcePath)
	return request[Record](ctx, f, http.MethodPost, path, f.apiVersion, nil, data)
}

func (f *Folder) StatPath(data Params) (Record, error) {
	return f.StatPathWithContext(f.rest.ctx, data)
}

func (f *Folder) SetReadOnlyWithContext(ctx context.Context, data Params) (Record, error) {
	path := fmt.Sprintf("%s/read_only", f.resourcePath)
	return request[Record](ctx, f, http.MethodPost, path, f.apiVersion, nil, data)
}

func (f *Folder) SetReadOnly(data Params) (Record, error) {
	return f.SetReadOnlyWithContext(f.rest.ctx, data)
}

func (f *Folder) GetReadOnlyWithContext(ctx context.Context, params Params) (Record, error) {
	path := fmt.Sprintf("%s/read_only", f.resourcePath)
	return request[Record](ctx, f, http.MethodGet, path, f.apiVersion, params, nil)
}

func (f *Folder) GetReadOnly(params Params) (Record, error) {
	return f.GetReadOnlyWithContext(f.rest.ctx, params)
}

func (f *Folder) DeleteReadOnlyWithContext(ctx context.Context, data Params) (EmptyRecord, error) {
	path := fmt.Sprintf("%s/read_only", f.resourcePath)
	return request[EmptyRecord](ctx, f, http.MethodDelete, path, f.apiVersion, nil, data)
}

func (f *Folder) DeleteReadOnly(data Params) (EmptyRecord, error) {
	return f.DeleteReadOnlyWithContext(f.rest.ctx, data)
}

// ------------------------------------------------------

type EventDefinition struct {
	*VastResource
}

// ------------------------------------------------------

type EventDefinitionConfig struct {
	*VastResource
}

// ------------------------------------------------------

type BGPConfig struct {
	*VastResource
}

// ------------------------------------------------------

type Vms struct {
	*VastResource
}

func (v *Vms) SetMaxApiTokensPerUserWithContext(ctx context.Context, vmsId int64, tokensCount int64) (EmptyRecord, error) {
	path := fmt.Sprintf("%s/%d/set_max_api_tokens_per_user", v.resourcePath, vmsId)
	body := Params{"max_api_tokens_per_user": tokensCount}
	return request[EmptyRecord](ctx, v, http.MethodPatch, path, v.apiVersion, nil, body)
}

func (v *Vms) SetMaxApiTokensPerUser(vmsId int64, tokensCount int64) (EmptyRecord, error) {
	return v.SetMaxApiTokensPerUserWithContext(v.rest.ctx, vmsId, tokensCount)
}

func (v *Vms) GetConfiguredIdPsWithContext(ctx context.Context, vmsId any) ([]string, error) {
	var result = make([]string, 0)
	path := buildResourcePathWithID(v.resourcePath, vmsId, "configured_idps")
	recordSet, err := request[RecordSet](ctx, v, http.MethodGet, path, v.apiVersion, nil, nil)
	if err != nil {
		return nil, err
	}
	for _, record := range recordSet {
		result = append(result, fmt.Sprintf("%v", record["@raw"]))
	}
	return result, nil
}

func (v *Vms) GetConfiguredIdPs(vmsId any) ([]string, error) {
	return v.GetConfiguredIdPsWithContext(v.rest.ctx, vmsId)
}

// ------------------------------------------------------

type Topic struct {
	*VastResource
}

func (t *Topic) ShowTopicWithContext(ctx context.Context, params Params) (Record, error) {
	path := fmt.Sprintf("%s/show", t.resourcePath)
	return request[Record](ctx, t, http.MethodGet, path, t.apiVersion, params, nil)
}

func (t *Topic) ShowTopic(params Params) (Record, error) {
	return t.ShowTopicWithContext(t.rest.ctx, params)
}

func (t *Topic) DeleteTopicWithContext(ctx context.Context, params Params) (EmptyRecord, error) {
	path := fmt.Sprintf("%s/delete", t.resourcePath)
	return request[EmptyRecord](ctx, t, http.MethodDelete, path, t.apiVersion, nil, params)
}

func (t *Topic) DeleteTopic(params Params) (EmptyRecord, error) {
	return t.DeleteTopicWithContext(t.rest.ctx, params)
}

// ------------------------------------------------------

type LocalProvider struct {
	*VastResource
}

// ------------------------------------------------------

type LocalS3Key struct {
	*VastResource
}

// ------------------------------------------------------

type EncryptionGroup struct {
	*VastResource
}

func (eg *EncryptionGroup) RevokeEncryptionGroupWithContext(ctx context.Context, encryptionGroupId any) (EmptyRecord, error) {
	path := buildResourcePathWithID(eg.resourcePath, encryptionGroupId, "revoke_encryption_group")
	return request[EmptyRecord](ctx, eg, http.MethodPost, path, eg.apiVersion, nil, nil)
}

func (eg *EncryptionGroup) RevokeEncryptionGroup(encryptionGroupId any) (EmptyRecord, error) {
	return eg.RevokeEncryptionGroupWithContext(eg.rest.ctx, encryptionGroupId)
}

func (eg *EncryptionGroup) DeactivateEncryptionGroupWithContext(ctx context.Context, encryptionGroupId any) (EmptyRecord, error) {
	path := buildResourcePathWithID(eg.resourcePath, encryptionGroupId, "deactivate_encryption_group")
	return request[EmptyRecord](ctx, eg, http.MethodPost, path, eg.apiVersion, nil, nil)
}

func (eg *EncryptionGroup) DeactivateEncryptionGroup(encryptionGroupId any) (EmptyRecord, error) {
	return eg.DeactivateEncryptionGroupWithContext(eg.rest.ctx, encryptionGroupId)
}

func (eg *EncryptionGroup) ReinstateEncryptionGroupWithContext(ctx context.Context, encryptionGroupId any) (EmptyRecord, error) {
	path := buildResourcePathWithID(eg.resourcePath, encryptionGroupId, "reinstate_encryption_group")
	return request[EmptyRecord](ctx, eg, http.MethodPost, path, eg.apiVersion, nil, nil)
}

func (eg *EncryptionGroup) ReinstateEncryptionGroup(encryptionGroupId any) (EmptyRecord, error) {
	return eg.ReinstateEncryptionGroupWithContext(eg.rest.ctx, encryptionGroupId)
}

func (eg *EncryptionGroup) RotateEncryptionGroupKeyWithContext(ctx context.Context, encryptionGroupId any) (EmptyRecord, error) {
	path := buildResourcePathWithID(eg.resourcePath, encryptionGroupId, "rotate_encryption_group_key")
	return request[EmptyRecord](ctx, eg, http.MethodPost, path, eg.apiVersion, nil, nil)
}

func (eg *EncryptionGroup) RotateEncryptionGroupKey(encryptionGroupId any) (EmptyRecord, error) {
	return eg.RotateEncryptionGroupKeyWithContext(eg.rest.ctx, encryptionGroupId)
}

// ------------------------------------------------------

type SamlConfig struct {
	*VastResource
}

func (sc *SamlConfig) GetConfigWithContext(ctx context.Context, vmsId any, idpName string) (Record, error) {
	path := fmt.Sprintf(sc.resourcePath, vmsId)
	params := Params{"idp_name": idpName}
	return request[Record](ctx, sc, http.MethodGet, path, sc.apiVersion, params, nil)
}

func (sc *SamlConfig) GetConfig(vmsId any, idpName string) (Record, error) {
	return sc.GetConfigWithContext(sc.rest.ctx, vmsId, idpName)
}

func (sc *SamlConfig) RemoveSignedCertWithContext(ctx context.Context, vmsId any, idpName string) (EmptyRecord, error) {
	path := fmt.Sprintf(sc.resourcePath, vmsId)
	params := Params{"idp_name": idpName}
	return request[EmptyRecord](ctx, sc, http.MethodPatch, path, sc.apiVersion, params, nil)
}

func (sc *SamlConfig) RemoveSignedCert(vmsId any, idpName string) (EmptyRecord, error) {
	return sc.RemoveSignedCertWithContext(sc.rest.ctx, vmsId, idpName)
}

func (sc *SamlConfig) UpdateConfigWithContext(ctx context.Context, vmsId any, idpName string, params Params) (Record, error) {
	path := fmt.Sprintf(sc.resourcePath, vmsId)
	queryParams := Params{"idp_name": idpName}
	return request[Record](ctx, sc, http.MethodPost, path, sc.apiVersion, queryParams, params)
}

func (sc *SamlConfig) UpdateConfig(vmsId any, idpName string, params Params) (Record, error) {
	return sc.UpdateConfigWithContext(sc.rest.ctx, vmsId, idpName, params)
}

func (sc *SamlConfig) DeleteConfigWithContext(ctx context.Context, vmsId any, idpName string) (EmptyRecord, error) {
	path := fmt.Sprintf(sc.resourcePath, vmsId)
	params := Params{"idp_name": idpName}
	return request[EmptyRecord](ctx, sc, http.MethodDelete, path, sc.apiVersion, params, nil)
}

func (sc *SamlConfig) DeleteConfig(vmsId any, idpName string) (EmptyRecord, error) {
	return sc.DeleteConfigWithContext(sc.rest.ctx, vmsId, idpName)
}

// ------------------------------------------------------

type Kerberos struct {
	*VastResource
}

// GenerateKeytabWithContext generates a keytab for a specific Kerberos configuration
// Requires admin_username and admin_password in params
func (k *Kerberos) GenerateKeytabWithContext(ctx context.Context, kerberosId any, params Params) (Record, error) {
	path := buildResourcePathWithID(k.resourcePath, kerberosId, "keytab")
	return request[Record](ctx, k, http.MethodPost, path, k.apiVersion, nil, params)
}

func (k *Kerberos) GenerateKeytab(kerberosId any, params Params) (Record, error) {
	return k.GenerateKeytabWithContext(k.rest.ctx, kerberosId, params)
}

// UploadKeytabWithContext uploads a keytab file for a specific Kerberos configuration
// This method uses multipart/form-data for file upload
func (k *Kerberos) UploadKeytabWithContext(ctx context.Context, kerberosId any, keytabFile []byte, filename string) (Record, error) {
	path := buildResourcePathWithID(k.resourcePath, kerberosId, "keytab")

	// Prepare multipart form data using Params
	params := Params{
		"keytab_file": FileData{
			Filename: filename,
			Content:  keytabFile,
		},
	}

	// Create headers to indicate multipart/form-data
	headers := []http.Header{{
		HeaderContentType: []string{ContentTypeMultipartForm},
	}}

	return requestWithHeaders[Record](ctx, k, http.MethodPut, path, k.apiVersion, nil, params, headers)
}

func (k *Kerberos) UploadKeytab(kerberosId any, keytabFile []byte, filename string) (Record, error) {
	return k.UploadKeytabWithContext(k.rest.ctx, kerberosId, keytabFile, filename)
}

// ------------------------------------------------------

type Cluster struct {
	*VastResource
}

func (c *Cluster) AddEkmWithContext(ctx context.Context, clusterId any, createParams Params) (EmptyRecord, error) {
	path := buildResourcePathWithID(c.resourcePath, clusterId, "add_ekm")
	return request[EmptyRecord](ctx, c, http.MethodPost, path, c.apiVersion, nil, createParams)
}

func (c *Cluster) AddEkm(clusterId any, createParams Params) (EmptyRecord, error) {
	return c.AddEkmWithContext(c.rest.ctx, clusterId, createParams)
}

// ------------------------------------------------------
