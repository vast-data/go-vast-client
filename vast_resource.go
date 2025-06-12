package vast_client

import (
	"context"
	"fmt"
	version "github.com/hashicorp/go-version"
	"net/http"
	"strings"
	"time"
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
	*VastResourceEntry
}

// ------------------------------------------------------

type Version struct {
	*VastResourceEntry
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
	*VastResourceEntry
}

// ------------------------------------------------------

type View struct {
	*VastResourceEntry
}

// ------------------------------------------------------

type VipPool struct {
	*VastResourceEntry
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
	*VastResourceEntry
}

// ------------------------------------------------------

type UserKey struct {
	*VastResourceEntry
}

func (uk *UserKey) CreateKeyWithContext(ctx context.Context, userId int64) (Record, error) {
	path := fmt.Sprintf(uk.resourcePath, userId)
	return request[Record](ctx, uk, http.MethodPost, path, uk.apiVersion, nil, nil)
}

func (uk *UserKey) CreateKey(userId int64) (Record, error) {
	return uk.CreateKeyWithContext(uk.rest.ctx, userId)
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
	*VastResourceEntry
}

// ------------------------------------------------------

type QosPolicy struct {
	*VastResourceEntry
}

// ------------------------------------------------------

type Dns struct {
	*VastResourceEntry
}

// ------------------------------------------------------

type ViewPolicy struct {
	*VastResourceEntry
}

// ------------------------------------------------------

type Group struct {
	*VastResourceEntry
}

// ------------------------------------------------------

type Nis struct {
	*VastResourceEntry
}

// ------------------------------------------------------

type Tenant struct {
	*VastResourceEntry
}

// ------------------------------------------------------

type Ldap struct {
	*VastResourceEntry
}

// ------------------------------------------------------

type S3LifeCycleRule struct {
	*VastResourceEntry
}

// ------------------------------------------------------

type ActiveDirectory struct {
	*VastResourceEntry
}

// ------------------------------------------------------

type S3Policy struct {
	*VastResourceEntry
}

// ------------------------------------------------------

type ProtectedPath struct {
	*VastResourceEntry
}

// ------------------------------------------------------

type GlobalSnapshotStream struct {
	*VastResourceEntry
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
	*VastResourceEntry
}

// ------------------------------------------------------

type ProtectionPolicy struct {
	*VastResourceEntry
}

// ------------------------------------------------------

type S3replicationPeers struct {
	*VastResourceEntry
}

// ------------------------------------------------------

type Realm struct {
	*VastResourceEntry
}

// ------------------------------------------------------

type Role struct {
	*VastResourceEntry
}

// ------------------------------------------------------

type Snapshot struct {
	*VastResourceEntry
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
	*VastResourceEntry
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
	*VastResourceEntry
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
	*VastResourceEntry
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
	*VastResourceEntry
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
	*VastResourceEntry
}

// ------------------------------------------------------

type NonLocalGroup struct {
	*VastResourceEntry
}

// ------------------------------------------------------

type ApiToken struct {
	*VastResourceEntry
}

// ------------------------------------------------------

type KafkaBroker struct {
	*VastResourceEntry
}

// ------------------------------------------------------
