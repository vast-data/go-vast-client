package vast_client

import (
	"context"
	"encoding/json"
	"fmt"
	version "github.com/hashicorp/go-version"
	"net/http"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
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
		ApiToken |
		KafkaBroker
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

var sysVersion *version.Version

func (v *Version) GetVersionWithContext(ctx context.Context) (*version.Version, error) {
	if sysVersion != nil {
		return sysVersion, nil
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
	sysVersion = clusterVersion.Core()
	return sysVersion, nil
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

func (gss *GlobalSnapshotStream) EnsureGssWithContext(ctx context.Context, name, destPath string, snapId, tenantId int64, enabled bool) (Renderable, error) {
	params := Params{"name": name}
	response, err := gss.GetWithContext(ctx, params)
	if err != nil {
		if IsNotFoundErr(err) {
			createParams := Params{
				"loanee_root_path": destPath,
				"name":             name,
				"enabled":          enabled,
				"loanee_tenant_id": tenantId,
			}
			path := fmt.Sprintf("snapshots/%d/clone/", snapId)
			return request[Record](ctx, gss, http.MethodPost, path, gss.apiVersion, nil, createParams)
		}
		return nil, err
	}
	return response, nil
}

func (gss *GlobalSnapshotStream) EnsureGss(name, destPath string, snapId, tenantId int64, enabled bool) (Renderable, error) {
	return gss.EnsureGssWithContext(gss.rest.ctx, name, destPath, snapId, tenantId, enabled)
}

func (gss *GlobalSnapshotStream) StopGssWithContext(ctx context.Context, gssId int64) (Awaitable, error) {
	path := fmt.Sprintf("%s/%d/stop", gss.resourcePath, gssId)
	record, err := request[Record](ctx, gss, http.MethodPatch, path, gss.apiVersion, nil, nil)
	if err != nil {
		return nil, err
	}
	return asyncResultFromRecord(ctx, record), nil
}

func (gss *GlobalSnapshotStream) StopGss(gssId int64) (Awaitable, error) {
	return gss.StopGssWithContext(gss.rest.ctx, gssId)
}

func (gss *GlobalSnapshotStream) EnsureGssDeletedWithContext(ctx context.Context, searchParams Params) (Renderable, error) {
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
			task, err := gss.StopGssWithContext(ctx, gssContainer.Id)
			if err != nil {
				return nil, err
			}
			if _, err = task.Wait(); err != nil {
				return nil, err
			}
		}
		if deleteResult, err := gss.DeleteByIdWithContext(ctx, response.RecordID(), Params{"remove_dir": true}); IsApiError(err) {
			if err.(*ApiError).StatusCode == 404 {
				return EmptyRecord{}, nil
			}
		} else {
			return deleteResult, err
		}
	}
	return response, err
}

func (gss *GlobalSnapshotStream) EnsureGssDeleted(searchParams Params) (Renderable, error) {
	return gss.EnsureGssDeletedWithContext(gss.rest.ctx, searchParams)
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

func (s *Snapshot) afterRequest(ctx context.Context, response Renderable) (Renderable, error) {
	// List of snapshots is returned under "results" key
	return applyCallbackForRecordUnion[Record](response, func(r Renderable) (Renderable, error) {
		// This callback is only invoked if response is a RecordSet
		if rawMap, ok := any(r).(map[string]interface{}); ok {
			if inner, found := rawMap["results"]; found {
				if list, ok := inner.([]map[string]any); ok {
					recordSet, err := toRecordSet(list)
					if err != nil {
						return nil, err
					}
					// Re set Resource key
					if err = setResourceKey(recordSet, s.GetResourceType()); err != nil {
						return nil, err
					}
					return recordSet, nil
				}
			}
		}
		return r, nil
	})
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

func (v *Volume) CloneVolumeWithContext(ctx context.Context, snapId, targetSubsystemId int64, targetVolumePath string) (Renderable, error) {
	body := Params{"target_subsystem_id": targetSubsystemId, "target_volume_path": targetVolumePath}
	path := fmt.Sprintf("snapshots/%d/clone_volume/", snapId)
	return request[Record](ctx, v, http.MethodPost, path, v.apiVersion, nil, body)
}

func (v *Volume) CloneVolume(snapId, targetSubsystemId int64, targetVolumePath string) (Renderable, error) {
	return v.CloneVolumeWithContext(v.rest.ctx, snapId, targetSubsystemId, targetVolumePath)
}

// ------------------------------------------------------

type VTask struct {
	*VastResource
}

// WaitTask waits for the task to complete
func (t *VTask) WaitTaskWithContext(ctx context.Context, taskId int64) (Record, error) {
	// isTaskComplete checks if the task is complete
	isTaskComplete := func(taskId int64) (Record, error) {
		task, err := t.GetByIdWithContext(ctx, taskId)
		if err != nil {
			return nil, err
		}
		// Check the task state
		taskName := task.RecordName()
		taskState := strings.ToLower(fmt.Sprintf("%v", task["state"]))
		_taskId := task.RecordID()
		if err != nil {
			return nil, err
		}
		switch taskState {
		case "completed":
			return task, nil
		case "running":
			return nil, fmt.Errorf("task %s with ID %d is still running, timeout occurred", taskName, _taskId)
		default:
			rawMessages := task["messages"]
			messages, ok := rawMessages.([]interface{})
			if !ok {
				return nil, fmt.Errorf("unexpected message format: %T", rawMessages)
			}
			if len(messages) == 0 {
				return nil, fmt.Errorf("task %s failed with ID %d: no messages found", taskName, _taskId)
			}
			lastMsg := fmt.Sprintf("%v", messages[len(messages)-1])
			return nil, fmt.Errorf("task %s failed with ID %d: %s", taskName, _taskId, lastMsg)
		}
	}
	// Retry logic to poll the task status
	retries := 30
	interval := time.Millisecond * 500
	backoffRate := 1

	for retries > 0 {
		task, err := isTaskComplete(taskId)
		if err == nil {
			return task, nil
		}
		time.Sleep(interval)
		// Backoff logic
		interval *= time.Duration(backoffRate)
		retries--
	}
	return nil, fmt.Errorf("task did not complete in time")
}

func (t *VTask) WaitTask(taskId int64) (Record, error) {
	return t.WaitTaskWithContext(t.rest.ctx, taskId)
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
	path := fmt.Sprintf("%s/bulk", bhm.resourcePath)
	// Make request on behalf of VTask (for proper parsing)
	record, err := request[Record](ctx, bhm, http.MethodPatch, path, bhm.apiVersion, nil, body)
	if err != nil {
		return nil, err
	}
	task := asyncResultFromRecord(ctx, record)
	return task.Wait()
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
	path := fmt.Sprintf("%s/bulk", bhm.resourcePath)
	record, err := request[Record](ctx, bhm, http.MethodPatch, path, bhm.apiVersion, nil, body)
	if err != nil {
		return nil, err
	}
	task := asyncResultFromRecord(ctx, record)
	return task.Wait()
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

// ------------------------------------------------------

type NonLocalUser struct {
	*VastResource
}

// ------------------------------------------------------

type NonLocalGroup struct {
	*VastResource
}

// ------------------------------------------------------

type ApiToken struct {
	*VastResource
}

// ------------------------------------------------------

type KafkaBroker struct {
	*VastResource
}

// ------------------------------------------------------
