package rest

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/vast-data/go-vast-client/core"
	"github.com/vast-data/go-vast-client/resources/untyped"
)

// UntypedVastResourceType defines the interface constraint for all untyped resources.
// Uses interface-based constraint to avoid Go's 100 union term limitation.
type UntypedVastResourceType interface {
	core.VastResourceAPIWithContext
}

// Bit flags representing which CRUD operations are supported
const (
	C = core.C
	L = core.L
	R = core.R
	U = core.U
	D = core.D
)

type UntypedVMSRest struct {
	ctx         context.Context
	Session     core.RESTSession
	resourceMap map[string]core.VastResourceAPIWithContext // Map to store resources by resourceType

	ActiveDirectories        *untyped.ActiveDirectory
	Alarms                   *untyped.Alarm
	Analytics                *untyped.Analytics
	ApiTokens                *untyped.ApiToken
	BasicSettings            *untyped.BasicSettings
	BGPConfigs               *untyped.BGPConfig
	BigCatalogConfigs        *untyped.BigCatalogConfig
	BigCatalogIndexedColumns *untyped.BigCatalogIndexedColumns
	BlockHosts               *untyped.BlockHost
	// +apiall:extraMethod:PATCH=/blockmappings/bulk/
	BlockHostMappings *untyped.BlockHostMapping
	CallhomeConfigs   *untyped.CallhomeConfigs
	Capacities        *untyped.Capacity
	Carriers          *untyped.Carrier
	Cboxes            *untyped.Cbox
	Certificates      *untyped.Certificate
	ChallengeTokens   *untyped.ChallengeTokens
	Clusters          *untyped.Cluster
	// +apiall:extraMethod:GET|PATCH=/cnodes/{id}/bgpconfig
	Cnodes      *untyped.Cnode
	CnodeGroups *untyped.CnodeGroup
	Columns     *untyped.Column
	// +apiexclude:extraMethod:GET|PATCH|DELETE=/config/{key}/
	Configs *untyped.Config
	Dboxes  *untyped.Dbox
	// +apiall:extraMethod:GET|PATCH=/delta/config/
	Deltas                 *untyped.Delta
	Dnodes                 *untyped.Dnode
	Dns                    *untyped.Dns
	Dtrays                 *untyped.Dtray
	Eboxes                 *untyped.Ebox
	EncryptedPaths         *untyped.EncryptedPath
	EncryptionGroups       *untyped.EncryptionGroup
	Envs                   *untyped.Env
	Events                 *untyped.Event
	EventDefinitions       *untyped.EventDefinition
	EventDefinitionConfigs *untyped.EventDefinitionConfig
	Fans                   *untyped.Fan
	Folders                *untyped.Folder
	Filesystems            *untyped.Filesystem
	GlobalSnapshotStreams  *untyped.GlobalSnapshotStream
	Groups                 *untyped.Group
	IamRoles               *untyped.IamRole
	Injections             *untyped.Injections
	Indestructibility      *untyped.Indestructibility
	IoData                 *untyped.IoData
	KafkaBrokers           *untyped.KafkaBroker
	// +apiexclude:extraMethod:PUT=/kerberos/{id}/keytab/
	Kerberos            *untyped.Kerberos
	Ldaps               *untyped.Ldap
	Licenses            *untyped.License
	LocalProviders      *untyped.LocalProvider
	LocalS3Keys         *untyped.LocalS3Key
	ManagedApplications *untyped.ManageApplications
	Managers            *untyped.Manager
	Metrics             *untyped.Metrics
	Modules             *untyped.Module
	// +apiexclude:extraMethod:GET=/monitors/ad_hoc_query/
	Monitors                 *untyped.Monitor
	Nics                     *untyped.Nic
	NicPorts                 *untyped.NicPort
	Nis                      *untyped.Nis
	Nvrams                   *untyped.Nvram
	Oidcs                    *untyped.Oidc
	Permissions              *untyped.Permissions
	Ports                    *untyped.Port
	Projections              *untyped.Projection
	ProjectionColumns        *untyped.ProjectionColumn
	PrometheusMetrics        *untyped.PrometheusMetrics
	ProtectedPaths           *untyped.ProtectedPath
	ProtectionPolicies       *untyped.ProtectionPolicy
	Psus                     *untyped.Psu
	QosPolicies              *untyped.QosPolicy
	Quotas                   *untyped.Quota
	QuotaEntityInfos         *untyped.QuotaEntityInfo
	Racks                    *untyped.Rack
	Realms                   *untyped.Realm
	ReplicationPeers         *untyped.ReplicationPeers
	ReplicationPolicies      *untyped.ReplicationPolicy
	ReplicationRestorePoints *untyped.ReplicationRestorePoint
	ReplicationStreams       *untyped.ReplicationStream
	Roles                    *untyped.Role
	S3Keys                   *untyped.S3Keys
	S3LifeCycleRules         *untyped.S3LifeCycleRule
	S3Policies               *untyped.S3Policy
	S3replicationPeers       *untyped.S3replicationPeers
	Schemas                  *untyped.Schema
	SettingDiffs             *untyped.SettingDiff
	Snapshots                *untyped.Snapshot
	SnapshotPolicies         *untyped.SnapshotPolicy
	Ssds                     *untyped.Ssd
	SubnetManagers           *untyped.SubnetManager
	SupportBundles           *untyped.SupportBundles
	SupportedDrivers         *untyped.SupportedDrivers
	Switches                 *untyped.Switch
	Tables                   *untyped.Table
	Tenants                  *untyped.Tenant
	// +apiall:extraMethod:GET|POST|PATCH=/topics/
	Topics              *untyped.Topic
	Users               *untyped.User
	UserQuotas          *untyped.UserQuota
	VastAuditLogs       *untyped.VastAuditLog
	VastDb              *untyped.VastDb
	Versions            *untyped.Version
	Views               *untyped.View
	ViewPolicies        *untyped.ViewPolicy
	Vips                *untyped.Vip
	VipPools            *untyped.VipPool
	Vms                 *untyped.Vms
	Volumes             *untyped.Volume
	VpnTunnels          *untyped.VpnTunnel
	VTasks              *untyped.VTask
	WebHooks            *untyped.WebHook
	Hosts               *untyped.Host
	VirtualMachines     *untyped.VirtualMachine
	BlobExpansions      *untyped.BlobExpansion
	ComputeClusters     *untyped.ComputeCluster
	EventBrokers        *untyped.EventBroker
	OpenFiles           *untyped.OpenFile
	OpenFileHandles     *untyped.OpenFileHandle
	OpenFilesQueries    *untyped.OpenFilesQuery
	QuotaGroups         *untyped.QuotaGroup
	SupportBundlesQueue *untyped.SupportBundlesQueue
	TlsCertificates     *untyped.TlsCertificate
	VastdbTables        *untyped.VastdbTable
}

func NewUntypedVMSRest(config *core.VMSConfig) (*UntypedVMSRest, error) {
	if err := config.Validate(
		core.WithAuth,
		core.WithHost,
		core.WithUserAgent,
		core.WithFillFn,
		core.WithApiVersion("v5"),
		core.WithTimeout(time.Second*30),
		core.WithMaxConnections(10),
		core.WithPort(443),
	); err != nil {
		return nil, err
	}
	session, err := core.NewVMSSession(config)
	if err != nil {
		return nil, err
	}
	rest := &UntypedVMSRest{
		Session:     session,
		resourceMap: make(map[string]core.VastResourceAPIWithContext),
	}

	// Set context: use provided context or default to background context
	if config.Context != nil {
		rest.SetCtx(config.Context)
	} else {
		rest.SetCtx(context.Background())
	}

	// Fill in each resource, pointing back to the same rest
	rest.ActiveDirectories = newUntypedResource[untyped.ActiveDirectory](rest, "activedirectory", C, L, R, U, D)
	rest.Alarms = newUntypedResource[untyped.Alarm](rest, "alarms", L, R, U, D)
	rest.Analytics = newUntypedResource[untyped.Analytics](rest, "analytics", L, R)
	rest.ApiTokens = newUntypedResource[untyped.ApiToken](rest, "apitokens", C, L, R, U)
	rest.BasicSettings = newUntypedResource[untyped.BasicSettings](rest, "basicsettings", L)
	rest.BGPConfigs = newUntypedResource[untyped.BGPConfig](rest, "bgpconfigs", C, L, R, U, D)
	rest.BigCatalogConfigs = newUntypedResource[untyped.BigCatalogConfig](rest, "bigcatalogconfig", C, L, R, U, D)
	rest.BigCatalogIndexedColumns = newUntypedResource[untyped.BigCatalogIndexedColumns](rest, "bigcatalogindexedcolumns", L)
	rest.BlockHosts = newUntypedResource[untyped.BlockHost](rest, "blockhosts", C, L, R, U, D)
	rest.BlockHostMappings = newUntypedResource[untyped.BlockHostMapping](rest, "blockhostvolumes", L)
	rest.CallhomeConfigs = newUntypedResource[untyped.CallhomeConfigs](rest, "callhomeconfigs", C, L, R, U)
	rest.Capacities = newUntypedResource[untyped.Capacity](rest, "capacity", L)
	rest.Carriers = newUntypedResource[untyped.Carrier](rest, "carriers", L, R, U)
	rest.Cboxes = newUntypedResource[untyped.Cbox](rest, "cboxes", C, L, R, U, D)
	rest.Certificates = newUntypedResource[untyped.Certificate](rest, "certificates", C, L, R, U, D)
	rest.ChallengeTokens = newUntypedResource[untyped.ChallengeTokens](rest, "challengetokens", L, R)
	rest.Clusters = newUntypedResource[untyped.Cluster](rest, "clusters", C, L, R, U, D)
	rest.Cnodes = newUntypedResource[untyped.Cnode](rest, "cnodes", C, L, R, U, D)
	rest.CnodeGroups = newUntypedResource[untyped.CnodeGroup](rest, "cnodegroups", C, L, R, U, D)
	rest.Columns = newUntypedResource[untyped.Column](rest, "columns", L)
	rest.Configs = newUntypedResource[untyped.Config](rest, "config", L)
	rest.Dboxes = newUntypedResource[untyped.Dbox](rest, "dboxes", C, L, R, U, D)
	rest.Deltas = newUntypedResource[untyped.Delta](rest, "deltas", L)
	rest.Dnodes = newUntypedResource[untyped.Dnode](rest, "dnodes", C, L, R, U, D)
	rest.Dns = newUntypedResource[untyped.Dns](rest, "dns", C, L, R, U, D)
	rest.Dtrays = newUntypedResource[untyped.Dtray](rest, "dtrays", C, L, R, U, D)
	rest.Eboxes = newUntypedResource[untyped.Ebox](rest, "eboxes", C, L, R, U, D)
	rest.EncryptedPaths = newUntypedResource[untyped.EncryptedPath](rest, "encryptedpaths", C, L, R, U, D)
	rest.EncryptionGroups = newUntypedResource[untyped.EncryptionGroup](rest, "encryptiongroups", L, R)
	rest.Envs = newUntypedResource[untyped.Env](rest, "envs", L, R)
	rest.Events = newUntypedResource[untyped.Event](rest, "events", C, L, R)
	rest.EventDefinitions = newUntypedResource[untyped.EventDefinition](rest, "eventdefinitions", C, L, R, U)
	rest.EventDefinitionConfigs = newUntypedResource[untyped.EventDefinitionConfig](rest, "eventdefinitionconfigs", C, L, R, U)
	rest.Fans = newUntypedResource[untyped.Fan](rest, "fans", L, R)
	rest.Folders = newUntypedResource[untyped.Folder](rest, "folders")
	rest.Filesystems = newUntypedResource[untyped.Filesystem](rest, "filesystem")
	rest.GlobalSnapshotStreams = newUntypedResource[untyped.GlobalSnapshotStream](rest, "globalsnapstreams", C, L, R, U, D)
	rest.Groups = newUntypedResource[untyped.Group](rest, "groups", C, L, R, U, D)
	rest.IamRoles = newUntypedResource[untyped.IamRole](rest, "iamroles", C, L, R, U, D)
	rest.Injections = newUntypedResource[untyped.Injections](rest, "injections", C, L, R, U, D)
	rest.Indestructibility = newUntypedResource[untyped.Indestructibility](rest, "indestructibility", C, L, R, U)
	rest.IoData = newUntypedResource[untyped.IoData](rest, "iodata", L)
	rest.KafkaBrokers = newUntypedResource[untyped.KafkaBroker](rest, "kafkabrokers", C, L, R, U, D)
	rest.Kerberos = newUntypedResource[untyped.Kerberos](rest, "kerberos", C, L, R, U, D)
	rest.Ldaps = newUntypedResource[untyped.Ldap](rest, "ldaps", C, L, R, U, D)
	rest.Licenses = newUntypedResource[untyped.License](rest, "licenses", C, L, R, D)
	rest.LocalProviders = newUntypedResource[untyped.LocalProvider](rest, "localproviders", C, L, R, U, D)
	rest.LocalS3Keys = newUntypedResource[untyped.LocalS3Key](rest, "locals3keys", L)
	rest.ManagedApplications = newUntypedResource[untyped.ManageApplications](rest, "managedapplications", C, L, R, U, D)
	rest.Managers = newUntypedResource[untyped.Manager](rest, "managers", C, L, R, U, D)
	rest.Metrics = newUntypedResource[untyped.Metrics](rest, "metrics", L)
	rest.Modules = newUntypedResource[untyped.Module](rest, "modules", L, R)
	rest.Monitors = newUntypedResource[untyped.Monitor](rest, "monitors", C, L, R, U, D)
	rest.Nics = newUntypedResource[untyped.Nic](rest, "nics", L, R)
	rest.NicPorts = newUntypedResource[untyped.NicPort](rest, "nicports", L, R, U)
	rest.Nis = newUntypedResource[untyped.Nis](rest, "nis", C, L, R, U, D)
	rest.Nvrams = newUntypedResource[untyped.Nvram](rest, "nvrams", L, R, U, D)
	rest.Oidcs = newUntypedResource[untyped.Oidc](rest, "oidcs", C, L, R, U, D)
	rest.Permissions = newUntypedResource[untyped.Permissions](rest, "permissions", L, R)
	rest.Ports = newUntypedResource[untyped.Port](rest, "ports", L, R)
	rest.Projections = newUntypedResource[untyped.Projection](rest, "projections", C, L)
	rest.ProjectionColumns = newUntypedResource[untyped.ProjectionColumn](rest, "projectioncolumns", L)
	rest.PrometheusMetrics = newUntypedResource[untyped.PrometheusMetrics](rest, "prometheusmetrics", R)
	rest.ProtectedPaths = newUntypedResource[untyped.ProtectedPath](rest, "protectedpaths", C, L, R, U, D)
	rest.ProtectionPolicies = newUntypedResource[untyped.ProtectionPolicy](rest, "protectionpolicies", C, L, R, U, D)
	rest.Psus = newUntypedResource[untyped.Psu](rest, "psus", L, R)
	rest.QosPolicies = newUntypedResource[untyped.QosPolicy](rest, "qospolicies", C, L, R, U, D)
	rest.Quotas = newUntypedResource[untyped.Quota](rest, "quotas", C, L, R, U, D)
	rest.QuotaEntityInfos = newUntypedResource[untyped.QuotaEntityInfo](rest, "quotaentityinfo", L)
	rest.Racks = newUntypedResource[untyped.Rack](rest, "racks", C, L, R, U, D)
	rest.Realms = newUntypedResource[untyped.Realm](rest, "realms", C, L, R, U, D)
	rest.ReplicationPeers = newUntypedResource[untyped.ReplicationPeers](rest, "nativereplicationremotetargets", C, L, R, U, D)
	rest.ReplicationPolicies = newUntypedResource[untyped.ReplicationPolicy](rest, "replicationpolicies", C, L, R, U, D)
	rest.ReplicationRestorePoints = newUntypedResource[untyped.ReplicationRestorePoint](rest, "replicationrestorepoints", L, R)
	rest.ReplicationStreams = newUntypedResource[untyped.ReplicationStream](rest, "replicationstreams", C, L, R, U, D)
	rest.Roles = newUntypedResource[untyped.Role](rest, "roles", C, L, R, U, D)
	rest.S3Keys = newUntypedResource[untyped.S3Keys](rest, "s3keys", C, L)
	rest.S3LifeCycleRules = newUntypedResource[untyped.S3LifeCycleRule](rest, "s3lifecyclerules", C, L, R, U, D)
	rest.S3Policies = newUntypedResource[untyped.S3Policy](rest, "s3policies", C, L, R, U, D)
	rest.S3replicationPeers = newUntypedResource[untyped.S3replicationPeers](rest, "replicationtargets", C, L, R, U, D)
	rest.Schemas = newUntypedResource[untyped.Schema](rest, "schemas", C, L)
	rest.SettingDiffs = newUntypedResource[untyped.SettingDiff](rest, "settingdiff", R)
	rest.Snapshots = newUntypedResource[untyped.Snapshot](rest, "snapshots", C, L, R, U, D)
	rest.SnapshotPolicies = newUntypedResource[untyped.SnapshotPolicy](rest, "snapshotpolicies", C, L, R, U, D)
	rest.Ssds = newUntypedResource[untyped.Ssd](rest, "ssds", L, R, U, D)
	rest.SubnetManagers = newUntypedResource[untyped.SubnetManager](rest, "subnetmanagers", C, L, R, U, D)
	rest.SupportBundles = newUntypedResource[untyped.SupportBundles](rest, "supportbundles", C, L, R, U, D)
	rest.SupportedDrivers = newUntypedResource[untyped.SupportedDrivers](rest, "supporteddrives", C, L, R, U, D)
	rest.Switches = newUntypedResource[untyped.Switch](rest, "switches", C, L, R, U, D)
	rest.Tables = newUntypedResource[untyped.Table](rest, "tables", C, L, U)
	rest.Tenants = newUntypedResource[untyped.Tenant](rest, "tenants", C, L, R, U, D)
	rest.Topics = newUntypedResource[untyped.Topic](rest, "topics", C, L, U)
	rest.Users = newUntypedResource[untyped.User](rest, "users", C, L, R, U, D)
	rest.UserQuotas = newUntypedResource[untyped.UserQuota](rest, "userquotas", C, L, R, U, D)
	rest.VastAuditLogs = newUntypedResource[untyped.VastAuditLog](rest, "vastauditlog", C, L)
	rest.VastDb = newUntypedResource[untyped.VastDb](rest, "vastdb")
	rest.Versions = newUntypedResource[untyped.Version](rest, "versions", L, R)
	rest.Views = newUntypedResource[untyped.View](rest, "views", C, L, R, U, D)
	rest.ViewPolicies = newUntypedResource[untyped.ViewPolicy](rest, "viewpolicies", C, L, R, U, D)
	rest.Vips = newUntypedResource[untyped.Vip](rest, "vips", L, R)
	rest.VipPools = newUntypedResource[untyped.VipPool](rest, "vippools", C, L, R, U, D)
	rest.Vms = newUntypedResource[untyped.Vms](rest, "vms", L, R, U)
	rest.Volumes = newUntypedResource[untyped.Volume](rest, "volumes", C, L, R, U, D)
	rest.VpnTunnels = newUntypedResource[untyped.VpnTunnel](rest, "vpntunnels", C, L, R, U, D)
	rest.VTasks = newUntypedResource[untyped.VTask](rest, "vtasks", L, R, U)
	rest.WebHooks = newUntypedResource[untyped.WebHook](rest, "webhooks", C, L, R, U, D)
	rest.VirtualMachines = newUntypedResource[untyped.VirtualMachine](rest, "virtual-machines", L, R)
	rest.Hosts = newUntypedResource[untyped.Host](rest, "hosts", L, R)
	rest.BlobExpansions = newUntypedResource[untyped.BlobExpansion](rest, "blobexpansions", C)
	rest.ComputeClusters = newUntypedResource[untyped.ComputeCluster](rest, "computeclusters", C, L, R, U, D)
	rest.EventBrokers = newUntypedResource[untyped.EventBroker](rest, "eventbrokers", C, L, R, U, D)
	rest.OpenFiles = newUntypedResource[untyped.OpenFile](rest, "openfiles", L, R)
	rest.OpenFileHandles = newUntypedResource[untyped.OpenFileHandle](rest, "openfilehandles", L, R)
	rest.OpenFilesQueries = newUntypedResource[untyped.OpenFilesQuery](rest, "openfilesqueries", C, L, R, D)
	rest.QuotaGroups = newUntypedResource[untyped.QuotaGroup](rest, "quotagroups", C, L, R, U, D)
	rest.SupportBundlesQueue = newUntypedResource[untyped.SupportBundlesQueue](rest, "supportbundlesqueue", L)
	rest.TlsCertificates = newUntypedResource[untyped.TlsCertificate](rest, "tlscertificates", C, L, R, U, D)
	rest.VastdbTables = newUntypedResource[untyped.VastdbTable](rest, "vastdbtable")

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

func newUntypedResource[T UntypedVastResourceType](rest *UntypedVMSRest, resourcePath string, resourceOps ...core.ResourceOps) *T {
	// Get the concrete type from the type parameter
	var zero T
	t := reflect.TypeOf(zero)
	resourceType := t.Name()

	// Create new instance using reflection
	instance := reflect.New(t).Interface()

	// Create VastResource with parent reference for method discovery via reflection
	resource := core.NewVastResource(resourcePath, resourceType, rest, core.NewResourceOps(resourceOps...), instance)

	// Set the embedded *VastResource field using reflection
	// All untyped resources embed *core.VastResource
	val := reflect.ValueOf(instance).Elem()

	// Find the embedded *VastResource field
	found := false
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if field.Type() == reflect.TypeOf((*core.VastResource)(nil)) {
			if field.CanSet() {
				field.Set(reflect.ValueOf(resource))
				found = true
				break
			}
		}
	}

	if !found {
		panic(fmt.Sprintf("Resource %s does not embed *core.VastResource or field is not settable", resourceType))
	}

	// Register in resource map
	if res, ok := instance.(core.VastResourceAPIWithContext); ok {
		rest.resourceMap[resourceType] = res
	}

	// Return as pointer to the constrained type
	if result, ok := instance.(*T); ok {
		return result
	}
	panic(fmt.Sprintf("Failed to convert instance to type *%s", resourceType))
}
