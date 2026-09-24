package rest

import (
	"context"

	"github.com/vast-data/go-vast-client/core"
	"github.com/vast-data/go-vast-client/resources/typed"
	"github.com/vast-data/go-vast-client/rest/dataengine"
)

type TypedVMSRest struct {
	Untyped *UntypedVMSRest

	ActiveDirectories        *typed.ActiveDirectory
	Alarms                   *typed.Alarm
	Analytics                *typed.Analytics
	ApiTokens                *typed.ApiToken
	BGPConfigs               *typed.BGPConfig
	BasicSettings            *typed.BasicSettings
	BigCatalogConfigs        *typed.BigCatalogConfig
	BigCatalogIndexedColumns *typed.BigCatalogIndexedColumns
	BlockHosts               *typed.BlockHost
	BlockHostMappings        *typed.BlockHostMapping
	CallhomeConfigs          *typed.CallhomeConfigs
	Capacities               *typed.Capacity
	Carriers                 *typed.Carrier
	Cboxes                   *typed.Cbox
	Certificates             *typed.Certificate
	ChallengeTokens          *typed.ChallengeTokens
	Clusters                 *typed.Cluster
	Cnodes                   *typed.Cnode
	CnodeGroups              *typed.CnodeGroup
	Columns                  *typed.Column
	Configs                  *typed.Config
	Dboxes                   *typed.Dbox
	Deltas                   *typed.Delta
	Dnodes                   *typed.Dnode
	Dnses                    *typed.Dns
	Dtrays                   *typed.Dtray
	Eboxes                   *typed.Ebox
	EncryptedPaths           *typed.EncryptedPath
	EncryptionGroups         *typed.EncryptionGroup
	Envs                     *typed.Env
	Events                   *typed.Event
	EventDefinitions         *typed.EventDefinition
	EventDefinitionConfigs   *typed.EventDefinitionConfig
	Fans                     *typed.Fan
	Folders                  *typed.Folder
	Filesystems              *typed.Filesystem
	GlobalSnapshotStreams    *typed.GlobalSnapshotStream
	Groups                   *typed.Group
	IamRoles                 *typed.IamRole
	Injectionses             *typed.Injections
	Indestructibility        *typed.Indestructibility
	IoDatas                  *typed.IoData
	KafkaBrokers             *typed.KafkaBroker
	Kerberos                 *typed.Kerberos
	Ldaps                    *typed.Ldap
	Licenses                 *typed.License
	LocalProviders           *typed.LocalProvider
	LocalS3Keys              *typed.LocalS3Key
	ManageApplications       *typed.ManageApplications
	Managers                 *typed.Manager
	Metrics                  *typed.Metrics
	Modules                  *typed.Module
	Monitors                 *typed.Monitor
	Nics                     *typed.Nic
	NicPorts                 *typed.NicPort
	Nises                    *typed.Nis
	Nvrams                   *typed.Nvram
	Oidc                     *typed.Oidc
	Permissions              *typed.Permissions
	Ports                    *typed.Port
	Projections              *typed.Projection
	ProjectionColumns        *typed.ProjectionColumn
	PrometheusMetrics        *typed.PrometheusMetrics
	ProtectedPaths           *typed.ProtectedPath
	ProtectionPolicies       *typed.ProtectionPolicy
	Psus                     *typed.Psu
	QosPolicies              *typed.QosPolicy
	Quotas                   *typed.Quota
	QuotaEntityInfos         *typed.QuotaEntityInfo
	Racks                    *typed.Rack
	Realms                   *typed.Realm
	ReplicationPeers         *typed.ReplicationPeers
	ReplicationPolicies      *typed.ReplicationPolicy
	ReplicationRestorePoints *typed.ReplicationRestorePoint
	ReplicationStreams       *typed.ReplicationStream
	Roles                    *typed.Role
	S3Keys                   *typed.S3Keys
	S3LifeCycleRules         *typed.S3LifeCycleRule
	S3Policies               *typed.S3Policy
	S3ReplicationPeers       *typed.S3replicationPeers
	Schemas                  *typed.Schema
	SettingDiffs             *typed.SettingDiff
	Snapshots                *typed.Snapshot
	SnapshotPolicies         *typed.SnapshotPolicy
	Ssds                     *typed.Ssd
	SubnetManagers           *typed.SubnetManager
	SupportBundles           *typed.SupportBundles
	SupportedDrivers         *typed.SupportedDrivers
	Switches                 *typed.Switch
	Tables                   *typed.Table
	Tenants                  *typed.Tenant
	Topics                   *typed.Topic
	Users                    *typed.User
	UserQuotas               *typed.UserQuota
	VTasks                   *typed.VTask
	VastAuditLogs            *typed.VastAuditLog
	VastDb                   *typed.VastDb
	Versions                 *typed.Version
	Views                    *typed.View
	ViewPolicies             *typed.ViewPolicy
	Vips                     *typed.Vip
	VipPools                 *typed.VipPool
	Vmses                    *typed.Vms
	Volumes                  *typed.Volume
	VpnTunnels               *typed.VpnTunnel
	WebHooks                 *typed.WebHook
	Hosts                    *typed.Host
	VirtualMachines          *typed.VirtualMachine
	BlobExpansions           *typed.BlobExpansion
	ComputeClusters          *typed.ComputeCluster
	EventBrokers             *typed.EventBroker
	OpenFiles                *typed.OpenFile
	OpenFileHandles          *typed.OpenFileHandle
	OpenFilesQueries         *typed.OpenFilesQuery
	QuotaGroups              *typed.QuotaGroup
	SupportBundlesQueue      *typed.SupportBundlesQueue
	TlsCertificates          *typed.TlsCertificate
	VastdbTables             *typed.VastdbTable

	// DataEngine is a nested typed VastRest (see rest/dataengine).
	DataEngine *dataengine.TypedRest
}

func NewTypedVMSRest(config *core.VMSConfig) (*TypedVMSRest, error) {
	untyped, err := NewUntypedVMSRest(config)
	if err != nil {
		return nil, err
	}

	rest := &TypedVMSRest{
		Untyped: untyped,
	}

	// Set context: use provided context or default to background context
	if config.Context != nil {
		rest.SetCtx(config.Context)
	} else {
		rest.SetCtx(context.Background())
	}

	rest.ActiveDirectories = core.NewTypedResource[typed.ActiveDirectory](rest.Untyped)
	rest.Alarms = core.NewTypedResource[typed.Alarm](rest.Untyped)
	rest.Analytics = core.NewTypedResource[typed.Analytics](rest.Untyped)
	rest.ApiTokens = core.NewTypedResource[typed.ApiToken](rest.Untyped)
	rest.BGPConfigs = core.NewTypedResource[typed.BGPConfig](rest.Untyped)
	rest.BasicSettings = core.NewTypedResource[typed.BasicSettings](rest.Untyped)
	rest.BigCatalogConfigs = core.NewTypedResource[typed.BigCatalogConfig](rest.Untyped)
	rest.BigCatalogIndexedColumns = core.NewTypedResource[typed.BigCatalogIndexedColumns](rest.Untyped)
	rest.BlockHosts = core.NewTypedResource[typed.BlockHost](rest.Untyped)
	rest.BlockHostMappings = core.NewTypedResource[typed.BlockHostMapping](rest.Untyped)
	rest.CallhomeConfigs = core.NewTypedResource[typed.CallhomeConfigs](rest.Untyped)
	rest.Capacities = core.NewTypedResource[typed.Capacity](rest.Untyped)
	rest.Carriers = core.NewTypedResource[typed.Carrier](rest.Untyped)
	rest.Cboxes = core.NewTypedResource[typed.Cbox](rest.Untyped)
	rest.Certificates = core.NewTypedResource[typed.Certificate](rest.Untyped)
	rest.ChallengeTokens = core.NewTypedResource[typed.ChallengeTokens](rest.Untyped)
	rest.Clusters = core.NewTypedResource[typed.Cluster](rest.Untyped)
	rest.Cnodes = core.NewTypedResource[typed.Cnode](rest.Untyped)
	rest.CnodeGroups = core.NewTypedResource[typed.CnodeGroup](rest.Untyped)
	rest.Columns = core.NewTypedResource[typed.Column](rest.Untyped)
	rest.Configs = core.NewTypedResource[typed.Config](rest.Untyped)
	rest.Dboxes = core.NewTypedResource[typed.Dbox](rest.Untyped)
	rest.Deltas = core.NewTypedResource[typed.Delta](rest.Untyped)
	rest.Dnodes = core.NewTypedResource[typed.Dnode](rest.Untyped)
	rest.Dnses = core.NewTypedResource[typed.Dns](rest.Untyped)
	rest.Dtrays = core.NewTypedResource[typed.Dtray](rest.Untyped)
	rest.Eboxes = core.NewTypedResource[typed.Ebox](rest.Untyped)
	rest.EncryptedPaths = core.NewTypedResource[typed.EncryptedPath](rest.Untyped)
	rest.EncryptionGroups = core.NewTypedResource[typed.EncryptionGroup](rest.Untyped)
	rest.Envs = core.NewTypedResource[typed.Env](rest.Untyped)
	rest.Events = core.NewTypedResource[typed.Event](rest.Untyped)
	rest.EventDefinitions = core.NewTypedResource[typed.EventDefinition](rest.Untyped)
	rest.EventDefinitionConfigs = core.NewTypedResource[typed.EventDefinitionConfig](rest.Untyped)
	rest.Fans = core.NewTypedResource[typed.Fan](rest.Untyped)
	rest.Folders = core.NewTypedResource[typed.Folder](rest.Untyped)
	rest.Filesystems = core.NewTypedResource[typed.Filesystem](rest.Untyped)
	rest.GlobalSnapshotStreams = core.NewTypedResource[typed.GlobalSnapshotStream](rest.Untyped)
	rest.Groups = core.NewTypedResource[typed.Group](rest.Untyped)
	rest.IamRoles = core.NewTypedResource[typed.IamRole](rest.Untyped)
	rest.Injectionses = core.NewTypedResource[typed.Injections](rest.Untyped)
	rest.Indestructibility = core.NewTypedResource[typed.Indestructibility](rest.Untyped)
	rest.IoDatas = core.NewTypedResource[typed.IoData](rest.Untyped)
	rest.KafkaBrokers = core.NewTypedResource[typed.KafkaBroker](rest.Untyped)
	rest.Kerberos = core.NewTypedResource[typed.Kerberos](rest.Untyped)
	rest.Ldaps = core.NewTypedResource[typed.Ldap](rest.Untyped)
	rest.Licenses = core.NewTypedResource[typed.License](rest.Untyped)
	rest.LocalProviders = core.NewTypedResource[typed.LocalProvider](rest.Untyped)
	rest.LocalS3Keys = core.NewTypedResource[typed.LocalS3Key](rest.Untyped)
	rest.ManageApplications = core.NewTypedResource[typed.ManageApplications](rest.Untyped)
	rest.Managers = core.NewTypedResource[typed.Manager](rest.Untyped)
	rest.Metrics = core.NewTypedResource[typed.Metrics](rest.Untyped)
	rest.Modules = core.NewTypedResource[typed.Module](rest.Untyped)
	rest.Monitors = core.NewTypedResource[typed.Monitor](rest.Untyped)
	rest.Nics = core.NewTypedResource[typed.Nic](rest.Untyped)
	rest.NicPorts = core.NewTypedResource[typed.NicPort](rest.Untyped)
	rest.Nises = core.NewTypedResource[typed.Nis](rest.Untyped)
	rest.Nvrams = core.NewTypedResource[typed.Nvram](rest.Untyped)
	rest.Oidc = core.NewTypedResource[typed.Oidc](rest.Untyped)
	rest.Permissions = core.NewTypedResource[typed.Permissions](rest.Untyped)
	rest.Ports = core.NewTypedResource[typed.Port](rest.Untyped)
	rest.Projections = core.NewTypedResource[typed.Projection](rest.Untyped)
	rest.ProjectionColumns = core.NewTypedResource[typed.ProjectionColumn](rest.Untyped)
	rest.PrometheusMetrics = core.NewTypedResource[typed.PrometheusMetrics](rest.Untyped)
	rest.ProtectedPaths = core.NewTypedResource[typed.ProtectedPath](rest.Untyped)
	rest.ProtectionPolicies = core.NewTypedResource[typed.ProtectionPolicy](rest.Untyped)
	rest.Psus = core.NewTypedResource[typed.Psu](rest.Untyped)
	rest.QosPolicies = core.NewTypedResource[typed.QosPolicy](rest.Untyped)
	rest.Quotas = core.NewTypedResource[typed.Quota](rest.Untyped)
	rest.QuotaEntityInfos = core.NewTypedResource[typed.QuotaEntityInfo](rest.Untyped)
	rest.Racks = core.NewTypedResource[typed.Rack](rest.Untyped)
	rest.Realms = core.NewTypedResource[typed.Realm](rest.Untyped)
	rest.ReplicationPeers = core.NewTypedResource[typed.ReplicationPeers](rest.Untyped)
	rest.ReplicationPolicies = core.NewTypedResource[typed.ReplicationPolicy](rest.Untyped)
	rest.ReplicationRestorePoints = core.NewTypedResource[typed.ReplicationRestorePoint](rest.Untyped)
	rest.ReplicationStreams = core.NewTypedResource[typed.ReplicationStream](rest.Untyped)
	rest.Roles = core.NewTypedResource[typed.Role](rest.Untyped)
	rest.S3Keys = core.NewTypedResource[typed.S3Keys](rest.Untyped)
	rest.S3LifeCycleRules = core.NewTypedResource[typed.S3LifeCycleRule](rest.Untyped)
	rest.S3Policies = core.NewTypedResource[typed.S3Policy](rest.Untyped)
	rest.S3ReplicationPeers = core.NewTypedResource[typed.S3replicationPeers](rest.Untyped)
	rest.Schemas = core.NewTypedResource[typed.Schema](rest.Untyped)
	rest.SettingDiffs = core.NewTypedResource[typed.SettingDiff](rest.Untyped)
	rest.Snapshots = core.NewTypedResource[typed.Snapshot](rest.Untyped)
	rest.SnapshotPolicies = core.NewTypedResource[typed.SnapshotPolicy](rest.Untyped)
	rest.Ssds = core.NewTypedResource[typed.Ssd](rest.Untyped)
	rest.SubnetManagers = core.NewTypedResource[typed.SubnetManager](rest.Untyped)
	rest.SupportBundles = core.NewTypedResource[typed.SupportBundles](rest.Untyped)
	rest.SupportedDrivers = core.NewTypedResource[typed.SupportedDrivers](rest.Untyped)
	rest.Switches = core.NewTypedResource[typed.Switch](rest.Untyped)
	rest.Tables = core.NewTypedResource[typed.Table](rest.Untyped)
	rest.Tenants = core.NewTypedResource[typed.Tenant](rest.Untyped)
	rest.Topics = core.NewTypedResource[typed.Topic](rest.Untyped)
	rest.Users = core.NewTypedResource[typed.User](rest.Untyped)
	rest.UserQuotas = core.NewTypedResource[typed.UserQuota](rest.Untyped)
	rest.VTasks = core.NewTypedResource[typed.VTask](rest.Untyped)
	rest.VastAuditLogs = core.NewTypedResource[typed.VastAuditLog](rest.Untyped)
	rest.VastDb = core.NewTypedResource[typed.VastDb](rest.Untyped)
	rest.Versions = core.NewTypedResource[typed.Version](rest.Untyped)
	rest.Views = core.NewTypedResource[typed.View](rest.Untyped)
	rest.ViewPolicies = core.NewTypedResource[typed.ViewPolicy](rest.Untyped)
	rest.Vips = core.NewTypedResource[typed.Vip](rest.Untyped)
	rest.VipPools = core.NewTypedResource[typed.VipPool](rest.Untyped)
	rest.Vmses = core.NewTypedResource[typed.Vms](rest.Untyped)
	rest.Volumes = core.NewTypedResource[typed.Volume](rest.Untyped)
	rest.VpnTunnels = core.NewTypedResource[typed.VpnTunnel](rest.Untyped)
	rest.WebHooks = core.NewTypedResource[typed.WebHook](rest.Untyped)
	rest.Hosts = core.NewTypedResource[typed.Host](rest.Untyped)
	rest.VirtualMachines = core.NewTypedResource[typed.VirtualMachine](rest.Untyped)
	rest.BlobExpansions = core.NewTypedResource[typed.BlobExpansion](rest.Untyped)
	rest.ComputeClusters = core.NewTypedResource[typed.ComputeCluster](rest.Untyped)
	rest.EventBrokers = core.NewTypedResource[typed.EventBroker](rest.Untyped)
	rest.OpenFiles = core.NewTypedResource[typed.OpenFile](rest.Untyped)
	rest.OpenFileHandles = core.NewTypedResource[typed.OpenFileHandle](rest.Untyped)
	rest.OpenFilesQueries = core.NewTypedResource[typed.OpenFilesQuery](rest.Untyped)
	rest.QuotaGroups = core.NewTypedResource[typed.QuotaGroup](rest.Untyped)
	rest.SupportBundlesQueue = core.NewTypedResource[typed.SupportBundlesQueue](rest.Untyped)
	rest.TlsCertificates = core.NewTypedResource[typed.TlsCertificate](rest.Untyped)
	rest.VastdbTables = core.NewTypedResource[typed.VastdbTable](rest.Untyped)

	// Nested DataEngine rest (serverless); shares session with parent VMS rest.
	rest.DataEngine = dataengine.NewTyped(untyped.DataEngine)

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

func (rest *TypedVMSRest) GetApiRoot() string {
	return rest.Untyped.GetApiRoot()
}

// String returns a log-friendly identity of this client: VMS host and auth mode.
func (rest *TypedVMSRest) String() string {
	return rest.Untyped.String()
}
