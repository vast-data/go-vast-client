package typed

import (
	"context"

	vast_client "github.com/vast-data/go-vast-client"
)

// -----------------------------------------------------
// SEARCH PARAMS
// -----------------------------------------------------

// ViewSearchParams represents the search parameters for View operations
// Generated from GET query parameters for resource: views
type ViewSearchParams struct {
	Alias string `json:"alias,omitempty" yaml:"alias,omitempty" required:"false" doc:"Filter by NFS export alias"`
	Bucket string `json:"bucket,omitempty" yaml:"bucket,omitempty" required:"false" doc:"Limit response by S3 bucket name"`
	ClusterId string `json:"cluster__id,omitempty" yaml:"cluster__id,omitempty" required:"false" doc:"Limit response by cluster ID"`
	ClusterName string `json:"cluster__name,omitempty" yaml:"cluster__name,omitempty" required:"false" doc:"Filter response by cluster name."`
	Name string `json:"name,omitempty" yaml:"name,omitempty" required:"false" doc:"Filter by View name"`
	Nqn string `json:"nqn,omitempty" yaml:"nqn,omitempty" required:"false" doc:"NVMe Qualified Name to filter by."`
	Path string `json:"path,omitempty" yaml:"path,omitempty" required:"false" doc:"Filter by Element Store path"`
	PolicyId string `json:"policy__id,omitempty" yaml:"policy__id,omitempty" required:"false" doc:"Filter by view policy ID"`
	PolicyName string `json:"policy__name,omitempty" yaml:"policy__name,omitempty" required:"false" doc:"Filter by view policy name"`
	Share string `json:"share,omitempty" yaml:"share,omitempty" required:"false" doc:"Filter by share name"`
	TenantId int64 `json:"tenant_id,omitempty" yaml:"tenant_id,omitempty" required:"false" doc:"Filter by tenant. Specify tenant ID."`
	TenantNameIcontains string `json:"tenant_name__icontains,omitempty" yaml:"tenant_name__icontains,omitempty" required:"false" doc:"Tenant name to filter by"`
	
}

// -----------------------------------------------------
// REQUEST BODY
// -----------------------------------------------------

// ViewRequestBody represents the request body for View operations
// Generated from POST request body for resource: views
type ViewRequestBody struct {
	CreateDir bool `json:"create_dir,omitempty" yaml:"create_dir,omitempty" required:"true" doc:"Create a directory at the specified path. Set to true if the specified path does not exist."`
	Path string `json:"path,omitempty" yaml:"path,omitempty" required:"true" doc:"The full Element Store path to from the top level of the storage system on the cluster to the location that you want to expose. Begin with '/'. Do not include a trailing slash."`
	PolicyId int64 `json:"policy_id,omitempty" yaml:"policy_id,omitempty" required:"true" doc:"Every view must be attached to one view policy, which specifies further configurations. Specify by view policy ID which view policy should be used for the view."`
	AbeMaxDepth int64 `json:"abe_max_depth,omitempty" yaml:"abe_max_depth,omitempty" required:"false" doc:"Restricts ABE to a specified path depth. For example, if max depth is 3, ABE does not affect paths deeper than three levels. If not specified, ABE affects all path depths."`
	AbeProtocols []string `json:"abe_protocols,omitempty" yaml:"abe_protocols,omitempty" required:"false" doc:"The protocols for which Access-Based Enumeration (ABE) is enabled"`
	Alias string `json:"alias,omitempty" yaml:"alias,omitempty" required:"false" doc:"Relevant if NFS is included in the protocols array. An alias for the mount path of an NFSv3 export. The alias must begin with a forward slash ('/') and must consist of only ASCII characters. If specified, the alias that can be used by NFSv3 clients to mount the view."`
	AllowAnonymousAccess bool `json:"allow_anonymous_access,omitempty" yaml:"allow_anonymous_access,omitempty" required:"false" doc:"not in use"`
	AllowS3AnonymousAccess bool `json:"allow_s3_anonymous_access,omitempty" yaml:"allow_s3_anonymous_access,omitempty" required:"false" doc:"Allow S3 anonymous access to S3 bucket. If true, anonymous requests are granted provided that the object ACL grants access to the All Users group (in S3 Native security flavor) or the permission mode bits on the requested file and directory path grant access permission to 'others' (in NFS security flavor)."`
	AutoCommit string `json:"auto_commit,omitempty" yaml:"auto_commit,omitempty" required:"false" doc:"Applicable if locking is enabled. Sets the auto-commit time for files that are locked automatically. These files are locked automatically after the auto-commit period elapses from the time the file is saved. Files locked automatically are locked for the default-retention-period, after which they are unlocked. Specify as an integer value followed by a letter for the unit (h - hours, d - days, y - years). Example: 2h (2 hours)."`
	Bucket string `json:"bucket,omitempty" yaml:"bucket,omitempty" required:"false" doc:"A name for the S3 bucket name. Must be specified if S3 bucket is specified in protocols."`
	BucketCreators []string `json:"bucket_creators,omitempty" yaml:"bucket_creators,omitempty" required:"false" doc:"For S3 endpoint views, specify a list of users, by user name, whose bucket create requests use this view. Any request to create an S3 bucket that is sent by S3 API by a specified user will use this S3 Endpoint view. Users should not be specified as bucket creators in more than one S3 Endpoint view. Naming a user as a bucket creator in two S3 Endpoint views will fail the creation of the view with an error."`
	BucketCreatorsGroups []string `json:"bucket_creators_groups,omitempty" yaml:"bucket_creators_groups,omitempty" required:"false" doc:"For S3 endpoint views, specify a list of groups, by group name, whose bucket create requests use this view. Any request to create an S3 bucket that is sent by S3 API by a user who belongs to a group listed here will use this S3 Endpoint view. Take extra care not to duplicate bucket creators through groups: If you specify a group as a bucket creator group in one view and you also specify a user who belongs to that group as a bucket creator user in another view, view creation will not fail. Yet, there is a conflict between the two configurations and the selection of a view for configuring the user's buckets is not predictable."`
	BucketLogging ViewsRequestBody_BucketLogging `json:"bucket_logging,omitempty" yaml:"bucket_logging,omitempty" required:"false" doc:""`
	BucketOwner string `json:"bucket_owner,omitempty" yaml:"bucket_owner,omitempty" required:"false" doc:"Specifies a user to be the bucket owner. Specify as user name. Must be specified if S3 Bucket is included in protocols."`
	ClusterId int64 `json:"cluster_id,omitempty" yaml:"cluster_id,omitempty" required:"false" doc:"Cluster ID"`
	CreateDirAcl []ViewsRequestBody_CreateDirAclItem `json:"create_dir_acl,omitempty" yaml:"create_dir_acl,omitempty" required:"false" doc:"Define ACL for the newly created dir"`
	CreateDirMode int64 `json:"create_dir_mode,omitempty" yaml:"create_dir_mode,omitempty" required:"false" doc:"Unix permissions mode for the new dir"`
	DefaultRetentionPeriod string `json:"default_retention_period,omitempty" yaml:"default_retention_period,omitempty" required:"false" doc:"Relevant if locking is enabled. Required if s3_locks_retention_mode is set to governance or compliance. Specifies a default retention period for objects in the bucket. If set, object versions that are placed in the bucket are automatically protected with the specified retention lock. Otherwise, by default, each object version has no automatic protection but can be configured with a retention period or legal hold. Specify as an integer followed by h for hours, d for days, m for months, or y for years. For example: 2d or 1y."`
	FilesRetentionMode string `json:"files_retention_mode,omitempty" yaml:"files_retention_mode,omitempty" required:"false" doc:"Applicable if locking is enabled. The retention mode for new files. For views enabled for NFSv3 or SMB, if locking is enabled, files_retention_mode must be set to GOVERNANCE or COMPLIANCE. If the view is enabled for S3 and not for NFSv3 or SMB, files_retention_mode can be set to NONE. If GOVERNANCE, locked files cannot be deleted or changed. The Retention settings can be shortened or extended by users with sufficient permissions. If COMPLIANCE, locked files cannot be deleted or changed. Retention settings can be extended, but not shortened, by users with sufficient permissions. If NONE (S3 only), the retention mode is not set for the view; it is set individually for each object."`
	IndestructibleObjectDuration int64 `json:"indestructible_object_duration,omitempty" yaml:"indestructible_object_duration,omitempty" required:"false" doc:"Retention period for objects, in days. Each object in the bucket is protected from deletion, overwriting, renaming and metadata changes for the specified number of days after its creation date."`
	InheritAcl bool `json:"inherit_acl,omitempty" yaml:"inherit_acl,omitempty" required:"false" doc:"Indicates whether the directory should inherit ACLs from its parent directory"`
	IsDefaultSubsystem bool `json:"is_default_subsystem,omitempty" yaml:"is_default_subsystem,omitempty" required:"false" doc:"Set to true to set view to be the default subsystem for block storage. There can be up to one default subsystem per tenant. The default subsystem is the default view selected when creating a block volume if no view is specified."`
	IsIndestructibleObjectEnabled bool `json:"is_indestructible_object_enabled,omitempty" yaml:"is_indestructible_object_enabled,omitempty" required:"false" doc:"Set to true to enable indestructible object mode on the view. This is supported only if S3 is the only specified protocol. Other limitations also apply."`
	IsSeamless bool `json:"is_seamless,omitempty" yaml:"is_seamless,omitempty" required:"false" doc:"Supports seamless failover between replication peers by syncing file handles between the view and remote views on the replicated path on replication peers. This enables NFSv3 client users to retain the same mount point to the view in the event of a failover of the view path to a replication peer. This feature enables NFSv3 client users to retain the same mount point to the view in the event of a failover of the view path to a replication peer. Enabling this option may cause overhead and should only be enabled when the use case is relevant. To complete the configuration for seamless failover between any two peers, a seamless view must be created on each peer."`
	KafkaFirstJoinGroupTimeoutSec int64 `json:"kafka_first_join_group_timeout_sec,omitempty" yaml:"kafka_first_join_group_timeout_sec,omitempty" required:"false" doc:"Kafka first join group timeout, in seconds"`
	KafkaRejoinGroupTimeoutSec int64 `json:"kafka_rejoin_group_timeout_sec,omitempty" yaml:"kafka_rejoin_group_timeout_sec,omitempty" required:"false" doc:"Kafka rejoin group timeout, in seconds"`
	KafkaVipPools []int64 `json:"kafka_vip_pools,omitempty" yaml:"kafka_vip_pools,omitempty" required:"false" doc:"For Kafka-enabled views, an array of IDs of Virtual IP pools used to access event topics exposed by the view. The specified virtual IP pool must belong to the same tenant as the Kafka-enabled view. Must also not be a virtual IP pool that is excluded by the view policy's virtual IP pool association."`
	Locking bool `json:"locking,omitempty" yaml:"locking,omitempty" required:"false" doc:"Set to true to enable object locking on a view. Object locking cannot be disabled after the view is created. Must be true if s3_versioning is true."`
	MaxRetentionPeriod string `json:"max_retention_period,omitempty" yaml:"max_retention_period,omitempty" required:"false" doc:"Applicable if locking is enabled. Sets a maximum retention period for files that are locked in the view. Files cannot be locked for longer than this period, whether they are locked manually (by setting the atime) or automatically, using auto-commit. Specify as an integer value followed by a letter for the unit (m - minutes, h - hours, d - days, y - years). Example: 2y (2 years)."`
	MinRetentionPeriod string `json:"min_retention_period,omitempty" yaml:"min_retention_period,omitempty" required:"false" doc:"Applicable if locking is enabled. Sets a minimum retention period for files that are locked in the view. Files cannot be locked for less than this period, whether locked manually (by setting the atime) or automatically, using auto-commit. Specify as an integer value followed by a letter for the unit (h - hours, d - days, m - months, y - years). Example: 1d (1 day)."`
	Name string `json:"name,omitempty" yaml:"name,omitempty" required:"false" doc:"A name for the view"`
	Owner string `json:"owner,omitempty" yaml:"owner,omitempty" required:"false" doc:"The owner of the folder. Specify the owner using the attribute type set by owner_type. You can specify a group as the owner, as supported by SMB. To enable setting a group as the owner, set owner_is_group=true. In all cases, set owning_group also."`
	OwnerIsGroup bool `json:"owner_is_group,omitempty" yaml:"owner_is_group,omitempty" required:"false" doc:"Set to true if passing a group as the owner of the folder. This feature is used to enable setting a group as the owner, as supported by SMB."`
	OwnerType string `json:"owner_type,omitempty" yaml:"owner_type,omitempty" required:"false" doc:"The type of attribute used to specify owner."`
	OwningGroup string `json:"owning_group,omitempty" yaml:"owning_group,omitempty" required:"false" doc:"The owning group of the folder."`
	OwningGroupType string `json:"owning_group_type,omitempty" yaml:"owning_group_type,omitempty" required:"false" doc:"The type of attribute to use to specify the owning group of the folder."`
	Protocols []string `json:"protocols,omitempty" yaml:"protocols,omitempty" required:"false" doc:"Protocols enabled for access to the view. 'NFS' enables access from NFS version 3, 'NFS4' enables access from NFS version 4.1 and 4.2, S3' creates an S3 bucket on the view, 'ENDPOINT' creates an S3 endpoint, used as template for views created via S3 RPCs, DATABASE exposes the view as a VAST database. KAFKA enables events related to elements on the view path to be published to the VAST Event Broker. BLOCK exposes the view as a block storage subsystem."`
	QosPolicy string `json:"qos_policy,omitempty" yaml:"qos_policy,omitempty" required:"false" doc:"QoS Policy"`
	QosPolicyId int64 `json:"qos_policy_id,omitempty" yaml:"qos_policy_id,omitempty" required:"false" doc:"Associates a QoS policy with the view."`
	S3LocksRetentionMode string `json:"s3_locks_retention_mode,omitempty" yaml:"s3_locks_retention_mode,omitempty" required:"false" doc:"The retention mode for new object versions stored in this bucket. You can override this if you upload a new object version with an explicit retention mode and period."`
	S3ObjectOwnershipRule string `json:"s3_object_ownership_rule,omitempty" yaml:"s3_object_ownership_rule,omitempty" required:"false" doc:""`
	S3UnverifiedLookup bool `json:"s3_unverified_lookup,omitempty" yaml:"s3_unverified_lookup,omitempty" required:"false" doc:"S3 Unverified Lookup"`
	S3Versioning bool `json:"s3_versioning,omitempty" yaml:"s3_versioning,omitempty" required:"false" doc:"Enable S3 Versioning if S3 bucket. Versioning cannot be disabled after the view is created."`
	SelectForLiveMonitoring bool `json:"select_for_live_monitoring,omitempty" yaml:"select_for_live_monitoring,omitempty" required:"false" doc:"Enables live monitoring on the view. Live monitoring can be enabled for up to ten views at one time. Analytics data for views is polled every 5 minutes by default and every 10 seconds with live monitoring."`
	Share string `json:"share,omitempty" yaml:"share,omitempty" required:"false" doc:"SMB share name. Must be specified if SMB is specified in protocols."`
	ShareAcl ViewsRequestBody_ShareAcl `json:"share_acl,omitempty" yaml:"share_acl,omitempty" required:"false" doc:"Share-level ACL details"`
	TenantId int64 `json:"tenant_id,omitempty" yaml:"tenant_id,omitempty" required:"false" doc:"Associates the specified tenant with the view."`
	UserImpersonation ViewsRequestBody_UserImpersonation `json:"user_impersonation,omitempty" yaml:"user_impersonation,omitempty" required:"false" doc:""`
	
}

// -----------------------------------------------------
// RESPONSE BODY
// -----------------------------------------------------


// ViewsRequestBody_BucketLogging represents a nested type for response body
type ViewsRequestBody_BucketLogging struct {
	DestinationId int64 `json:"destination_id,omitempty" yaml:"destination_id,omitempty" required:"true" doc:"Specifies a view ID as the destination bucket for S3 bucket logging. The specified view must have the S3 bucket protocol enabled, must be on the same tenant as the view itself (the source view), must have the same bucket owner, and cannot be the same view as the source view. It also must not have S3 object locking enabled.  In bucket logging, a log entry is created in AWS log format for each request made to the source bucket. The log entries are periodically uploaded to the destination bucket. Configuring destination_id enables S3 bucket logging for the view."`
	KeyFormat string `json:"key_format,omitempty" yaml:"key_format,omitempty" required:"false" doc:"The format for the S3 bucket logging object keys. SIMPLE_PREFIX=[DestinationPrefix][YYYY]-[MM]-[DD]-[hh]-[mm]-[ss]-[UniqueString], PARTITIONED_PREFIX_EVENT_TIME=[DestinationPrefix][SourceUsername]/[SourceBucket]/[YYYY]/[MM]/[DD]/[YYYY]-[MM]-[DD]-[hh]-[mm]-[ss]-[UniqueString] where the partitioning is done based on the time when the logged events occurred, PARTITIONED_PREFIX_DELIVERY_TIME=[DestinationPrefix][SourceUsername]/[SourceBucket]/[YYYY]/[MM]/[DD]/[YYYY]-[MM]-[DD]-[hh]-[mm]-[ss]-[UniqueString] where the partitioning is done based on the time when the log object has been delivered to the destination bucket. Default: SIMPLE_PREFIX"`
	Prefix string `json:"prefix,omitempty" yaml:"prefix,omitempty" required:"false" doc:"Specifies a prefix to be prepended to each key of a log object uploaded to the destination bucket. This prefix can be used to categorize log objects; for example, if you use the same destination bucket for multiple source buckets. The prefix can be up to 128 characters and must follow S3 object naming rules."`
	
}


// ViewsRequestBody_UserImpersonation represents a nested type for response body
type ViewsRequestBody_UserImpersonation struct {
	Enabled bool `json:"enabled,omitempty" yaml:"enabled,omitempty" required:"false" doc:"True if user impersonation is enabled"`
	Identifier string `json:"identifier,omitempty" yaml:"identifier,omitempty" required:"false" doc:"Identifier of the user to impersonate"`
	IdentifierType string `json:"identifier_type,omitempty" yaml:"identifier_type,omitempty" required:"false" doc:"The identifier type of the specified identifier."`
	LoginName string `json:"login_name,omitempty" yaml:"login_name,omitempty" required:"false" doc:"Full username of user to impersonate, including domain name"`
	Username string `json:"username,omitempty" yaml:"username,omitempty" required:"false" doc:"The username of the user to impersonate"`
	
}


// ViewsRequestBody_ShareAcl represents a nested type for response body
type ViewsRequestBody_ShareAcl struct {
	Acl []ViewsRequestBody_ShareAcl_AclItem `json:"acl,omitempty" yaml:"acl,omitempty" required:"false" doc:"Share-level ACL"`
	Enabled bool `json:"enabled,omitempty" yaml:"enabled,omitempty" required:"false" doc:"True if Share ACL is enabled on the view, otherwise False"`
	
}


// ViewsResponseBody_ShareAcl_AclItem represents a nested type for response body
type ViewsResponseBody_ShareAcl_AclItem struct {
	Fqdn string `json:"fqdn,omitempty" yaml:"fqdn,omitempty" required:"false" doc:"FQDN of the chosen grantee"`
	Grantee string `json:"grantee,omitempty" yaml:"grantee,omitempty" required:"false" doc:"grantee type"`
	Name string `json:"name,omitempty" yaml:"name,omitempty" required:"false" doc:"name of the chosen grantee"`
	Perm string `json:"perm,omitempty" yaml:"perm,omitempty" required:"false" doc:"Grantee’s permissions"`
	SidStr string `json:"sid_str,omitempty" yaml:"sid_str,omitempty" required:"false" doc:"grantee’s SID"`
	UidOrGid int64 `json:"uid_or_gid,omitempty" yaml:"uid_or_gid,omitempty" required:"false" doc:"grantee’s uid (if user) or gid (if group)"`
	
}


// ViewsResponseBody_BucketLogging represents a nested type for response body
type ViewsResponseBody_BucketLogging struct {
	DestinationId int64 `json:"destination_id,omitempty" yaml:"destination_id,omitempty" required:"false" doc:"The ID of the S3 bucket view configured as the bucket logging destination, to store S3 bucket logs for the view. If destination_id is configured, S3 bucket logging is enabled."`
	KeyFormat string `json:"key_format,omitempty" yaml:"key_format,omitempty" required:"false" doc:"The format for log object keys. SIMPLE_PREFIX=[DestinationPrefix][YYYY]-[MM]-[DD]-[hh]-[mm]-[ss]-[UniqueString], PARTITIONED_PREFIX_EVENT_TIME=[DestinationPrefix][SourceUsername]/[SourceBucket]/[YYYY]/[MM]/[DD]/[YYYY]-[MM]-[DD]-[hh]-[mm]-[ss]-[UniqueString] where the partitioning is done based on the time when the logged events occurred, PARTITIONED_PREFIX_DELIVERY_TIME=[DestinationPrefix][SourceUsername]/[SourceBucket]/[YYYY]/[MM]/[DD]/[YYYY]-[MM]-[DD]-[hh]-[mm]-[ss]-[UniqueString] where the partitioning is done based on the time when the log object has been delivered to the destination bucket. Default: SIMPLE_PREFIX"`
	Prefix string `json:"prefix,omitempty" yaml:"prefix,omitempty" required:"false" doc:"A prefix that is prepended to each key of a log object uploaded to the destination bucket. This prefix can be used to categorize log objects if, for example, you use the same destination bucket for multiple source buckets. The prefix can be up to 128 characters and must follow S3 object naming rules."`
	
}


// ViewsRequestBody_CreateDirAclItem represents a nested type for response body
type ViewsRequestBody_CreateDirAclItem struct {
	Grantee string `json:"grantee,omitempty" yaml:"grantee,omitempty" required:"true" doc:"type of grantee"`
	Perm string `json:"perm,omitempty" yaml:"perm,omitempty" required:"true" doc:"The type of permission to grant to the grantee"`
	GroupType string `json:"group_type,omitempty" yaml:"group_type,omitempty" required:"false" doc:""`
	SidStr string `json:"sid_str,omitempty" yaml:"sid_str,omitempty" required:"false" doc:"SID attribute of grantee. Specify this attribute or another for the grantee."`
	UidOrGid string `json:"uid_or_gid,omitempty" yaml:"uid_or_gid,omitempty" required:"false" doc:"UID of user type grantee or GID of group type grantee. Specify this attribute or another attribute for the grantee."`
	VidOrVaid string `json:"vid_or_vaid,omitempty" yaml:"vid_or_vaid,omitempty" required:"false" doc:"VID of user type grantee or VAID of group type grantee. This is a VAST user or group attribute. Specify this attribute or another attribute for the guarantee."`
	
}


// ViewsRequestBody_ShareAcl_AclItem represents a nested type for response body
type ViewsRequestBody_ShareAcl_AclItem struct {
	Fqdn string `json:"fqdn,omitempty" yaml:"fqdn,omitempty" required:"false" doc:"FQDN of the grantee"`
	Grantee string `json:"grantee,omitempty" yaml:"grantee,omitempty" required:"false" doc:"Type of grantee"`
	Name string `json:"name,omitempty" yaml:"name,omitempty" required:"false" doc:"Name of the grantee"`
	Perm string `json:"perm,omitempty" yaml:"perm,omitempty" required:"false" doc:"Permission to grant to the grantee"`
	SidStr string `json:"sid_str,omitempty" yaml:"sid_str,omitempty" required:"false" doc:"Grantee’s SID"`
	UidOrGid int64 `json:"uid_or_gid,omitempty" yaml:"uid_or_gid,omitempty" required:"false" doc:"Grantee’s uid (if user) or gid (if group)"`
	
}


// ViewsResponseBody_UserImpersonation represents a nested type for response body
type ViewsResponseBody_UserImpersonation struct {
	Enabled bool `json:"enabled,omitempty" yaml:"enabled,omitempty" required:"false" doc:"True if user impersonation is enabled"`
	Identifier string `json:"identifier,omitempty" yaml:"identifier,omitempty" required:"false" doc:"Identifier of the user to impersonate"`
	IdentifierType string `json:"identifier_type,omitempty" yaml:"identifier_type,omitempty" required:"false" doc:"The identifier type of the specified identifier."`
	LoginName string `json:"login_name,omitempty" yaml:"login_name,omitempty" required:"false" doc:"Full username of user to impersonate, including domain name"`
	Username string `json:"username,omitempty" yaml:"username,omitempty" required:"false" doc:"The username of the user to impersonate"`
	
}


// ViewsResponseBody_ShareAcl represents a nested type for response body
type ViewsResponseBody_ShareAcl struct {
	Acl []ViewsResponseBody_ShareAcl_AclItem `json:"acl,omitempty" yaml:"acl,omitempty" required:"false" doc:"Share-level ACL"`
	Enabled bool `json:"enabled,omitempty" yaml:"enabled,omitempty" required:"false" doc:"True if Share ACL is enabled on the view, otherwise False"`
	
}


// ViewsResponseBody_EventNotificationsItem represents a nested type for response body
type ViewsResponseBody_EventNotificationsItem struct {
	BrokerId int64 `json:"broker_id,omitempty" yaml:"broker_id,omitempty" required:"false" doc:"Event broker ID"`
	Name string `json:"name,omitempty" yaml:"name,omitempty" required:"false" doc:"Event unique name"`
	PrefixFilter string `json:"prefix_filter,omitempty" yaml:"prefix_filter,omitempty" required:"false" doc:"Event prefix filter"`
	SuffixFilter string `json:"suffix_filter,omitempty" yaml:"suffix_filter,omitempty" required:"false" doc:"Event suffix filter"`
	Topic string `json:"topic,omitempty" yaml:"topic,omitempty" required:"false" doc:"Event topic"`
	Triggers []string `json:"triggers,omitempty" yaml:"triggers,omitempty" required:"false" doc:"Event triggers"`
	
}


// ViewResponseBody represents the response data for View operations
// Generated from POST response body for resource: views
type ViewResponseBody struct {
	Path string `json:"path,omitempty" yaml:"path,omitempty" required:"true" doc:"The Element Store path exposed by the view. Begin with a forward slash. Do not include a trailing slash"`
	AbacTags []string `json:"abac_tags,omitempty" yaml:"abac_tags,omitempty" required:"false" doc:"Comma separated tags."`
	AbeMaxDepth int64 `json:"abe_max_depth,omitempty" yaml:"abe_max_depth,omitempty" required:"false" doc:"Restricts ABE to a specified path depth. For example, if max depth is 3, ABE does not affect paths deeper than three levels. If not specified, ABE affects all path depths."`
	AbeProtocols []string `json:"abe_protocols,omitempty" yaml:"abe_protocols,omitempty" required:"false" doc:"The protocols for which Access-Based Enumeration (ABE) is enabled"`
	Alias string `json:"alias,omitempty" yaml:"alias,omitempty" required:"false" doc:"Alias for NFS export, must start with '/' and only ASCII characters are allowed. If configured, this supersedes the exposed NFS export path"`
	AllowAnonymousAccess bool `json:"allow_anonymous_access,omitempty" yaml:"allow_anonymous_access,omitempty" required:"false" doc:"Allow S3 anonymous access"`
	AllowS3AnonymousAccess bool `json:"allow_s3_anonymous_access,omitempty" yaml:"allow_s3_anonymous_access,omitempty" required:"false" doc:"Allow S3 anonymous access"`
	AutoCommit string `json:"auto_commit,omitempty" yaml:"auto_commit,omitempty" required:"false" doc:"Applicable if locking is enabled. Sets the auto-commit time for files that are locked automatically. These files are locked automatically after the auto-commit period elapses from the time the file is saved. Files locked automatically are locked for the default-retention-period, after which they are unlocked. Specify as an integer value followed by a letter for the unit (h - hours, d - days, y - years). Example: 2h (2 hours)."`
	Bucket string `json:"bucket,omitempty" yaml:"bucket,omitempty" required:"false" doc:"S3 Bucket name"`
	BucketCreators []string `json:"bucket_creators,omitempty" yaml:"bucket_creators,omitempty" required:"false" doc:"For S3 endpoint buckets, this is a list of users whose bucket create requests use this view."`
	BucketCreatorsGroups []string `json:"bucket_creators_groups,omitempty" yaml:"bucket_creators_groups,omitempty" required:"false" doc:"For S3 endpoint buckets, this is a list of groups whose bucket create requests use this view."`
	BucketLogging ViewsResponseBody_BucketLogging `json:"bucket_logging,omitempty" yaml:"bucket_logging,omitempty" required:"false" doc:"S3 bucket logging configuration. S3 bucket logging records S3 operations on a source bucket, with logs written to a different bucket configured as the destination. When the source bucket has S3 bucket logging enabled, VAST Cluster creates a log entry in AWS log format for each request made to the source bucket, and periodically uploads the log objects to a destination bucket. The format of log object keys can be configured to allow for date-based partitioning of log objects."`
	BucketOwner string `json:"bucket_owner,omitempty" yaml:"bucket_owner,omitempty" required:"false" doc:"S3 Bucket owner"`
	BulkPermissionUpdateProgress int64 `json:"bulk_permission_update_progress,omitempty" yaml:"bulk_permission_update_progress,omitempty" required:"false" doc:"Progress"`
	BulkPermissionUpdateState string `json:"bulk_permission_update_state,omitempty" yaml:"bulk_permission_update_state,omitempty" required:"false" doc:"State"`
	Cluster string `json:"cluster,omitempty" yaml:"cluster,omitempty" required:"false" doc:"Parent Cluster"`
	ClusterId int64 `json:"cluster_id,omitempty" yaml:"cluster_id,omitempty" required:"false" doc:"Parent Cluster ID"`
	CreateDir string `json:"create_dir,omitempty" yaml:"create_dir,omitempty" required:"false" doc:"Creates the directory specified by the path"`
	Created string `json:"created,omitempty" yaml:"created,omitempty" required:"false" doc:""`
	DefaultRetentionPeriod string `json:"default_retention_period,omitempty" yaml:"default_retention_period,omitempty" required:"false" doc:"Default retention period for objects in the bucket. Required if s3_locks_retention_mode is set to governance or compliance. Object versions that are placed in the bucket are automatically protected with the specified retention for the specified amount of time. Otherwise, by default, each object version has no automatic protection but can be configured with a retention period or legal hold. Specify as an integer followed by h for hours, d for days, m for months, or y for years. For example: 2d or 1y."`
	Directory bool `json:"directory,omitempty" yaml:"directory,omitempty" required:"false" doc:"Create the directory if it does not exist"`
	EventNotifications []ViewsResponseBody_EventNotificationsItem `json:"event_notifications,omitempty" yaml:"event_notifications,omitempty" required:"false" doc:""`
	FilesRetentionMode string `json:"files_retention_mode,omitempty" yaml:"files_retention_mode,omitempty" required:"false" doc:"Applicable if locking is enabled. The retention mode for new files. For views enabled for NFSv3 or SMB, if locking is enabled, files_retention_mode must be set to GOVERNANCE or COMPLIANCE. If the view is enabled for S3 and not for NFSv3 or SMB, files_retention_mode can be set to NONE. If GOVERNANCE, locked files cannot be deleted or changed. The Retention settings can be shortened or extended by users with sufficient permissions. If COMPLIANCE, locked files cannot be deleted or changed. Retention settings can be extended, but not shortened, by users with sufficient permissions. If NONE (S3 only), the retention mode is not set for the view; it is set individually for each object."`
	Guid string `json:"guid,omitempty" yaml:"guid,omitempty" required:"false" doc:""`
	HasBucketLoggingDestination bool `json:"has_bucket_logging_destination,omitempty" yaml:"has_bucket_logging_destination,omitempty" required:"false" doc:"Has a destination bucket configured as a destination for S3 bucket logging"`
	HasBucketLoggingSources bool `json:"has_bucket_logging_sources,omitempty" yaml:"has_bucket_logging_sources,omitempty" required:"false" doc:"Is referenced by other S3 bucket views as the destination bucket for S3 bucket logging."`
	Id int64 `json:"id,omitempty" yaml:"id,omitempty" required:"false" doc:""`
	IgnoreOos bool `json:"ignore_oos,omitempty" yaml:"ignore_oos,omitempty" required:"false" doc:""`
	IndestructibleObjectDuration int64 `json:"indestructible_object_duration,omitempty" yaml:"indestructible_object_duration,omitempty" required:"false" doc:"Retention period for indestructible object mode, in days."`
	Internal bool `json:"internal,omitempty" yaml:"internal,omitempty" required:"false" doc:""`
	IsDefaultSubsystem bool `json:"is_default_subsystem,omitempty" yaml:"is_default_subsystem,omitempty" required:"false" doc:"True if the view is the default subsystem for block storage. There can be up to one default subsystem per tenant. The default subsystem is the default view selected when creating a block volume if no view is specified."`
	IsIndestructibleObjectEnabled bool `json:"is_indestructible_object_enabled,omitempty" yaml:"is_indestructible_object_enabled,omitempty" required:"false" doc:"True if indestructible object mode is enabled."`
	IsRemote bool `json:"is_remote,omitempty" yaml:"is_remote,omitempty" required:"false" doc:""`
	IsSeamless bool `json:"is_seamless,omitempty" yaml:"is_seamless,omitempty" required:"false" doc:"Supports seamless failover between replication peers by syncing file handles between the view and remote views on the replicated path on replication peers. This enables NFSv3 client users to retain the same mount point to the view in the event of a failover of the view path to a replication peer. This feature enables NFSv3 client users to retain the same mount point to the view in the event of a failover of the view path to a replication peer. Enabling this option may cause overhead and should only be enabled when the use case is relevant. To complete the configuration for seamless failover between any two peers, a seamless view must be created on each peer."`
	KafkaFirstJoinGroupTimeoutSec int64 `json:"kafka_first_join_group_timeout_sec,omitempty" yaml:"kafka_first_join_group_timeout_sec,omitempty" required:"false" doc:"Kafka first join group timeout in seconds"`
	KafkaRejoinGroupTimeoutSec int64 `json:"kafka_rejoin_group_timeout_sec,omitempty" yaml:"kafka_rejoin_group_timeout_sec,omitempty" required:"false" doc:"Kafka rejoin group timeout in seconds"`
	KafkaVipPools []int64 `json:"kafka_vip_pools,omitempty" yaml:"kafka_vip_pools,omitempty" required:"false" doc:"For Kafka-enabled views, a comma separated list of vip pool IDs used to access event topics exposed by the view. The specified virtual IP pool must belong to the same tenant as the Kafka-enabled view. Must also not be a virtual IP pool that is excluded by the view policy's virtual IP pool association."`
	Locking bool `json:"locking,omitempty" yaml:"locking,omitempty" required:"false" doc:"Write Once Read Many (WORM) locking enabled"`
	LogicalCapacity int64 `json:"logical_capacity,omitempty" yaml:"logical_capacity,omitempty" required:"false" doc:"Logical Capacity consumed by view"`
	MaxRetentionPeriod string `json:"max_retention_period,omitempty" yaml:"max_retention_period,omitempty" required:"false" doc:"Applicable if locking is enabled. Sets a maximum retention period for files that are locked in the view. Files cannot be locked for longer than this period, whether they are locked manually (by setting the atime) or automatically, using auto-commit. Specify as an integer value followed by a letter for the unit (m - minutes, h - hours, d - days, y - years). Example: 2y (2 years)."`
	MinRetentionPeriod string `json:"min_retention_period,omitempty" yaml:"min_retention_period,omitempty" required:"false" doc:"Applicable if locking is enabled. Sets a minimum retention period for files that are locked in the view. Files cannot be locked for less than this period, whether locked manually (by setting the atime) or automatically, using auto-commit. Specify as an integer value followed by a letter for the unit (h - hours, d - days, m - months, y - years). Example: 1d (1 day)."`
	Name string `json:"name,omitempty" yaml:"name,omitempty" required:"false" doc:""`
	Nqn string `json:"nqn,omitempty" yaml:"nqn,omitempty" required:"false" doc:"Applicable to subsystem (block protocol enabled) views. The subsystem's NVMe Qualified Name. A unique identifier used to identify the subsystem in NVMe operations."`
	PhysicalCapacity int64 `json:"physical_capacity,omitempty" yaml:"physical_capacity,omitempty" required:"false" doc:"Physical Capacity consumed by view"`
	Policy string `json:"policy,omitempty" yaml:"policy,omitempty" required:"false" doc:"The name of the associated view policy"`
	PolicyId int64 `json:"policy_id,omitempty" yaml:"policy_id,omitempty" required:"false" doc:"The ID of the associated view policy"`
	Protocols []string `json:"protocols,omitempty" yaml:"protocols,omitempty" required:"false" doc:"Protocols enabled for access to the view. 'NFS' enables access from NFS version 3, 'NFS4' enables access from NFS version 4.1 and 4.2, S3' creates an S3 bucket on the view, 'ENDPOINT' creates an S3 endpoint, used as template for views created via S3 RPCs, DATABASE exposes the view as a VAST database. KAFKA enables events related to elements on the view path to be published to the VAST Event Broker. BLOCK exposes the view as a block storage subsystem.""`
	QosPolicy string `json:"qos_policy,omitempty" yaml:"qos_policy,omitempty" required:"false" doc:"QoS Policy"`
	QosPolicyId int64 `json:"qos_policy_id,omitempty" yaml:"qos_policy_id,omitempty" required:"false" doc:"QoS Policy ID"`
	S3LocksRetentionMode string `json:"s3_locks_retention_mode,omitempty" yaml:"s3_locks_retention_mode,omitempty" required:"false" doc:"The retention mode for new object versions stored in this bucket. You can override this if you upload a new object version with an explicit retention mode and period."`
	S3ObjectOwnershipRule string `json:"s3_object_ownership_rule,omitempty" yaml:"s3_object_ownership_rule,omitempty" required:"false" doc:""`
	S3UnverifiedLookup bool `json:"s3_unverified_lookup,omitempty" yaml:"s3_unverified_lookup,omitempty" required:"false" doc:"S3 Unverified Lookup"`
	S3Versioning bool `json:"s3_versioning,omitempty" yaml:"s3_versioning,omitempty" required:"false" doc:"S3 Versioning enabled on S3 bucket."`
	SelectForLiveMonitoring bool `json:"select_for_live_monitoring,omitempty" yaml:"select_for_live_monitoring,omitempty" required:"false" doc:"True when the view has live monitoring enabled.  Views that have live monitoring enabled are polled for metrics every ten seconds. Otherwise, views are polled every five minutes."`
	Share string `json:"share,omitempty" yaml:"share,omitempty" required:"false" doc:"Name of the SMB share. Must not include certain special characters."`
	ShareAcl ViewsResponseBody_ShareAcl `json:"share_acl,omitempty" yaml:"share_acl,omitempty" required:"false" doc:"Share-level ACL details"`
	Sync string `json:"sync,omitempty" yaml:"sync,omitempty" required:"false" doc:"Synchronization state with leader"`
	SyncTime string `json:"sync_time,omitempty" yaml:"sync_time,omitempty" required:"false" doc:"Synchronization time with leader"`
	TenantId int64 `json:"tenant_id,omitempty" yaml:"tenant_id,omitempty" required:"false" doc:"Tenant ID"`
	TenantName string `json:"tenant_name,omitempty" yaml:"tenant_name,omitempty" required:"false" doc:"Tenant Name"`
	Title string `json:"title,omitempty" yaml:"title,omitempty" required:"false" doc:""`
	Url string `json:"url,omitempty" yaml:"url,omitempty" required:"false" doc:"The endpoint URL for API operations on the view"`
	UserImpersonation ViewsResponseBody_UserImpersonation `json:"user_impersonation,omitempty" yaml:"user_impersonation,omitempty" required:"false" doc:""`
	
}

// -----------------------------------------------------
// RESOURCE METHODS
// -----------------------------------------------------

// View represents a typed resource for view operations
type View struct {
	Untyped *vast_client.VMSRest
}

// Get retrieves a single view with typed request/response
func (r *View) Get(req *ViewSearchParams) (*ViewResponseBody, error) {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		return nil, err
	}

	record, err := r.Untyped.Views.Get(params)
	if err != nil {
		return nil, err
	}

	var response ViewResponseBody
	if err := record.Fill(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

// GetWithContext retrieves a single view with typed request/response using provided context
func (r *View) GetWithContext(ctx context.Context, req *ViewSearchParams) (*ViewResponseBody, error) {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		return nil, err
	}
	
	record, err := r.Untyped.Views.GetWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	var response ViewResponseBody
	if err := record.Fill(&response); err != nil {
		return nil, err
	}
	
	return &response, nil
}

// GetById retrieves a single view by ID
func (r *View) GetById(id any) (*ViewResponseBody, error) {
	record, err := r.Untyped.Views.GetById(id)
	if err != nil {
		return nil, err
	}

	var response ViewResponseBody
	if err := record.Fill(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

// GetByIdWithContext retrieves a single view by ID using provided context
func (r *View) GetByIdWithContext(ctx context.Context, id any) (*ViewResponseBody, error) {
	record, err := r.Untyped.Views.GetByIdWithContext(ctx, id)
	if err != nil {
		return nil, err
	}

	var response ViewResponseBody
	if err := record.Fill(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

// List retrieves multiple views with typed request/response
func (r *View) List(req *ViewSearchParams) ([]*ViewResponseBody, error) {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		return nil, err
	}
	
	recordSet, err := r.Untyped.Views.List(params)
	if err != nil {
		return nil, err
	}

	var response []*ViewResponseBody
	if err := recordSet.Fill(&response); err != nil {
		return nil, err
	}
	
	return response, nil
}

// ListWithContext retrieves multiple views with typed request/response using provided context
func (r *View) ListWithContext(ctx context.Context, req *ViewSearchParams) ([]*ViewResponseBody, error) {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		return nil, err
	}
	
	recordSet, err := r.Untyped.Views.ListWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	var response []*ViewResponseBody
	if err := recordSet.Fill(&response); err != nil {
		return nil, err
	}
	
	return response, nil
}


// Create creates a new view with typed request/response
func (r *View) Create(req *ViewRequestBody) (*ViewResponseBody, error) {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		return nil, err
	}

	record, err := r.Untyped.Views.Create(params)
	if err != nil {
		return nil, err
	}

	var response ViewResponseBody
	if err := record.Fill(&response); err != nil {
		return nil, err
	}
	
	return &response, nil
}

// CreateWithContext creates a new view with typed request/response using provided context
func (r *View) CreateWithContext(ctx context.Context, req *ViewRequestBody) (*ViewResponseBody, error) {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		return nil, err
	}

	record, err := r.Untyped.Views.CreateWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	var response ViewResponseBody
	if err := record.Fill(&response); err != nil {
		return nil, err
	}
	
	return &response, nil
}


// Update updates an existing view with typed request/response
func (r *View) Update(id any, req *ViewRequestBody) (*ViewResponseBody, error) {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		return nil, err
	}

	record, err := r.Untyped.Views.Update(id, params)
	if err != nil {
		return nil, err
	}

	var response ViewResponseBody
	if err := record.Fill(&response); err != nil {
		return nil, err
	}
	
	return &response, nil
}

// UpdateWithContext updates an existing view with typed request/response using provided context
func (r *View) UpdateWithContext(ctx context.Context, id any, req *ViewRequestBody) (*ViewResponseBody, error) {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		return nil, err
	}

	record, err := r.Untyped.Views.UpdateWithContext(ctx, id, params)
	if err != nil {
		return nil, err
	}

	var response ViewResponseBody
	if err := record.Fill(&response); err != nil {
		return nil, err
	}
	
	return &response, nil
}



// Delete deletes a view with search parameters
func (r *View) Delete(req *ViewSearchParams) error {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		return err
	}
	_, err = r.Untyped.Views.Delete(params, nil)
	if err != nil {
		return err
	}
	return nil
}

// DeleteWithContext deletes a view with search parameters using provided context
func (r *View) DeleteWithContext(ctx context.Context, req *ViewSearchParams) error {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		return err
	}
	_, err = r.Untyped.Views.DeleteWithContext(ctx, params, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

// DeleteById deletes a view by ID
func (r *View) DeleteById(id any) error {
	_, err := r.Untyped.Views.DeleteById(id, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

// DeleteByIdWithContext deletes a view by ID using provided context
func (r *View) DeleteByIdWithContext(ctx context.Context, id any) error {
	_, err := r.Untyped.Views.DeleteByIdWithContext(ctx, id, nil, nil)
	if err != nil {
		return err
	}
	return nil
}


// Ensure ensures a view exists with typed response
func (r *View) Ensure(searchParams *ViewSearchParams, body *ViewRequestBody) (*ViewResponseBody, error) {
	searchParamsConverted, err := vast_client.NewParamsFromStruct(searchParams)
	if err != nil {
		return nil, err
	}
	bodyConverted, err := vast_client.NewParamsFromStruct(body)
	if err != nil {
		return nil, err
	}
	
	record, err := r.Untyped.Views.Ensure(searchParamsConverted, bodyConverted)
	if err != nil {
		return nil, err
	}

	var response ViewResponseBody
	if err := record.Fill(&response); err != nil {
		return nil, err
	}
	
	return &response, nil
}

// EnsureWithContext ensures a view exists with typed response using provided context
func (r *View) EnsureWithContext(ctx context.Context, searchParams *ViewSearchParams, body *ViewRequestBody) (*ViewResponseBody, error) {
	searchParamsConverted, err := vast_client.NewParamsFromStruct(searchParams)
	if err != nil {
		return nil, err
	}
	bodyConverted, err := vast_client.NewParamsFromStruct(body)
	if err != nil {
		return nil, err
	}
	
	record, err := r.Untyped.Views.EnsureWithContext(ctx, searchParamsConverted, bodyConverted)
	if err != nil {
		return nil, err
	}

	var response ViewResponseBody
	if err := record.Fill(&response); err != nil {
		return nil, err
	}
	
	return &response, nil
}

// EnsureByName ensures a view exists by name with typed response
func (r *View) EnsureByName(name string, body *ViewRequestBody) (*ViewResponseBody, error) {
	bodyConverted, err := vast_client.NewParamsFromStruct(body)
	if err != nil {
		return nil, err
	}
	
	record, err := r.Untyped.Views.EnsureByName(name, bodyConverted)
	if err != nil {
		return nil, err
	}

	var response ViewResponseBody
	if err := record.Fill(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

// EnsureByNameWithContext ensures a view exists by name with typed response using provided context
func (r *View) EnsureByNameWithContext(ctx context.Context, name string, body *ViewRequestBody) (*ViewResponseBody, error) {
	bodyConverted, err := vast_client.NewParamsFromStruct(body)
	if err != nil {
		return nil, err
	}
	
	record, err := r.Untyped.Views.EnsureByNameWithContext(ctx, name, bodyConverted)
	if err != nil {
		return nil, err
	}

	var response ViewResponseBody
	if err := record.Fill(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

// Exists checks if a view exists
func (r *View) Exists(req *ViewSearchParams) (bool, error) {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		return false, err
	}
	return r.Untyped.Views.Exists(params)
}

// ExistsWithContext checks if a view exists using provided context
func (r *View) ExistsWithContext(ctx context.Context, req *ViewSearchParams) (bool, error) {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		return false, err
	}
	return r.Untyped.Views.ExistsWithContext(ctx, params)
}

// MustExists checks if a view exists and panics if not
func (r *View) MustExists(req *ViewSearchParams) bool {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		panic(err)
	}
	return r.Untyped.Views.MustExists(params)
}

// MustExistsWithContext checks if a view exists and panics if not using provided context
func (r *View) MustExistsWithContext(ctx context.Context, req *ViewSearchParams) bool {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		panic(err)
	}
	return r.Untyped.Views.MustExistsWithContext(ctx, params)
}


