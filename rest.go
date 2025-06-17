package vast_client

import (
	"context"
	"fmt"
	version "github.com/hashicorp/go-version"
	"reflect"
	"time"
)

const dummyClusterVersion = "0.0.0"

// Dummy resource is used to support Request interceptors for "low level" session methods like GET, POST etc.
var dummyResource *Dummy

type VMSRest struct {
	ctx         context.Context
	Session     RESTSession
	resourceMap map[string]VastResourceAPI // Map to store resources by resourceType

	dummy                 *Dummy
	Versions              *Version
	VTasks                *VTask
	Quotas                *Quota
	Views                 *View
	VipPools              *VipPool
	Users                 *User
	UserKeys              *UserKey
	Snapshots             *Snapshot
	BlockHosts            *BlockHost
	Volumes               *Volume
	BlockHostMappings     *BlockHostMapping
	Cnodes                *Cnode
	QosPolicies           *QosPolicy
	Dns                   *Dns
	ViewPolies            *ViewPolicy
	Groups                *Group
	Nis                   *Nis
	Tenants               *Tenant
	Ldaps                 *Ldap
	S3LifeCycleRules      *S3LifeCycleRule
	ActiveDirectories     *ActiveDirectory
	S3Policies            *S3Policy
	ProtectedPaths        *ProtectedPath
	GlobalSnapshotStreams *GlobalSnapshotStream
	ReplicationPeers      *ReplicationPeers
	ProtectionPolicies    *ProtectionPolicy
	S3replicationPeers    *S3replicationPeers
	Realms                *Realm
	Roles                 *Role
	NonLocalUsers         *NonLocalUser
	NonLocalGroups        *NonLocalGroup
	ApiTokens             *ApiToken
	KafkaBrokers          *KafkaBroker
}

func NewVMSRest(config *VMSConfig) (*VMSRest, error) {
	config.Validate(
		withAuth,
		withHost,
		withUserAgent,
		witApiVersion("v5"),
		withTimeout(time.Second*30),
		withMaxConnections(10),
		withPort(443),
	)
	session, err := NewVMSSession(config)
	if err != nil {
		return nil, err
	}
	rest := &VMSRest{
		Session:     session,
		resourceMap: make(map[string]VastResourceAPI),
	}
	rest.dummy = newResource[Dummy](rest, "", dummyClusterVersion)
	dummyResource = rest.dummy

	// Fill in each resource, pointing back to the same rest
	// NOTE: to add new type you need to update VastResourceType generic
	rest.Versions = newResource[Version](rest, "versions", dummyClusterVersion)
	rest.VTasks = newResource[VTask](rest, "vtasks", dummyClusterVersion)
	rest.Quotas = newResource[Quota](rest, "quotas", dummyClusterVersion)
	rest.Views = newResource[View](rest, "views", dummyClusterVersion)
	rest.VipPools = newResource[VipPool](rest, "vippools", dummyClusterVersion)
	rest.Users = newResource[User](rest, "users", dummyClusterVersion)
	rest.UserKeys = newResource[UserKey](rest, "users/%d/access_keys", dummyClusterVersion)
	rest.Snapshots = newResource[Snapshot](rest, "snapshots", dummyClusterVersion)
	rest.BlockHosts = newResource[BlockHost](rest, "blockhosts", "5.3.0")
	rest.Volumes = newResource[Volume](rest, "volumes", "5.3.0")
	rest.BlockHostMappings = newResource[BlockHostMapping](rest, "blockhostvolumes", "5.3.0")
	rest.Cnodes = newResource[Cnode](rest, "cnodes", dummyClusterVersion)
	rest.QosPolicies = newResource[QosPolicy](rest, "qospolicies", dummyClusterVersion)
	rest.Dns = newResource[Dns](rest, "dns", dummyClusterVersion)
	rest.ViewPolies = newResource[ViewPolicy](rest, "viewpolicies", dummyClusterVersion)
	rest.Groups = newResource[Group](rest, "groups", dummyClusterVersion)
	rest.Nis = newResource[Nis](rest, "nis", dummyClusterVersion)
	rest.Tenants = newResource[Tenant](rest, "tenants", dummyClusterVersion)
	rest.Ldaps = newResource[Ldap](rest, "ldaps", dummyClusterVersion)
	rest.S3LifeCycleRules = newResource[S3LifeCycleRule](rest, "s3lifecyclerules", dummyClusterVersion)
	rest.ActiveDirectories = newResource[ActiveDirectory](rest, "activedirectory", dummyClusterVersion)
	rest.S3Policies = newResource[S3Policy](rest, "s3userpolicies", dummyClusterVersion)
	rest.ProtectedPaths = newResource[ProtectedPath](rest, "protectedpaths", dummyClusterVersion)
	rest.GlobalSnapshotStreams = newResource[GlobalSnapshotStream](rest, "globalsnapstreams", dummyClusterVersion)
	rest.ReplicationPeers = newResource[ReplicationPeers](rest, "nativereplicationremotetargets", dummyClusterVersion)
	rest.ProtectionPolicies = newResource[ProtectionPolicy](rest, "protectionpolicies", dummyClusterVersion)
	rest.S3replicationPeers = newResource[S3replicationPeers](rest, "replicationtargets", dummyClusterVersion)
	rest.Realms = newResource[Realm](rest, "realms", dummyClusterVersion)
	rest.Roles = newResource[Role](rest, "roles", dummyClusterVersion)
	rest.NonLocalUsers = newResource[NonLocalUser](rest, "users/query", dummyClusterVersion)
	rest.NonLocalGroups = newResource[NonLocalGroup](rest, "groups/query", dummyClusterVersion)
	rest.ApiTokens = newResource[ApiToken](rest, "apitokens", "5.3.0")
	rest.KafkaBrokers = newResource[KafkaBroker](rest, "kafkabrokers", dummyClusterVersion)

	return rest, nil
}

// BuildUrl Helper method to build full URL from path, query and api version.
// NOTE: Path is not full url. schema/host/port are taken from provided config. Path represents sub-resource
func (rest *VMSRest) BuildUrl(path, query, apiVer string) (string, error) {
	return buildUrl(rest.Session, path, query, apiVer)
}

func (rest *VMSRest) SetCtx(ctx context.Context) {
	rest.ctx = ctx
}

func newResource[T VastResourceType](rest *VMSRest, resourcePath, availableFromVersion string) *T {
	var availableFrom *version.Version
	if availableFromVersion == dummyClusterVersion {
		availableFrom = nil
	} else {
		availableFrom, _ = version.NewVersion(availableFromVersion)
	}
	resourceType := reflect.TypeOf(T{}).Name()
	resource := &T{
		&VastResource{
			resourcePath:         resourcePath,
			resourceType:         resourceType,
			rest:                 rest,
			availableFromVersion: availableFrom,
		},
	}
	if res, ok := any(resource).(VastResourceAPI); ok {
		rest.resourceMap[resourceType] = res
	} else {
		panic(fmt.Sprintf("Resource %s doesnt implement VastResource interface!", resourceType))
	}
	return resource
}
