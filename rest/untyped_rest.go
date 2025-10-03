package rest

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/vast-data/go-vast-client/core"
	"github.com/vast-data/go-vast-client/resources/untyped"
)

type UntypedVMSRest struct {
	ctx         context.Context
	Session     core.RESTSession
	resourceMap map[string]core.VastResourceAPIWithContext // Map to store resources by resourceType

	dummy                  *core.Dummy
	Versions               *untyped.Version
	VTasks                 *untyped.VTask
	Quotas                 *untyped.Quota
	Views                  *untyped.View
	VipPools               *untyped.VipPool
	Users                  *untyped.User
	UserKeys               *untyped.UserKey
	Snapshots              *untyped.Snapshot
	BlockHosts             *untyped.BlockHost
	Volumes                *untyped.Volume
	BlockHostMappings      *untyped.BlockHostMapping
	Cnodes                 *untyped.Cnode
	QosPolicies            *untyped.QosPolicy
	Dns                    *untyped.Dns
	ViewPolies             *untyped.ViewPolicy
	Groups                 *untyped.Group
	Nis                    *untyped.Nis
	Tenants                *untyped.Tenant
	Ldaps                  *untyped.Ldap
	S3LifeCycleRules       *untyped.S3LifeCycleRule
	ActiveDirectories      *untyped.ActiveDirectory
	S3Policies             *untyped.S3Policy
	ProtectedPaths         *untyped.ProtectedPath
	GlobalSnapshotStreams  *untyped.GlobalSnapshotStream
	ReplicationPeers       *untyped.ReplicationPeers
	ProtectionPolicies     *untyped.ProtectionPolicy
	S3replicationPeers     *untyped.S3replicationPeers
	Realms                 *untyped.Realm
	Roles                  *untyped.Role
	NonLocalUsers          *untyped.NonLocalUser
	NonLocalGroups         *untyped.NonLocalGroup
	NonLocalUserKeys       *untyped.NonLocalUserKey
	ApiTokens              *untyped.ApiToken
	KafkaBrokers           *untyped.KafkaBroker
	Managers               *untyped.Manager
	Folders                *untyped.Folder
	EventDefinitions       *untyped.EventDefinition
	EventDefinitionConfigs *untyped.EventDefinitionConfig
	BGPConfigs             *untyped.BGPConfig
	Vms                    *untyped.Vms
	Topics                 *untyped.Topic
	LocalProviders         *untyped.LocalProvider
	LocalS3Keys            *untyped.LocalS3Key
	EncryptionGroups       *untyped.EncryptionGroup
	SamlConfigs            *untyped.SamlConfig
	Kerberos               *untyped.Kerberos
	Clusters               *untyped.Cluster
	SupportedDrivers       *untyped.SupportedDrivers
	Racks                  *untyped.Rack
	Fans                   *untyped.Fan
	Nics                   *untyped.Nic
	NicPorts               *untyped.NicPort
	IamRoles               *untyped.IamRole
	Oidcs                  *untyped.Oidc
	Vips                   *untyped.Vip
}

func NewUntypedVMSRest(config *core.VMSConfig) (*UntypedVMSRest, error) {
	config.Validate(
		core.WithAuth,
		core.WithHost,
		core.WithUserAgent,
		core.WithFillFn,
		core.WithApiVersion("v5"),
		core.WithTimeout(time.Second*30),
		core.WithMaxConnections(10),
		core.WithPort(443),
	)
	session, err := core.NewVMSSession(config)
	if err != nil {
		return nil, err
	}
	rest := &UntypedVMSRest{
		Session:     session,
		resourceMap: make(map[string]core.VastResourceAPIWithContext),
	}
	rest.dummy = newUntypedResource[core.Dummy](rest, "")

	// Fill in each resource, pointing back to the same rest
	// NOTE: to add new type you need to update VastResourceType generic
	rest.Versions = newUntypedResource[untyped.Version](rest, "versions")
	rest.VTasks = newUntypedResource[untyped.VTask](rest, "vtasks")
	rest.Quotas = newUntypedResource[untyped.Quota](rest, "quotas")
	rest.Views = newUntypedResource[untyped.View](rest, "views")
	rest.VipPools = newUntypedResource[untyped.VipPool](rest, "vippools")
	rest.Users = newUntypedResource[untyped.User](rest, "users")
	rest.UserKeys = newUntypedResource[untyped.UserKey](rest, "users/%d/access_keys")
	rest.Snapshots = newUntypedResource[untyped.Snapshot](rest, "snapshots")
	rest.BlockHosts = newUntypedResource[untyped.BlockHost](rest, "blockhosts")
	rest.Volumes = newUntypedResource[untyped.Volume](rest, "volumes")
	rest.BlockHostMappings = newUntypedResource[untyped.BlockHostMapping](rest, "blockhostvolumes")
	rest.Cnodes = newUntypedResource[untyped.Cnode](rest, "cnodes")
	rest.QosPolicies = newUntypedResource[untyped.QosPolicy](rest, "qospolicies")
	rest.Dns = newUntypedResource[untyped.Dns](rest, "dns")
	rest.ViewPolies = newUntypedResource[untyped.ViewPolicy](rest, "viewpolicies")
	rest.Groups = newUntypedResource[untyped.Group](rest, "groups")
	rest.Nis = newUntypedResource[untyped.Nis](rest, "nis")
	rest.Tenants = newUntypedResource[untyped.Tenant](rest, "tenants")
	rest.Ldaps = newUntypedResource[untyped.Ldap](rest, "ldaps")
	rest.S3LifeCycleRules = newUntypedResource[untyped.S3LifeCycleRule](rest, "s3lifecyclerules")
	rest.ActiveDirectories = newUntypedResource[untyped.ActiveDirectory](rest, "activedirectory")
	rest.S3Policies = newUntypedResource[untyped.S3Policy](rest, "s3policies")
	rest.ProtectedPaths = newUntypedResource[untyped.ProtectedPath](rest, "protectedpaths")
	rest.GlobalSnapshotStreams = newUntypedResource[untyped.GlobalSnapshotStream](rest, "globalsnapstreams")
	rest.ReplicationPeers = newUntypedResource[untyped.ReplicationPeers](rest, "nativereplicationremotetargets")
	rest.ProtectionPolicies = newUntypedResource[untyped.ProtectionPolicy](rest, "protectionpolicies")
	rest.S3replicationPeers = newUntypedResource[untyped.S3replicationPeers](rest, "replicationtargets")
	rest.Realms = newUntypedResource[untyped.Realm](rest, "realms")
	rest.Roles = newUntypedResource[untyped.Role](rest, "roles")
	rest.NonLocalUsers = newUntypedResource[untyped.NonLocalUser](rest, "users/query")
	rest.NonLocalGroups = newUntypedResource[untyped.NonLocalGroup](rest, "groups/query")
	rest.NonLocalUserKeys = newUntypedResource[untyped.NonLocalUserKey](rest, "users/non_local_keys")
	rest.ApiTokens = newUntypedResource[untyped.ApiToken](rest, "apitokens")
	rest.KafkaBrokers = newUntypedResource[untyped.KafkaBroker](rest, "kafkabrokers")
	rest.Managers = newUntypedResource[untyped.Manager](rest, "managers")
	rest.Folders = newUntypedResource[untyped.Folder](rest, "folders")
	rest.EventDefinitions = newUntypedResource[untyped.EventDefinition](rest, "eventdefinitions")
	rest.EventDefinitionConfigs = newUntypedResource[untyped.EventDefinitionConfig](rest, "eventdefinitionconfigs")
	rest.BGPConfigs = newUntypedResource[untyped.BGPConfig](rest, "bgpconfigs")
	rest.Vms = newUntypedResource[untyped.Vms](rest, "vms")
	rest.Topics = newUntypedResource[untyped.Topic](rest, "topics")
	rest.LocalProviders = newUntypedResource[untyped.LocalProvider](rest, "localproviders")
	rest.LocalS3Keys = newUntypedResource[untyped.LocalS3Key](rest, "locals3keys")
	rest.EncryptionGroups = newUntypedResource[untyped.EncryptionGroup](rest, "encryptiongroups")
	rest.SamlConfigs = newUntypedResource[untyped.SamlConfig](rest, "vms/%d/saml_config")
	rest.Kerberos = newUntypedResource[untyped.Kerberos](rest, "kerberos")
	rest.Clusters = newUntypedResource[untyped.Cluster](rest, "clusters")
	rest.SupportedDrivers = newUntypedResource[untyped.SupportedDrivers](rest, "supporteddrives")
	rest.Racks = newUntypedResource[untyped.Rack](rest, "racks")
	rest.Fans = newUntypedResource[untyped.Fan](rest, "fans")
	rest.Nics = newUntypedResource[untyped.Nic](rest, "nics")
	rest.NicPorts = newUntypedResource[untyped.NicPort](rest, "nicports")
	rest.IamRoles = newUntypedResource[untyped.IamRole](rest, "iamroles")
	rest.Oidcs = newUntypedResource[untyped.Oidc](rest, "oidcs")
	rest.Vips = newUntypedResource[untyped.Vip](rest, "vips")

	return rest, nil
}

func (rest *UntypedVMSRest) GetSession() core.RESTSession {
	return rest.Session
}

func (rest *UntypedVMSRest) GetResourceMap() map[string]core.VastResourceAPIWithContext {
	return rest.resourceMap
}

func (rest *UntypedVMSRest) GetCtx() context.Context {
	return rest.ctx
}

func (rest *UntypedVMSRest) SetCtx(ctx context.Context) {
	rest.ctx = ctx
}

func newUntypedResource[T UntypedVastResourceType](rest *UntypedVMSRest, resourcePath string) *T {
	resourceType := reflect.TypeOf(T{}).Name()
	resource := &T{
		core.NewVastResource(resourcePath, resourceType, rest),
	}
	if res, ok := any(resource).(core.VastResourceAPIWithContext); ok {
		rest.resourceMap[resourceType] = res
	} else {
		panic(fmt.Sprintf("Resource %s doesnt implement VastResource interface!", resourceType))
	}
	return resource
}
