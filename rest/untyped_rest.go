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

	// +apiall:extraMethod:POST=/activedirectory/{id}/is_operation_healthy/
	// +apiall:extraMethod:PATCH=/activedirectory/{id}/refresh/
	// +apiall:extraMethod:GET=/activedirectory/{id}/domains/
	// +apiall:extraMethod:GET=/activedirectory/{id}/dcs/
	// +apiall:extraMethod:GET=/activedirectory/{id}/gcs/
	// +apiall:extraMethod:GET=/activedirectory/{id}/current_gc/
	// +apiall:extraMethod:POST=/activedirectory/{id}/change_machine_account_password/
	ActiveDirectories *untyped.ActiveDirectory
	// +apiall:extraMethod:PATCH=/alarms/clear/
	Alarms    *untyped.Alarm
	Analytics *untyped.Analytics
	// +apiall:extraMethod:PATCH=/apitokens/{id}/revoke/
	ApiTokens     *untyped.ApiToken
	BasicSettings *untyped.BasicSettings
	BGPConfigs    *untyped.BGPConfig
	// +apiall:extraMethod:GET|POST=/bigcatalogconfig/query_data/
	// +apiall:extraMethod:GET=/bigcatalogconfig/columns/
	// +apiall:extraMethod:GET=/bigcatalogconfig/stats/
	BigCatalogConfigs *untyped.BigCatalogConfig
	// +apiall:extraMethod:PATCH=/bigcatalogindexedcolumns/add/
	// +apiall:extraMethod:DELETE=/bigcatalogindexedcolumns/remove/
	BigCatalogIndexedColumns *untyped.BigCatalogIndexedColumns
	// +apiall:extraMethod:PATCH=/blockhosts/{id}/set_volumes/
	// +apiall:extraMethod:PATCH=/blockhosts/{id}/update_volumes/
	// +apiall:extraMethod:DELETE=/blockhosts/bulk/
	BlockHosts *untyped.BlockHost
	// +apiall:extraMethod:PATCH=/blockmappings/bulk/
	BlockHostMappings *untyped.BlockHostMapping
	// +apiall:extraMethod:PATCH=/callhomeconfigs/{id}/send/
	// +apiall:extraMethod:PATCH=/callhomeconfigs/{id}/register-cluster/
	CallhomeConfigs *untyped.CallhomeConfigs
	Capacities      *untyped.Capacity
	// +apiall:extraMethod:PATCH=/carriers/{id}/control_led/
	// +apiall:extraMethod:PATCH=/carriers/{id}/highlight/
	// +apiall:extraMethod:PATCH=/carriers/{id}/reset_pci/
	Carriers *untyped.Carrier
	// +apiall:extraMethod:PATCH=/cboxes/{id}/refresh_uid/
	// +apiall:extraMethod:PATCH=/cboxes/{id}/control_led/
	Cboxes          *untyped.Cbox
	Certificates    *untyped.Certificate
	ChallengeTokens *untyped.ChallengeTokens
	// +apiall:extraMethod:PATCH=/clusters/{id}/resume_deploy/
	// +apiall:extraMethod:PATCH=/clusters/{id}/set_password/
	// +apiall:extraMethod:DELETE=/clusters/{id}/celery_remove_queued_task/
	// +apiall:extraMethod:GET=/clusters/{id}/celery_queue/
	// +apiall:extraMethod:GET=/clusters/{id}/celery_scheduled/
	// +apiall:extraMethod:GET=/clusters/{id}/celery_reserved/
	// +apiall:extraMethod:GET=/clusters/{id}/celery_status/
	// +apiall:extraMethod:DELETE=/clusters/{id}/delete_folder/
	// +apiall:extraMethod:PATCH=/clusters/{id}/system_settings/
	// +apiall:extraMethod:PATCH=/clusters/{id}/rpc/
	// +apiall:extraMethod:POST=/clusters/{id}/upload_from_s3/
	// +apiall:extraMethod:POST=/clusters/{id}/upgrade_optane/
	// +apiall:extraMethod:POST=/clusters/{id}/upgrade_ssd/
	// +apiall:extraMethod:PATCH=/clusters/{id}/upgrade/
	// +apiall:extraMethod:POST=/clusters/{id}/locks/
	// +apiall:extraMethod:DELETE=/clusters/{id}/release_recursive_locks/
	// +apiall:extraMethod:POST=/clusters/{id}/expand/
	// +apiall:extraMethod:POST=/clusters/{id}/stop_upgrade/
	// +apiall:extraMethod:GET=/clusters/{id}/pre_upgrade_validation_exceptions/
	// +apiall:extraMethod:POST=/clusters/{id}/upgrade_without_file/
	// +apiall:extraMethod:POST=/clusters/{id}/notify_new_version/
	// +apiall:extraMethod:POST=/clusters/wipe/
	// +apiall:extraMethod:PATCH=/clusters/run_hardware_check/
	// +apiall:extraMethod:PATCH=/clusters/block_providers/
	// +apiall:extraMethod:GET|PATCH|DELETE=/clusters/{id}/vsettings/
	// +apiall:extraMethod:GET=/clusters/list_smb_client_connections/
	// +apiall:extraMethod:GET=/clusters/list_smb_open_files/
	// +apiall:extraMethod:GET=/clusters/list_open_protocol_handles/
	// +apiall:extraMethod:GET=/clusters/{id}/advanced/
	// +apiall:extraMethod:GET|PATCH=/clusters/{id}/auditing/
	// +apiall:extraMethod:GET|PATCH=/clusters/{id}/vast_db/
	// +apiall:extraMethod:GET=/clusters/get_snapshoted_paths/
	// +apiall:extraMethod:DELETE=/clusters/close_protocol_handle/
	// +apiall:extraMethod:POST=/clusters/shard_expand/
	// +apiall:extraMethod:POST=/clusters/dbox_migration/
	// +apiall:extraMethod:GET|PATCH=/clusters/dbox_migration_update_source_target/
	// +apiall:extraMethod:GET=/clusters/dbox_migration_status/
	// +apiall:extraMethod:GET=/clusters/dbox_migration_validate/
	// +apiall:extraMethod:GET=/clusters/dbox_migration_validate_state/
	// +apiall:extraMethod:GET=/clusters/dboxes_total_capacity/
	// +apiall:extraMethod:GET=/clusters/list_tenants_remote/
	// +apiall:extraMethod:GET=/clusters/list_snapshoted_paths_remote/
	// +apiall:extraMethod:GET=/clusters/list_clone_snapshoted_paths_remote/
	// +apiall:extraMethod:GET=/clusters/get_shard_expansion_status/
	// +apiall:extraMethod:PATCH=/clusters/add_boxes/
	// +apiall:extraMethod:POST=/clusters/rotate_master_encryption_group_key/
	// +apiall:extraMethod:GET=/clusters/list_prefetch_paths_info/
	// +apiall:extraMethod:POST=/clusters/{id}/set_certificates/
	// +apiall:extraMethod:POST=/clusters/{id}/generate_unfreeze_token/
	// +apiall:extraMethod:GET=/clusters/bgp_table/
	// +apiall:extraMethod:POST=/clusters/{id}/unfreeze/
	// +apiall:extraMethod:POST=/clusters/{id}/set_drive_fw_upgrade/
	// +apiall:extraMethod:POST=/clusters/{id}/add_ekm/
	Clusters *untyped.Cluster
	// +apiall:extraMethod:POST=/cnodes/add_cnodes/
	// +apiall:extraMethod:POST=/cnodes/set_tenants/
	// +apiall:extraMethod:PATCH=/cnodes/{id}/control_led/
	// +apiall:extraMethod:PATCH=/cnodes/{id}/highlight/
	// +apiall:extraMethod:PATCH=/cnodes/{id}/rename/
	// +apiall:extraMethod:GET|PATCH=/cnodes/{id}/bgpconfig
	Cnodes      *untyped.Cnode
	CnodeGroups *untyped.CnodeGroup
	// +apiall:extraMethod:GET=/columns/show/
	// +apiall:extraMethod:DELETE=/columns/delete/
	// +apiall:extraMethod:PATCH=/columns/rename/
	Columns *untyped.Column
	// +apiall:extraMethod:POST=/config/reset/
	Configs *untyped.Config
	// +apiall:extraMethod:POST=/dboxes/add/
	// +apiall:extraMethod:PATCH=/dboxes/{id}/control_led/
	// +apiall:extraMethod:PATCH=/dboxes/{id}/reset_dp_i2c/
	Dboxes *untyped.Dbox
	// +apiall:extraMethod:GET|PATCH=/delta/config/
	Deltas *untyped.Delta
	// +apiall:extraMethod:PATCH=/dnodes/{id}/control_led/
	// +apiall:extraMethod:PATCH=/dnodes/{id}/highlight/
	// +apiall:extraMethod:PATCH=/dnodes/{id}/rename/
	Dnodes *untyped.Dnode
	Dns    *untyped.Dns
	// +apiall:extraMethod:PATCH=/dtrays/{id}/control_led/
	// +apiall:extraMethod:PATCH=/dtrays/{id}/rename/
	Dtrays *untyped.Dtray
	// +apiall:extraMethod:POST=/eboxes/add/
	// +apiall:extraMethod:PATCH=/eboxes/{id}/control_led/
	Eboxes         *untyped.Ebox
	EncryptedPaths *untyped.EncryptedPath
	// +apiall:extraMethod:POST=/encryptiongroups/{id}/revoke_encryption_group/
	// +apiall:extraMethod:POST=/encryptiongroups/{id}/deactivate_encryption_group/
	// +apiall:extraMethod:POST=/encryptiongroups/{id}/reinstate_encryption_group/
	// +apiall:extraMethod:POST=/encryptiongroups/{id}/rotate_encryption_group_key/
	EncryptionGroups *untyped.EncryptionGroup
	Envs             *untyped.Env
	Events           *untyped.Event
	// +apiall:extraMethod:PATCH=/eventdefinitions/{id}/test/
	EventDefinitions *untyped.EventDefinition
	// +apiall:extraMethod:PATCH=/eventdefinitionconfigs/{id}/test/
	EventDefinitionConfigs *untyped.EventDefinitionConfig
	Fans                   *untyped.Fan
	// +apiall:extraMethod:POST=/folders/create_folder/
	// +apiall:extraMethod:PATCH=/folders/modify_folder/
	// +apiall:extraMethod:DELETE=/folders/delete_folder/
	// +apiall:extraMethod:POST=/folders/stat_path/
	// +apiall:extraMethod:POST|GET|DELETE=/folders/read_only/
	Folders *untyped.Folder
	// +apiall:extraMethod:POST=/filesystem/clone/
	Filesystems *untyped.Filesystem
	// +apiall:extraMethod:PATCH=/globalsnapstreams/{id}/pause/
	// +apiall:extraMethod:PATCH=/globalsnapstreams/{id}/resume/
	// +apiall:extraMethod:PATCH=/globalsnapstreams/{id}/stop/
	GlobalSnapshotStreams *untyped.GlobalSnapshotStream
	// +apiall:extraMethod:GET|PATCH=/groups/query/
	// +apiall:extraMethod:GET=/groups/names/
	Groups *untyped.Group
	// +apiall:extraMethod:GET=/iamroles/{id}/credentials/
	// +apiall:extraMethod:PATCH=/iamroles/{id}/revoke_access_keys/
	IamRoles   *untyped.IamRole
	Injections *untyped.Injections
	// +apiall:extraMethod:PATCH=/indestructibility/{id}/generate_token/
	// +apiall:extraMethod:PATCH=/indestructibility/{id}/unlock/
	// +apiall:extraMethod:PATCH=/indestructibility/{id}/reset_passwd/
	Indestructibility *untyped.Indestructibility
	IoData            *untyped.IoData
	// +apiall:extraMethod:GET=/kafkabrokers/{id}/list_topics/
	KafkaBrokers *untyped.KafkaBroker
	// +apiall:extraMethod:POST=/kerberos/{id}/keytab/
	Kerberos *untyped.Kerberos
	// +apiall:extraMethod:PATCH=/ldaps/{id}/set_posix_primary/
	Ldaps               *untyped.Ldap
	Licenses            *untyped.License
	LocalProviders      *untyped.LocalProvider
	LocalS3Keys         *untyped.LocalS3Key
	ManagedApplications *untyped.ManageApplications
	// +apiall:extraMethod:PATCH=/managers/{id}/unlock/
	// +apiall:extraMethod:PATCH=/managers/password/
	// +apiall:extraMethod:GET=/managers/authorized_status/
	Managers *untyped.Manager
	Metrics  *untyped.Metrics
	Modules  *untyped.Module
	// +apiall:extraMethod:GET=/monitors/topn/
	// +apiall:extraMethod:GET=/monitors/{id}/query/
	Monitors *untyped.Monitor
	Nics     *untyped.Nic
	// +apiall:extraMethod:GET=/nicports/{id}/related_nicports/
	NicPorts *untyped.NicPort
	// +apiall:extraMethod:PATCH=/nis/{id}/set_posix_primary/
	// +apiall:extraMethod:PATCH=/nis/refresh/
	Nis *untyped.Nis
	// +apiall:extraMethod:PATCH=/nvrams/{id}/format/
	// +apiall:extraMethod:PATCH=/nvrams/{id}/control_led/
	Nvrams *untyped.Nvram
	// +apiall:extraMethod:PATCH=/oidcs/{id}/refresh_keys/
	Oidcs *untyped.Oidc
	// +apiall:extraMethod:GET=/permissions/objects/
	Permissions *untyped.Permissions
	Ports       *untyped.Port
	// +apiall:extraMethod:GET=/projections/show/
	// +apiall:extraMethod:PATCH=/projections/rename/
	// +apiall:extraMethod:DELETE=/projections/delete/
	Projections *untyped.Projection
	// +apiall:extraMethod:GET=/projectioncolumns/show/
	ProjectionColumns *untyped.ProjectionColumn
	// +apiall:extraMethod:GET=/prometheusmetrics/users/
	// +apiall:extraMethod:GET=/prometheusmetrics/defrag/
	// +apiall:extraMethod:GET=/prometheusmetrics/views/
	// +apiall:extraMethod:GET=/prometheusmetrics/devices/
	// +apiall:extraMethod:GET=/prometheusmetrics/quotas/
	// +apiall:extraMethod:GET=/prometheusmetrics/all/
	// +apiall:extraMethod:GET=/prometheusmetrics/switches/
	PrometheusMetrics *untyped.PrometheusMetrics
	// +apiall:extraMethod:POST=/protectedpaths/{id}/restore/
	// +apiall:extraMethod:PATCH=/protectedpaths/{id}/commit/
	// +apiall:extraMethod:PATCH=/protectedpaths/{id}/pause/
	// +apiall:extraMethod:PATCH=/protectedpaths/{id}/resume/
	// +apiall:extraMethod:PATCH=/protectedpaths/{id}/stop/
	// +apiall:extraMethod:PATCH=/protectedpaths/{id}/force_failover/
	// +apiall:extraMethod:PATCH=/protectedpaths/{id}/add_stream/
	// +apiall:extraMethod:PATCH=/protectedpaths/{id}/remove_stream/
	// +apiall:extraMethod:PATCH=/protectedpaths/{id}/reattach_stream/
	// +apiall:extraMethod:PATCH=/protectedpaths/{id}/modify_member/
	// +apiall:extraMethod:GET=/protectedpaths/{id}/validate/
	// +apiall:extraMethod:GET|POST|DELETE=/protectedpaths/{id}/prefetch_path/
	// +apiall:extraMethod:PATCH=/protectedpaths/{id}/replicate_now/
	ProtectedPaths     *untyped.ProtectedPath
	ProtectionPolicies *untyped.ProtectionPolicy
	Psus               *untyped.Psu
	QosPolicies        *untyped.QosPolicy
	// +apiall:extraMethod:PATCH=/quotas/{id}/refresh_user_quotas/
	// +apiall:extraMethod:PATCH=/quotas/recalc/
	// +apiall:extraMethod:PATCH=/quotas/recalc_stop/
	// +apiall:extraMethod:PATCH=/quotas/{id}/reset_grace_period/
	Quotas           *untyped.Quota
	QuotaEntityInfos *untyped.QuotaEntityInfo
	// +apiall:extraMethod:PATCH=/racks/{id}/rename/
	// +apiall:extraMethod:POST=/racks/{id}/add_boxes/
	// +apiall:extraMethod:PATCH=/racks/{id}/control_led/
	// +apiall:extraMethod:POST=/racks/{id}/bgpconfig/
	Racks *untyped.Rack
	// +apiall:extraMethod:PATCH=/realms/{id}/assign/
	// +apiall:extraMethod:PATCH=/realms/{id}/unassign/
	Realms *untyped.Realm
	// +apiall:extraMethod:GET=/nativereplicationremotetargets/get_remote_mapping/
	ReplicationPeers         *untyped.ReplicationPeers
	ReplicationPolicies      *untyped.ReplicationPolicy
	ReplicationRestorePoints *untyped.ReplicationRestorePoint
	ReplicationStreams       *untyped.ReplicationStream
	Roles                    *untyped.Role
	// +apiall:extraMethod:DELETE=/s3keys/{access_key}/
	S3Keys *untyped.S3Keys
	// +apiall:extraMethod:GET=/s3lifecyclerules/get_object_expiration/
	S3LifeCycleRules   *untyped.S3LifeCycleRule
	S3Policies         *untyped.S3Policy
	S3replicationPeers *untyped.S3replicationPeers
	// +apiall:extraMethod:GET=/schemas/show/
	// +apiall:extraMethod:PATCH=/schemas/rename/
	// +apiall:extraMethod:DELETE=/schemas/delete/
	Schemas      *untyped.Schema
	SettingDiffs *untyped.SettingDiff
	// +apiall:extraMethod:POST=/snapshots/{id}/clone/
	Snapshots        *untyped.Snapshot
	SnapshotPolicies *untyped.SnapshotPolicy
	// +apiall:extraMethod:PATCH=/ssds/{id}/format/
	// +apiall:extraMethod:PATCH=/ssds/{id}/control_led/
	Ssds           *untyped.Ssd
	SubnetManagers *untyped.SubnetManager
	// +apiall:extraMethod:PATCH=/supportbundles/{id}/upload/
	// +apiall:extraMethod:GET=/supportbundles/{id}/download/
	SupportBundles   *untyped.SupportBundles
	SupportedDrivers *untyped.SupportedDrivers
	Switches         *untyped.Switch
	// +apiall:extraMethod:GET=/tables/show/
	// +apiall:extraMethod:PATCH=/tables/rename/
	// +apiall:extraMethod:DELETE=/tables/delete/
	// +apiall:extraMethod:PATCH=/tables/add_columns/
	Tables *untyped.Table
	// +apiall:extraMethod:POST=/tenants/{id}/is_operation_healthy/
	// +apiall:extraMethod:POST=/tenants/{id}/revoke_encryption_group/
	// +apiall:extraMethod:POST=/tenants/{id}/deactivate_encryption_group/
	// +apiall:extraMethod:POST=/tenants/{id}/reinstate_encryption_group/
	// +apiall:extraMethod:POST=/tenants/{id}/rotate_encryption_group_key/
	// +apiall:extraMethod:GET=/tenants/{id}/same_encryption_group_tenants/
	// +apiall:extraMethod:GET=/tenants/configured_idp/
	// +apiall:extraMethod:GET=/tenants/remote_objects/
	// +apiall:extraMethod:GET=/tenants/{id}/vippool_ip_ranges/
	// +apiall:extraMethod:PATCH=/tenants/{id}/client_ip_ranges/
	// +apiall:extraMethod:GET|PATCH=/tenants/{id}/client_metrics/
	// +apiall:extraMethod:GET=/tenants/{id}/nfs4_delegs/
	// +apiall:extraMethod:DELETE=/tenants/{id}/nfs4_deleg/
	Tenants *untyped.Tenant
	// +apiall:extraMethod:GET|POST|PATCH=/topics/
	// +apiall:extraMethod:GET=/topics/show/
	// +apiall:extraMethod:DELETE=/topics/delete/
	Topics *untyped.Topic
	// +apiall:extraMethod:PATCH|GET=/users/{id}/tenant_data/
	// +apiall:extraMethod:POST|PATCH|DELETE=/users/{id}/access_keys/
	// +apiall:extraMethod:GET|PATCH=/users/query/
	// +apiall:extraMethod:GET=/users/names/
	// +apiall:extraMethod:PATCH=/users/refresh/
	// +apiall:extraMethod:POST=/users/copy/
	// +apiall:extraMethod:POST|PATCH|DELETE=/users/non_local_keys/
	Users      *untyped.User
	UserQuotas *untyped.UserQuota
	// +apiall:extraMethod:GET=/vastauditlog/query_data/
	// +apiall:extraMethod:GET=/vastauditlog/columns/
	// +apiall:extraMethod:GET=/vastauditlog/stats/
	VastAuditLogs *untyped.VastAuditLog
	// +apiall:extraMethod:GET=/vastdb/vips/
	VastDb   *untyped.VastDb
	Versions *untyped.Version
	// +apiall:extraMethod:GET=/views/list_open_smb_handles/
	// +apiall:extraMethod:DELETE=/views/close_smb_handle/
	// +apiall:extraMethod:GET=/views/list_seamless_peers/
	// +apiall:extraMethod:DELETE|POST=/views/{id}/permissions_repair/
	// +apiall:extraMethod:POST=/views/{id}/check_permissions_templates/
	// +apiall:extraMethod:GET|PATCH=/views/{id}/legal_hold/
	Views *untyped.View
	// +apiall:extraMethod:PATCH=/viewpolicies/{id}/refresh_netgroups/
	// +apiall:extraMethod:POST|DELETE=/viewpolicies/{id}/remote_mapping/
	ViewPolicies *untyped.ViewPolicy
	Vips         *untyped.Vip
	VipPools     *untyped.VipPool
	// +apiall:extraMethod:PATCH=/vms/{id}/set_certificate/
	// +apiall:extraMethod:GET|PATCH=/vms/{id}/network_settings/
	// +apiall:extraMethod:POST=/vms/{id}/network_settings_summary/
	// +apiall:extraMethod:PATCH=/vms/{id}/set_client_certificate/
	// +apiall:extraMethod:GET=/vms/{id}/configured_idps/
	// +apiall:extraMethod:GET|PATCH|POST|DELETE=/vms/{id}/saml_config/
	// +apiall:extraMethod:PATCH=/vms/{id}/reset_certificate/
	// +apiall:extraMethod:PATCH=/vms/{id}/remove_client_certificate/
	// +apiall:extraMethod:PATCH=/vms/{id}/set_ssl_ciphers/
	// +apiall:extraMethod:PATCH=/vms/{id}/reset_ssl_ciphers/
	// +apiall:extraMethod:PATCH=/vms/{id}/set_ssl_port/
	// +apiall:extraMethod:GET=/vms/{id}/login_banner/
	// +apiall:extraMethod:PATCH=/vms/{id}/toggle_maintenance_mode/
	// +apiall:extraMethod:GET|PATCH=/vms/{id}/pwd_settings/
	// +apiall:extraMethod:PATCH=/vms/{id}/set_max_api_tokens_per_user/
	Vms *untyped.Vms
	// +apiall:extraMethod:PATCH=/volumes/{id}/set_hosts/
	// +apiall:extraMethod:PATCH=/volumes/{id}/update_hosts/
	// +apiall:extraMethod:GET=/volumes/{id}/get_snapshots/
	// +apiall:extraMethod:GET=/volumes/{id}/fetch_capacity/
	// +apiall:extraMethod:DELETE=/volumes/bulk/
	Volumes *untyped.Volume
	// +apiall:extraMethod:DELETE=/vpntunnels/delete_all/
	VpnTunnels *untyped.VpnTunnel
	// +apiall:extraMethod:PATCH=/vtasks/{id}/retry/
	VTasks   *untyped.VTask
	WebHooks *untyped.WebHook
	// +apiall:extraMethod:GET=/hosts/discovered_hosts/
	// +apiall:extraMethod:GET=/hosts/discover/
	Hosts           *untyped.Host
	VirtualMachines *untyped.VirtualMachine
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

	// Set context: use provided context or default to background context
	if config.Context != nil {
		rest.SetCtx(config.Context)
	} else {
		rest.SetCtx(context.Background())
	}

	// Fill in each resource, pointing back to the same rest
	rest.ActiveDirectories = newUntypedResource[untyped.ActiveDirectory](rest, "activedirectory", C, L, R, U, D)
	rest.Alarms = newUntypedResource[untyped.Alarm](rest, "alarms", L, R, U, D)
	rest.Analytics = newUntypedResource[untyped.Analytics](rest, "analytics", L)
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
	rest.SnapshotPolicies = newUntypedResource[untyped.SnapshotPolicy](rest, "snapshots", C, L, R, U, D)
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
