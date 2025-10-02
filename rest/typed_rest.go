package rest

import (
	"context"
	"fmt"
	"reflect"

	"github.com/vast-data/go-vast-client/core"
	"github.com/vast-data/go-vast-client/resources/typed"
)

// TypedVastResourceType defines the interface constraint for all typed resources.
// Uses interface-based constraint to avoid Go's 100 union term limitation.
// All typed resources implement this by embedding *core.TypedVastResource.
type TypedVastResourceType interface {
	GetResourceType() string
}

type TypedVMSRest struct {
	Untyped *UntypedVMSRest

	ActiveDirectories          *typed.ActiveDirectory
	Alarms                     *typed.Alarm
	Analyticses                *typed.Analytics
	ApiTokens                  *typed.ApiToken
	BGPConfigs                 *typed.BGPConfig
	BasicSettingses            *typed.BasicSettings
	BigCatalogConfigs          *typed.BigCatalogConfig
	BigCatalogIndexedColumnses *typed.BigCatalogIndexedColumns
	BlockHosts                 *typed.BlockHost
	BlockHostMappings          *typed.BlockHostMapping
	CallhomeConfigses          *typed.CallhomeConfigs
	Capacities                 *typed.Capacity
	Carriers                   *typed.Carrier
	Cboxes                     *typed.Cbox
	Certificates               *typed.Certificate
	ChallengeTokenses          *typed.ChallengeTokens
	Clusters                   *typed.Cluster
	Cnodes                     *typed.Cnode
	CnodeGroups                *typed.CnodeGroup
	Columns                    *typed.Column
	Configs                    *typed.Config
	Dboxes                     *typed.Dbox
	Deltas                     *typed.Delta
	Dnodes                     *typed.Dnode
	Dnses                      *typed.Dns
	Dtrays                     *typed.Dtray
	Eboxes                     *typed.Ebox
	EncryptedPaths             *typed.EncryptedPath
	EncryptionGroups           *typed.EncryptionGroup
	Envs                       *typed.Env
	Events                     *typed.Event
	EventDefinitions           *typed.EventDefinition
	EventDefinitionConfigs     *typed.EventDefinitionConfig
	Fans                       *typed.Fan
	Folders                    *typed.Folder
	Filesystems                *typed.Filesystem
	GlobalSnapshotStreams      *typed.GlobalSnapshotStream
	Groups                     *typed.Group
	IamRoles                   *typed.IamRole
	Injectionses               *typed.Injections
	Indestructibility          *typed.Indestructibility
	IoDatas                    *typed.IoData
	KafkaBrokers               *typed.KafkaBroker
	Kerberos                   *typed.Kerberos
	Ldaps                      *typed.Ldap
	Licenses                   *typed.License
	LocalProviders             *typed.LocalProvider
	LocalS3Keys                *typed.LocalS3Key
	ManageApplicationses       *typed.ManageApplications
	Managers                   *typed.Manager
	Metricses                  *typed.Metrics
	Modules                    *typed.Module
	Monitors                   *typed.Monitor
	Nics                       *typed.Nic
	NicPorts                   *typed.NicPort
	Nises                      *typed.Nis
	Nvrams                     *typed.Nvram
	Oidc                       *typed.Oidc
	Permissionses              *typed.Permissions
	Ports                      *typed.Port
	Projections                *typed.Projection
	ProjectionColumns          *typed.ProjectionColumn
	PrometheusMetricses        *typed.PrometheusMetrics
	ProtectedPaths             *typed.ProtectedPath
	ProtectionPolicies         *typed.ProtectionPolicy
	Psus                       *typed.Psu
	QosPolicies                *typed.QosPolicy
	Quotas                     *typed.Quota
	QuotaEntityInfos           *typed.QuotaEntityInfo
	Racks                      *typed.Rack
	Realms                     *typed.Realm
	ReplicationPeerses         *typed.ReplicationPeers
	ReplicationPolicies        *typed.ReplicationPolicy
	ReplicationRestorePoints   *typed.ReplicationRestorePoint
	ReplicationStreams         *typed.ReplicationStream
	Roles                      *typed.Role
	S3Keyses                   *typed.S3Keys
	S3LifeCycleRules           *typed.S3LifeCycleRule
	S3Policies                 *typed.S3Policy
	S3replicationPeerses       *typed.S3replicationPeers
	Schemas                    *typed.Schema
	SettingDiffs               *typed.SettingDiff
	Snapshots                  *typed.Snapshot
	SnapshotPolicies           *typed.SnapshotPolicy
	Ssds                       *typed.Ssd
	SubnetManagers             *typed.SubnetManager
	SupportBundleses           *typed.SupportBundles
	SupportedDrivers           *typed.SupportedDrivers
	Switches                   *typed.Switch
	Tables                     *typed.Table
	Tenants                    *typed.Tenant
	Topics                     *typed.Topic
	Users                      *typed.User
	UserQuotas                 *typed.UserQuota
	VTasks                     *typed.VTask
	VastAuditLogs              *typed.VastAuditLog
	Versions                   *typed.Version
	Views                      *typed.View
	ViewPolicies               *typed.ViewPolicy
	Vips                       *typed.Vip
	VipPools                   *typed.VipPool
	Vmses                      *typed.Vms
	Volumes                    *typed.Volume
	VpnTunnels                 *typed.VpnTunnel
	WebHooks                   *typed.WebHook
	Hosts                      *typed.Host
	VirtualMachines            *typed.VirtualMachine
}

func NewTypedVMSRest(config *core.VMSConfig) (*TypedVMSRest, error) {
	untyped, err := NewUntypedVMSRest(config)
	if err != nil {
		return nil, err
	}

	rest := &TypedVMSRest{
		Untyped: untyped,
	}

	// Set external context
	if config.Context != nil {
		rest.SetCtx(config.Context)
	}

	rest.ActiveDirectories = newTypedResource[typed.ActiveDirectory](rest)
	rest.Alarms = newTypedResource[typed.Alarm](rest)
	rest.Analyticses = newTypedResource[typed.Analytics](rest)
	rest.ApiTokens = newTypedResource[typed.ApiToken](rest)
	rest.BGPConfigs = newTypedResource[typed.BGPConfig](rest)
	rest.BasicSettingses = newTypedResource[typed.BasicSettings](rest)
	rest.BigCatalogConfigs = newTypedResource[typed.BigCatalogConfig](rest)
	rest.BigCatalogIndexedColumnses = newTypedResource[typed.BigCatalogIndexedColumns](rest)
	rest.BlockHosts = newTypedResource[typed.BlockHost](rest)
	rest.BlockHostMappings = newTypedResource[typed.BlockHostMapping](rest)
	rest.CallhomeConfigses = newTypedResource[typed.CallhomeConfigs](rest)
	rest.Capacities = newTypedResource[typed.Capacity](rest)
	rest.Carriers = newTypedResource[typed.Carrier](rest)
	rest.Cboxes = newTypedResource[typed.Cbox](rest)
	rest.Certificates = newTypedResource[typed.Certificate](rest)
	rest.ChallengeTokenses = newTypedResource[typed.ChallengeTokens](rest)
	rest.Clusters = newTypedResource[typed.Cluster](rest)
	rest.Cnodes = newTypedResource[typed.Cnode](rest)
	rest.CnodeGroups = newTypedResource[typed.CnodeGroup](rest)
	rest.Columns = newTypedResource[typed.Column](rest)
	rest.Configs = newTypedResource[typed.Config](rest)
	rest.Dboxes = newTypedResource[typed.Dbox](rest)
	rest.Deltas = newTypedResource[typed.Delta](rest)
	rest.Dnodes = newTypedResource[typed.Dnode](rest)
	rest.Dnses = newTypedResource[typed.Dns](rest)
	rest.Dtrays = newTypedResource[typed.Dtray](rest)
	rest.Eboxes = newTypedResource[typed.Ebox](rest)
	rest.EncryptedPaths = newTypedResource[typed.EncryptedPath](rest)
	rest.EncryptionGroups = newTypedResource[typed.EncryptionGroup](rest)
	rest.Envs = newTypedResource[typed.Env](rest)
	rest.Events = newTypedResource[typed.Event](rest)
	rest.EventDefinitions = newTypedResource[typed.EventDefinition](rest)
	rest.EventDefinitionConfigs = newTypedResource[typed.EventDefinitionConfig](rest)
	rest.Fans = newTypedResource[typed.Fan](rest)
	rest.Folders = newTypedResource[typed.Folder](rest)
	rest.Filesystems = newTypedResource[typed.Filesystem](rest)
	rest.GlobalSnapshotStreams = newTypedResource[typed.GlobalSnapshotStream](rest)
	rest.Groups = newTypedResource[typed.Group](rest)
	rest.IamRoles = newTypedResource[typed.IamRole](rest)
	rest.Injectionses = newTypedResource[typed.Injections](rest)
	rest.Indestructibility = newTypedResource[typed.Indestructibility](rest)
	rest.IoDatas = newTypedResource[typed.IoData](rest)
	rest.KafkaBrokers = newTypedResource[typed.KafkaBroker](rest)
	rest.Kerberos = newTypedResource[typed.Kerberos](rest)
	rest.Ldaps = newTypedResource[typed.Ldap](rest)
	rest.Licenses = newTypedResource[typed.License](rest)
	rest.LocalProviders = newTypedResource[typed.LocalProvider](rest)
	rest.LocalS3Keys = newTypedResource[typed.LocalS3Key](rest)
	rest.ManageApplicationses = newTypedResource[typed.ManageApplications](rest)
	rest.Managers = newTypedResource[typed.Manager](rest)
	rest.Metricses = newTypedResource[typed.Metrics](rest)
	rest.Modules = newTypedResource[typed.Module](rest)
	rest.Monitors = newTypedResource[typed.Monitor](rest)
	rest.Nics = newTypedResource[typed.Nic](rest)
	rest.NicPorts = newTypedResource[typed.NicPort](rest)
	rest.Nises = newTypedResource[typed.Nis](rest)
	rest.Nvrams = newTypedResource[typed.Nvram](rest)
	rest.Oidc = newTypedResource[typed.Oidc](rest)
	rest.Permissionses = newTypedResource[typed.Permissions](rest)
	rest.Ports = newTypedResource[typed.Port](rest)
	rest.Projections = newTypedResource[typed.Projection](rest)
	rest.ProjectionColumns = newTypedResource[typed.ProjectionColumn](rest)
	rest.PrometheusMetricses = newTypedResource[typed.PrometheusMetrics](rest)
	rest.ProtectedPaths = newTypedResource[typed.ProtectedPath](rest)
	rest.ProtectionPolicies = newTypedResource[typed.ProtectionPolicy](rest)
	rest.Psus = newTypedResource[typed.Psu](rest)
	rest.QosPolicies = newTypedResource[typed.QosPolicy](rest)
	rest.Quotas = newTypedResource[typed.Quota](rest)
	rest.QuotaEntityInfos = newTypedResource[typed.QuotaEntityInfo](rest)
	rest.Racks = newTypedResource[typed.Rack](rest)
	rest.Realms = newTypedResource[typed.Realm](rest)
	rest.ReplicationPeerses = newTypedResource[typed.ReplicationPeers](rest)
	rest.ReplicationPolicies = newTypedResource[typed.ReplicationPolicy](rest)
	rest.ReplicationRestorePoints = newTypedResource[typed.ReplicationRestorePoint](rest)
	rest.ReplicationStreams = newTypedResource[typed.ReplicationStream](rest)
	rest.Roles = newTypedResource[typed.Role](rest)
	rest.S3Keyses = newTypedResource[typed.S3Keys](rest)
	rest.S3LifeCycleRules = newTypedResource[typed.S3LifeCycleRule](rest)
	rest.S3Policies = newTypedResource[typed.S3Policy](rest)
	rest.S3replicationPeerses = newTypedResource[typed.S3replicationPeers](rest)
	rest.Schemas = newTypedResource[typed.Schema](rest)
	rest.SettingDiffs = newTypedResource[typed.SettingDiff](rest)
	rest.Snapshots = newTypedResource[typed.Snapshot](rest)
	rest.SnapshotPolicies = newTypedResource[typed.SnapshotPolicy](rest)
	rest.Ssds = newTypedResource[typed.Ssd](rest)
	rest.SubnetManagers = newTypedResource[typed.SubnetManager](rest)
	rest.SupportBundleses = newTypedResource[typed.SupportBundles](rest)
	rest.SupportedDrivers = newTypedResource[typed.SupportedDrivers](rest)
	rest.Switches = newTypedResource[typed.Switch](rest)
	rest.Tables = newTypedResource[typed.Table](rest)
	rest.Tenants = newTypedResource[typed.Tenant](rest)
	rest.Topics = newTypedResource[typed.Topic](rest)
	rest.Users = newTypedResource[typed.User](rest)
	rest.UserQuotas = newTypedResource[typed.UserQuota](rest)
	rest.VTasks = newTypedResource[typed.VTask](rest)
	rest.VastAuditLogs = newTypedResource[typed.VastAuditLog](rest)
	rest.Versions = newTypedResource[typed.Version](rest)
	rest.Views = newTypedResource[typed.View](rest)
	rest.ViewPolicies = newTypedResource[typed.ViewPolicy](rest)
	rest.Vips = newTypedResource[typed.Vip](rest)
	rest.VipPools = newTypedResource[typed.VipPool](rest)
	rest.Vmses = newTypedResource[typed.Vms](rest)
	rest.Volumes = newTypedResource[typed.Volume](rest)
	rest.VpnTunnels = newTypedResource[typed.VpnTunnel](rest)
	rest.WebHooks = newTypedResource[typed.WebHook](rest)
	rest.Hosts = newTypedResource[typed.Host](rest)
	rest.VirtualMachines = newTypedResource[typed.VirtualMachine](rest)

	return rest, nil
}

func (rest *TypedVMSRest) GetSession() core.RESTSession {
	return rest.Untyped.Session
}

func (rest *TypedVMSRest) GetResourceMap() map[string]core.VastResourceAPIWithContext {
	return rest.Untyped.resourceMap
}

func (rest *TypedVMSRest) GetCtx() context.Context {
	return rest.Untyped.ctx
}

func (rest *TypedVMSRest) SetCtx(ctx context.Context) {
	rest.Untyped.ctx = ctx
}

func newTypedResource[T TypedVastResourceType](rest *TypedVMSRest) *T {
	// Get the concrete type from the type parameter
	var zero T
	t := reflect.TypeOf(zero)
	resourceType := t.Name()

	// Create new instance using reflection
	instance := reflect.New(t).Interface()

	// Create the typed resource
	typedRes := core.NewTypedVastResource(resourceType, rest.Untyped)

	// Set the embedded *TypedVastResource field using reflection
	// All typed resources embed *core.TypedVastResource
	val := reflect.ValueOf(instance).Elem()

	// Find the embedded *TypedVastResource field
	found := false
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if field.Type() == reflect.TypeOf((*core.TypedVastResource)(nil)) {
			if field.CanSet() {
				field.Set(reflect.ValueOf(typedRes))
				found = true
				break
			}
		}
	}

	if !found {
		panic(fmt.Sprintf("Resource %s does not embed *core.TypedVastResource or field is not settable", resourceType))
	}

	// Verify the corresponding untyped resource exists
	if _, ok := rest.Untyped.resourceMap[resourceType]; !ok {
		panic(fmt.Sprintf("untyped resource type %s not found in REST", resourceType))
	}

	// Return as pointer to the constrained type
	if result, ok := instance.(*T); ok {
		return result
	}
	panic(fmt.Sprintf("Failed to convert instance to type *%s", resourceType))
}
