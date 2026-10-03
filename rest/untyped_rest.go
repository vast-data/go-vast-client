package rest

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vast-data/go-vast-client/core"
	"github.com/vast-data/go-vast-client/resources/untyped"
	"github.com/vast-data/go-vast-client/rest/dataengine"
)

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
	resourceMap map[string]core.ResourceEntry // Map to store resources by resourceType
	// apiRoot is empty for the main VMS rest (/api/{version}/...).
	apiRoot string

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

	// DataEngine is a nested VastRest (see rest/dataengine).
	DataEngine *dataengine.UntypedRest
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
		resourceMap: make(map[string]core.ResourceEntry),
		apiRoot:     "", // main VMS rest
	}

	// Set context: use provided context or default to background context
	if config.Context != nil {
		rest.SetCtx(config.Context)
	} else {
		rest.SetCtx(context.Background())
	}

	// Fill in each resource, pointing back to the same rest
	rest.ActiveDirectories = core.NewUntypedResource[untyped.ActiveDirectory](rest, "activedirectory", C, L, R, U, D)
	rest.Alarms = core.NewUntypedResource[untyped.Alarm](rest, "alarms", L, R, U, D)
	rest.Analytics = core.NewUntypedResource[untyped.Analytics](rest, "analytics", L, R)
	rest.ApiTokens = core.NewUntypedResource[untyped.ApiToken](rest, "apitokens", C, L, R, U)
	rest.BasicSettings = core.NewUntypedResource[untyped.BasicSettings](rest, "basicsettings", L)
	rest.BGPConfigs = core.NewUntypedResource[untyped.BGPConfig](rest, "bgpconfigs", C, L, R, U, D)
	rest.BigCatalogConfigs = core.NewUntypedResource[untyped.BigCatalogConfig](rest, "bigcatalogconfig", C, L, R, U, D)
	rest.BigCatalogIndexedColumns = core.NewUntypedResource[untyped.BigCatalogIndexedColumns](rest, "bigcatalogindexedcolumns", L)
	rest.BlockHosts = core.NewUntypedResource[untyped.BlockHost](rest, "blockhosts", C, L, R, U, D)
	rest.BlockHostMappings = core.NewUntypedResource[untyped.BlockHostMapping](rest, "blockhostvolumes", L)
	rest.CallhomeConfigs = core.NewUntypedResource[untyped.CallhomeConfigs](rest, "callhomeconfigs", C, L, R, U)
	rest.Capacities = core.NewUntypedResource[untyped.Capacity](rest, "capacity", L)
	rest.Carriers = core.NewUntypedResource[untyped.Carrier](rest, "carriers", L, R, U)
	rest.Cboxes = core.NewUntypedResource[untyped.Cbox](rest, "cboxes", C, L, R, U, D)
	rest.Certificates = core.NewUntypedResource[untyped.Certificate](rest, "certificates", C, L, R, U, D)
	rest.ChallengeTokens = core.NewUntypedResource[untyped.ChallengeTokens](rest, "challengetokens", L, R)
	rest.Clusters = core.NewUntypedResource[untyped.Cluster](rest, "clusters", C, L, R, U, D)
	rest.Cnodes = core.NewUntypedResource[untyped.Cnode](rest, "cnodes", C, L, R, U, D)
	rest.CnodeGroups = core.NewUntypedResource[untyped.CnodeGroup](rest, "cnodegroups", C, L, R, U, D)
	rest.Columns = core.NewUntypedResource[untyped.Column](rest, "columns", L)
	rest.Configs = core.NewUntypedResource[untyped.Config](rest, "config", L)
	rest.Dboxes = core.NewUntypedResource[untyped.Dbox](rest, "dboxes", C, L, R, U, D)
	rest.Deltas = core.NewUntypedResource[untyped.Delta](rest, "deltas", L)
	rest.Dnodes = core.NewUntypedResource[untyped.Dnode](rest, "dnodes", C, L, R, U, D)
	rest.Dns = core.NewUntypedResource[untyped.Dns](rest, "dns", C, L, R, U, D)
	rest.Dtrays = core.NewUntypedResource[untyped.Dtray](rest, "dtrays", C, L, R, U, D)
	rest.Eboxes = core.NewUntypedResource[untyped.Ebox](rest, "eboxes", C, L, R, U, D)
	rest.EncryptedPaths = core.NewUntypedResource[untyped.EncryptedPath](rest, "encryptedpaths", C, L, R, U, D)
	rest.EncryptionGroups = core.NewUntypedResource[untyped.EncryptionGroup](rest, "encryptiongroups", L, R)
	rest.Envs = core.NewUntypedResource[untyped.Env](rest, "envs", L, R)
	rest.Events = core.NewUntypedResource[untyped.Event](rest, "events", C, L, R)
	rest.EventDefinitions = core.NewUntypedResource[untyped.EventDefinition](rest, "eventdefinitions", C, L, R, U)
	rest.EventDefinitionConfigs = core.NewUntypedResource[untyped.EventDefinitionConfig](rest, "eventdefinitionconfigs", C, L, R, U)
	rest.Fans = core.NewUntypedResource[untyped.Fan](rest, "fans", L, R)
	rest.Folders = core.NewUntypedResource[untyped.Folder](rest, "folders")
	rest.Filesystems = core.NewUntypedResource[untyped.Filesystem](rest, "filesystem")
	rest.GlobalSnapshotStreams = core.NewUntypedResource[untyped.GlobalSnapshotStream](rest, "globalsnapstreams", C, L, R, U, D)
	rest.Groups = core.NewUntypedResource[untyped.Group](rest, "groups", C, L, R, U, D)
	rest.IamRoles = core.NewUntypedResource[untyped.IamRole](rest, "iamroles", C, L, R, U, D)
	rest.Injections = core.NewUntypedResource[untyped.Injections](rest, "injections", C, L, R, U, D)
	rest.Indestructibility = core.NewUntypedResource[untyped.Indestructibility](rest, "indestructibility", C, L, R, U)
	rest.IoData = core.NewUntypedResource[untyped.IoData](rest, "iodata", L)
	rest.KafkaBrokers = core.NewUntypedResource[untyped.KafkaBroker](rest, "kafkabrokers", C, L, R, U, D)
	rest.Kerberos = core.NewUntypedResource[untyped.Kerberos](rest, "kerberos", C, L, R, U, D)
	rest.Ldaps = core.NewUntypedResource[untyped.Ldap](rest, "ldaps", C, L, R, U, D)
	rest.Licenses = core.NewUntypedResource[untyped.License](rest, "licenses", C, L, R, D)
	rest.LocalProviders = core.NewUntypedResource[untyped.LocalProvider](rest, "localproviders", C, L, R, U, D)
	rest.LocalS3Keys = core.NewUntypedResource[untyped.LocalS3Key](rest, "locals3keys", L)
	rest.ManagedApplications = core.NewUntypedResource[untyped.ManageApplications](rest, "managedapplications", C, L, R, U, D)
	rest.Managers = core.NewUntypedResource[untyped.Manager](rest, "managers", C, L, R, U, D)
	rest.Metrics = core.NewUntypedResource[untyped.Metrics](rest, "metrics", L)
	rest.Modules = core.NewUntypedResource[untyped.Module](rest, "modules", L, R)
	rest.Monitors = core.NewUntypedResource[untyped.Monitor](rest, "monitors", C, L, R, U, D)
	rest.Nics = core.NewUntypedResource[untyped.Nic](rest, "nics", L, R)
	rest.NicPorts = core.NewUntypedResource[untyped.NicPort](rest, "nicports", L, R, U)
	rest.Nis = core.NewUntypedResource[untyped.Nis](rest, "nis", C, L, R, U, D)
	rest.Nvrams = core.NewUntypedResource[untyped.Nvram](rest, "nvrams", L, R, U, D)
	rest.Oidcs = core.NewUntypedResource[untyped.Oidc](rest, "oidcs", C, L, R, U, D)
	rest.Permissions = core.NewUntypedResource[untyped.Permissions](rest, "permissions", L, R)
	rest.Ports = core.NewUntypedResource[untyped.Port](rest, "ports", L, R)
	rest.Projections = core.NewUntypedResource[untyped.Projection](rest, "projections", C, L)
	rest.ProjectionColumns = core.NewUntypedResource[untyped.ProjectionColumn](rest, "projectioncolumns", L)
	rest.PrometheusMetrics = core.NewUntypedResource[untyped.PrometheusMetrics](rest, "prometheusmetrics", R)
	rest.ProtectedPaths = core.NewUntypedResource[untyped.ProtectedPath](rest, "protectedpaths", C, L, R, U, D)
	rest.ProtectionPolicies = core.NewUntypedResource[untyped.ProtectionPolicy](rest, "protectionpolicies", C, L, R, U, D)
	rest.Psus = core.NewUntypedResource[untyped.Psu](rest, "psus", L, R)
	rest.QosPolicies = core.NewUntypedResource[untyped.QosPolicy](rest, "qospolicies", C, L, R, U, D)
	rest.Quotas = core.NewUntypedResource[untyped.Quota](rest, "quotas", C, L, R, U, D)
	rest.QuotaEntityInfos = core.NewUntypedResource[untyped.QuotaEntityInfo](rest, "quotaentityinfo", L)
	rest.Racks = core.NewUntypedResource[untyped.Rack](rest, "racks", C, L, R, U, D)
	rest.Realms = core.NewUntypedResource[untyped.Realm](rest, "realms", C, L, R, U, D)
	rest.ReplicationPeers = core.NewUntypedResource[untyped.ReplicationPeers](rest, "nativereplicationremotetargets", C, L, R, U, D)
	rest.ReplicationPolicies = core.NewUntypedResource[untyped.ReplicationPolicy](rest, "replicationpolicies", C, L, R, U, D)
	rest.ReplicationRestorePoints = core.NewUntypedResource[untyped.ReplicationRestorePoint](rest, "replicationrestorepoints", L, R)
	rest.ReplicationStreams = core.NewUntypedResource[untyped.ReplicationStream](rest, "replicationstreams", C, L, R, U, D)
	rest.Roles = core.NewUntypedResource[untyped.Role](rest, "roles", C, L, R, U, D)
	rest.S3Keys = core.NewUntypedResource[untyped.S3Keys](rest, "s3keys", C, L)
	rest.S3LifeCycleRules = core.NewUntypedResource[untyped.S3LifeCycleRule](rest, "s3lifecyclerules", C, L, R, U, D)
	rest.S3Policies = core.NewUntypedResource[untyped.S3Policy](rest, "s3policies", C, L, R, U, D)
	rest.S3replicationPeers = core.NewUntypedResource[untyped.S3replicationPeers](rest, "replicationtargets", C, L, R, U, D)
	rest.Schemas = core.NewUntypedResource[untyped.Schema](rest, "schemas", C, L)
	rest.SettingDiffs = core.NewUntypedResource[untyped.SettingDiff](rest, "settingdiff", R)
	rest.Snapshots = core.NewUntypedResource[untyped.Snapshot](rest, "snapshots", C, L, R, U, D)
	rest.SnapshotPolicies = core.NewUntypedResource[untyped.SnapshotPolicy](rest, "snapshotpolicies", C, L, R, U, D)
	rest.Ssds = core.NewUntypedResource[untyped.Ssd](rest, "ssds", L, R, U, D)
	rest.SubnetManagers = core.NewUntypedResource[untyped.SubnetManager](rest, "subnetmanagers", C, L, R, U, D)
	rest.SupportBundles = core.NewUntypedResource[untyped.SupportBundles](rest, "supportbundles", C, L, R, U, D)
	rest.SupportedDrivers = core.NewUntypedResource[untyped.SupportedDrivers](rest, "supporteddrives", C, L, R, U, D)
	rest.Switches = core.NewUntypedResource[untyped.Switch](rest, "switches", C, L, R, U, D)
	rest.Tables = core.NewUntypedResource[untyped.Table](rest, "tables", C, L, U)
	rest.Tenants = core.NewUntypedResource[untyped.Tenant](rest, "tenants", C, L, R, U, D)
	rest.Topics = core.NewUntypedResource[untyped.Topic](rest, "topics", C, L, U)
	rest.Users = core.NewUntypedResource[untyped.User](rest, "users", C, L, R, U, D)
	rest.UserQuotas = core.NewUntypedResource[untyped.UserQuota](rest, "userquotas", C, L, R, U, D)
	rest.VastAuditLogs = core.NewUntypedResource[untyped.VastAuditLog](rest, "vastauditlog", C, L)
	rest.VastDb = core.NewUntypedResource[untyped.VastDb](rest, "vastdb")
	rest.Versions = core.NewUntypedResource[untyped.Version](rest, "versions", L, R)
	rest.Views = core.NewUntypedResource[untyped.View](rest, "views", C, L, R, U, D)
	rest.ViewPolicies = core.NewUntypedResource[untyped.ViewPolicy](rest, "viewpolicies", C, L, R, U, D)
	rest.Vips = core.NewUntypedResource[untyped.Vip](rest, "vips", L, R)
	rest.VipPools = core.NewUntypedResource[untyped.VipPool](rest, "vippools", C, L, R, U, D)
	rest.Vms = core.NewUntypedResource[untyped.Vms](rest, "vms", L, R, U)
	rest.Volumes = core.NewUntypedResource[untyped.Volume](rest, "volumes", C, L, R, U, D)
	rest.VpnTunnels = core.NewUntypedResource[untyped.VpnTunnel](rest, "vpntunnels", C, L, R, U, D)
	rest.VTasks = core.NewUntypedResource[untyped.VTask](rest, "vtasks", L, R, U)
	rest.WebHooks = core.NewUntypedResource[untyped.WebHook](rest, "webhooks", C, L, R, U, D)
	rest.VirtualMachines = core.NewUntypedResource[untyped.VirtualMachine](rest, "virtual-machines", L, R)
	rest.Hosts = core.NewUntypedResource[untyped.Host](rest, "hosts", L, R)
	rest.BlobExpansions = core.NewUntypedResource[untyped.BlobExpansion](rest, "blobexpansions", C)
	rest.ComputeClusters = core.NewUntypedResource[untyped.ComputeCluster](rest, "computeclusters", C, L, R, U, D)
	rest.EventBrokers = core.NewUntypedResource[untyped.EventBroker](rest, "eventbrokers", C, L, R, U, D)
	rest.OpenFiles = core.NewUntypedResource[untyped.OpenFile](rest, "openfiles", L, R)
	rest.OpenFileHandles = core.NewUntypedResource[untyped.OpenFileHandle](rest, "openfilehandles", L, R)
	rest.OpenFilesQueries = core.NewUntypedResource[untyped.OpenFilesQuery](rest, "openfilesqueries", C, L, R, D)
	rest.QuotaGroups = core.NewUntypedResource[untyped.QuotaGroup](rest, "quotagroups", C, L, R, U, D)
	rest.SupportBundlesQueue = core.NewUntypedResource[untyped.SupportBundlesQueue](rest, "supportbundlesqueue", L)
	rest.TlsCertificates = core.NewUntypedResource[untyped.TlsCertificate](rest, "tlscertificates", C, L, R, U, D)
	rest.VastdbTables = core.NewUntypedResource[untyped.VastdbTable](rest, "vastdbtable")

	// Nested DataEngine rest (serverless); shares session with this VMS rest.
	rest.DataEngine = dataengine.NewUntyped(rest.Session, rest.GetCtx())

	return rest, nil
}

func (rest *UntypedVMSRest) GetSession() core.RESTSession {
	return rest.Session
}

func (rest *UntypedVMSRest) GetResourceMap() map[string]core.ResourceEntry {
	return rest.resourceMap
}

func (rest *UntypedVMSRest) GetCtx() context.Context {
	return rest.ctx
}

func (rest *UntypedVMSRest) SetCtx(ctx context.Context) {
	rest.ctx = ctx
	// Nested rests keep their own ctx; keep them in sync with the parent.
	if rest.DataEngine != nil {
		rest.DataEngine.SetCtx(ctx)
	}
}

func (rest *UntypedVMSRest) GetApiRoot() string {
	return rest.apiRoot
}

// String returns a log-friendly identity of this client: VMS host and auth mode.
// Examples: "10.0.0.1 [type=api-token]", "vms.example.com [type=bearer-token;user=admin;tenant=foo]"
func (rest *UntypedVMSRest) String() string {
	cfg := rest.Session.GetConfig()
	var parts []string
	switch a := rest.Session.GetAuthenticator().(type) {
	case *core.ApiRTokenAuthenticator:
		parts = append(parts, "type=api-token")
		if a.Tenant != "" {
			parts = append(parts, "tenant="+a.Tenant)
		}
	case *core.BaseAuthAuthenticator:
		parts = append(parts, "type=basic-auth", "user="+a.Username)
		if a.Tenant != "" {
			parts = append(parts, "tenant="+a.Tenant)
		}
	case *core.JWTAuthenticator:
		parts = append(parts, "type=bearer-token", "user="+a.Username)
		if a.Tenant != "" {
			parts = append(parts, "tenant="+a.Tenant)
		}
	default:
		panic(fmt.Sprintf("UntypedVMSRest.String: unexpected authenticator type %T", rest.Session.GetAuthenticator()))
	}
	return fmt.Sprintf("%s [%s]", cfg.Host, strings.Join(parts, ";"))
}
