package typed

import (
	"context"

	vast_client "github.com/vast-data/go-vast-client"
)

// -----------------------------------------------------
// SEARCH PARAMS
// -----------------------------------------------------

// QuotaSearchParams represents the search parameters for Quota operations
// Generated from GET query parameters for resource: quotas
type QuotaSearchParams struct {
	HardLimit string `json:"hard_limit,omitempty" yaml:"hard_limit,omitempty" required:"false" doc:"Filter results by hard capacity limit."`
	HardLimitInodes string `json:"hard_limit_inodes,omitempty" yaml:"hard_limit_inodes,omitempty" required:"false" doc:"Filter results by hard limit on number of files and directories"`
	Name string `json:"name,omitempty" yaml:"name,omitempty" required:"false" doc:""`
	ShowUserRules bool `json:"show_user_rules,omitempty" yaml:"show_user_rules,omitempty" required:"false" doc:"Include user and group quota rules in response."`
	SoftLimit string `json:"soft_limit,omitempty" yaml:"soft_limit,omitempty" required:"false" doc:"Filter results by soft capacity limit."`
	SoftLimitInodes string `json:"soft_limit_inodes,omitempty" yaml:"soft_limit_inodes,omitempty" required:"false" doc:"Filter results by soft limit on number of files and directories."`
	SystemId string `json:"system_id,omitempty" yaml:"system_id,omitempty" required:"false" doc:""`
	TenantId int64 `json:"tenant_id,omitempty" yaml:"tenant_id,omitempty" required:"false" doc:"Filter by tenant. Specify tenant ID."`
	TenantNameIcontains string `json:"tenant_name__icontains,omitempty" yaml:"tenant_name__icontains,omitempty" required:"false" doc:"Tenant name to filter by"`
	
}

// -----------------------------------------------------
// REQUEST BODY
// -----------------------------------------------------

// QuotaRequestBody represents the request body for Quota operations
// Generated from POST request body for resource: quotas
type QuotaRequestBody struct {
	CreateDir bool `json:"create_dir,omitempty" yaml:"create_dir,omitempty" required:"false" doc:"Set to true to create the directory if the directory was not created yet."`
	CreateDirMode int64 `json:"create_dir_mode,omitempty" yaml:"create_dir_mode,omitempty" required:"false" doc:"Unix permissions mode for the new directory"`
	DefaultEmail string `json:"default_email,omitempty" yaml:"default_email,omitempty" required:"false" doc:"Emails are sent to users if and when they exceed their user/group quota limits. default_email is a default email address that is used instead of a user's email address in the event that no email address is found for the user on a provider and no email suffix is set."`
	DefaultGroupQuota string `json:"default_group_quota,omitempty" yaml:"default_group_quota,omitempty" required:"false" doc:""`
	DefaultUserQuota string `json:"default_user_quota,omitempty" yaml:"default_user_quota,omitempty" required:"false" doc:""`
	EnableAlarms bool `json:"enable_alarms,omitempty" yaml:"enable_alarms,omitempty" required:"false" doc:"Enables alarms on relevant events for user and group quotas. Applicable only if is_user_quota is true. Raises alarms reporting the number of users that exceed their quotas and when one or more users is/are blocked from writing to the quota directory."`
	EnableEmailProviders bool `json:"enable_email_providers,omitempty" yaml:"enable_email_providers,omitempty" required:"false" doc:"Set to true to enable querying Active Directory and LDAP services for user emails when sending user notifications to users if they exceed their user/group quota limits. If enabled, the provider query is the first priority source for a user's email. If a user's email is not found on the provider, a global suffix is used to form an email. If no suffix is set, default_email is used."`
	GracePeriod string `json:"grace_period,omitempty" yaml:"grace_period,omitempty" required:"false" doc:"Quota enforcement grace period. An alarm is triggered and write operations are blocked if storage usage continues to exceed the soft limit for the grace period. Format: [DD] [HH:[MM:]]ss"`
	GroupQuotas []string `json:"group_quotas,omitempty" yaml:"group_quotas,omitempty" required:"false" doc:"An array of group quota rule objects. A group quota rule overrides a default group quota rule for the specified group."`
	HardLimit int64 `json:"hard_limit,omitempty" yaml:"hard_limit,omitempty" required:"false" doc:"Storage usage limit beyond which no writes will be allowed."`
	HardLimitInodes int64 `json:"hard_limit_inodes,omitempty" yaml:"hard_limit_inodes,omitempty" required:"false" doc:"Number of directories and unique files under the path beyond which no writes will be allowed. A file with multiple hardlinks is counted only once."`
	InheritAcl bool `json:"inherit_acl,omitempty" yaml:"inherit_acl,omitempty" required:"false" doc:"Indicates whether the directory should inherit ACLs from its parent directory"`
	IsUserQuota bool `json:"is_user_quota,omitempty" yaml:"is_user_quota,omitempty" required:"false" doc:"Set to true to enable user and group quotas. False by default. Cannot be disabled later."`
	Name string `json:"name,omitempty" yaml:"name,omitempty" required:"false" doc:"A name for the quota"`
	Path string `json:"path,omitempty" yaml:"path,omitempty" required:"false" doc:"The directory path on which to enforce the quota"`
	SoftLimit int64 `json:"soft_limit,omitempty" yaml:"soft_limit,omitempty" required:"false" doc:"Storage usage limit at which warnings of exceeding the quota are issued."`
	SoftLimitInodes int64 `json:"soft_limit_inodes,omitempty" yaml:"soft_limit_inodes,omitempty" required:"false" doc:"Number of directories and unique files under the path at which warnings of exceeding the quota will be issued. A file with multiple hardlinks is counted only once."`
	TenantId int64 `json:"tenant_id,omitempty" yaml:"tenant_id,omitempty" required:"false" doc:"Tenant ID"`
	UserQuotas []string `json:"user_quotas,omitempty" yaml:"user_quotas,omitempty" required:"false" doc:"An array of user quota rule objects. A user quota rule overrides a default user quota rule for the specified user."`
	
}

// -----------------------------------------------------
// RESPONSE BODY
// -----------------------------------------------------


// QuotasResponseBody_DefaultUserQuota represents a nested type for response body
type QuotasResponseBody_DefaultUserQuota struct {
	GracePeriod string `json:"grace_period,omitempty" yaml:"grace_period,omitempty" required:"false" doc:"Quota enforcement grace period in seconds, minutes, hours or days. Example: 90m"`
	HardLimit int64 `json:"hard_limit,omitempty" yaml:"hard_limit,omitempty" required:"false" doc:"Hard quota limit"`
	HardLimitInodes int64 `json:"hard_limit_inodes,omitempty" yaml:"hard_limit_inodes,omitempty" required:"false" doc:"Hard inodes quota limit"`
	QuotaSystemId int64 `json:"quota_system_id,omitempty" yaml:"quota_system_id,omitempty" required:"false" doc:""`
	SoftLimit int64 `json:"soft_limit,omitempty" yaml:"soft_limit,omitempty" required:"false" doc:"Soft quota limit"`
	SoftLimitInodes int64 `json:"soft_limit_inodes,omitempty" yaml:"soft_limit_inodes,omitempty" required:"false" doc:"Soft inodes quota limit"`
	
}


// QuotasResponseBody_DefaultGroupQuota represents a nested type for response body
type QuotasResponseBody_DefaultGroupQuota struct {
	GracePeriod string `json:"grace_period,omitempty" yaml:"grace_period,omitempty" required:"false" doc:"Quota enforcement grace period in seconds, minutes, hours or days. Example: 90m"`
	HardLimit int64 `json:"hard_limit,omitempty" yaml:"hard_limit,omitempty" required:"false" doc:"Hard quota limit"`
	HardLimitInodes int64 `json:"hard_limit_inodes,omitempty" yaml:"hard_limit_inodes,omitempty" required:"false" doc:"Hard inodes quota limit"`
	QuotaSystemId int64 `json:"quota_system_id,omitempty" yaml:"quota_system_id,omitempty" required:"false" doc:""`
	SoftLimit int64 `json:"soft_limit,omitempty" yaml:"soft_limit,omitempty" required:"false" doc:"Soft quota limit"`
	SoftLimitInodes int64 `json:"soft_limit_inodes,omitempty" yaml:"soft_limit_inodes,omitempty" required:"false" doc:"Soft inodes quota limit"`
	
}


// QuotaResponseBody represents the response data for Quota operations
// Generated from POST response body for resource: quotas
type QuotaResponseBody struct {
	Cluster string `json:"cluster,omitempty" yaml:"cluster,omitempty" required:"false" doc:"Parent Cluster"`
	ClusterId int64 `json:"cluster_id,omitempty" yaml:"cluster_id,omitempty" required:"false" doc:"Parent Cluster ID"`
	DefaultEmail string `json:"default_email,omitempty" yaml:"default_email,omitempty" required:"false" doc:"The default email for sending user quota alert emails. This is used if no suffix is set and no address is found on providers."`
	DefaultGroupQuota QuotasResponseBody_DefaultGroupQuota `json:"default_group_quota,omitempty" yaml:"default_group_quota,omitempty" required:"false" doc:""`
	DefaultUserQuota QuotasResponseBody_DefaultUserQuota `json:"default_user_quota,omitempty" yaml:"default_user_quota,omitempty" required:"false" doc:""`
	EnableAlarms bool `json:"enable_alarms,omitempty" yaml:"enable_alarms,omitempty" required:"false" doc:"Enable alarms when users or groups are exceeding their limit"`
	EnableEmailProviders bool `json:"enable_email_providers,omitempty" yaml:"enable_email_providers,omitempty" required:"false" doc:"Enable this setting to query Active Directory and LDAP services for user emails when sending userquota alert emails. If enabled, the provider query is the first priority source for a user's email. If a user's email is not found on the provider, a global email suffix is used if configured in cluster settings. If no suffix is set, default_email is used."`
	GracePeriod string `json:"grace_period,omitempty" yaml:"grace_period,omitempty" required:"false" doc:"Quota enforcement grace period in seconds, minutes, hours or days. Example: 90m"`
	GroupQuotas []string `json:"group_quotas,omitempty" yaml:"group_quotas,omitempty" required:"false" doc:""`
	Guid string `json:"guid,omitempty" yaml:"guid,omitempty" required:"false" doc:"Quota guid"`
	HardLimit int64 `json:"hard_limit,omitempty" yaml:"hard_limit,omitempty" required:"false" doc:"Storage space usage limit beyond which no writes are allowed."`
	HardLimitInodes int64 `json:"hard_limit_inodes,omitempty" yaml:"hard_limit_inodes,omitempty" required:"false" doc:"Number of directories and unique files under the path beyond which no writes will be allowed. A file with multiple hardlinks is counted only once."`
	Id int64 `json:"id,omitempty" yaml:"id,omitempty" required:"false" doc:""`
	Internal bool `json:"internal,omitempty" yaml:"internal,omitempty" required:"false" doc:""`
	IsUserQuota bool `json:"is_user_quota,omitempty" yaml:"is_user_quota,omitempty" required:"false" doc:"Set to true to enable user and group quotas. False by default."`
	LastUserQuotasUpdate string `json:"last_user_quotas_update,omitempty" yaml:"last_user_quotas_update,omitempty" required:"false" doc:"Time of last user quota update"`
	Name string `json:"name,omitempty" yaml:"name,omitempty" required:"false" doc:"The name"`
	NumBlockedUsers int64 `json:"num_blocked_users,omitempty" yaml:"num_blocked_users,omitempty" required:"false" doc:"The number of users that are blocked from writing to the quota path due to exceeding a hard user/group quota limit."`
	NumExceededUsers int64 `json:"num_exceeded_users,omitempty" yaml:"num_exceeded_users,omitempty" required:"false" doc:"The number of users that have exceeded a user quota"`
	Path string `json:"path,omitempty" yaml:"path,omitempty" required:"false" doc:"Directory path"`
	PercentCapacity int64 `json:"percent_capacity,omitempty" yaml:"percent_capacity,omitempty" required:"false" doc:"Percentage in use of the capacity hard limit"`
	PercentInodes int64 `json:"percent_inodes,omitempty" yaml:"percent_inodes,omitempty" required:"false" doc:"Percentage in use of the hard limit on directories and unique files"`
	PrettyGracePeriod string `json:"pretty_grace_period,omitempty" yaml:"pretty_grace_period,omitempty" required:"false" doc:"Quota enforcement grace period expressed in human readable format as seconds, minutes, hours or days. Example: 12 days 43 minutes 43 seconds"`
	PrettyGracePeriodExpiration string `json:"pretty_grace_period_expiration,omitempty" yaml:"pretty_grace_period_expiration,omitempty" required:"false" doc:"The time remaining until the end of the grace period, in human readable format. Displayed when soft limit is exceeded."`
	PrettyState string `json:"pretty_state,omitempty" yaml:"pretty_state,omitempty" required:"false" doc:""`
	SoftLimit int64 `json:"soft_limit,omitempty" yaml:"soft_limit,omitempty" required:"false" doc:"Storage usage limit at which warnings of exceeding the quota are issued."`
	SoftLimitInodes int64 `json:"soft_limit_inodes,omitempty" yaml:"soft_limit_inodes,omitempty" required:"false" doc:"Number of directories and unique files under the path at which warnings of exceeding the quota will be issued. A file with multiple hardlinks is counted only once."`
	State string `json:"state,omitempty" yaml:"state,omitempty" required:"false" doc:"Quota state"`
	SyncState string `json:"sync_state,omitempty" yaml:"sync_state,omitempty" required:"false" doc:""`
	SystemId int64 `json:"system_id,omitempty" yaml:"system_id,omitempty" required:"false" doc:""`
	TenantId int64 `json:"tenant_id,omitempty" yaml:"tenant_id,omitempty" required:"false" doc:"Tenant ID"`
	TenantName string `json:"tenant_name,omitempty" yaml:"tenant_name,omitempty" required:"false" doc:"Tenant Name"`
	TimeToBlock string `json:"time_to_block,omitempty" yaml:"time_to_block,omitempty" required:"false" doc:"The time remaining until the end of the grace period. Displayed when soft limit is exceeded."`
	Title string `json:"title,omitempty" yaml:"title,omitempty" required:"false" doc:"Quota name"`
	Url string `json:"url,omitempty" yaml:"url,omitempty" required:"false" doc:"Endpoint URL for API operations on the quota"`
	UsedCapacity int64 `json:"used_capacity,omitempty" yaml:"used_capacity,omitempty" required:"false" doc:"Used capacity in bytes"`
	UsedCapacityTb float32 `json:"used_capacity_tb,omitempty" yaml:"used_capacity_tb,omitempty" required:"false" doc:"Used capacity in TB"`
	UsedEffectiveCapacity int64 `json:"used_effective_capacity,omitempty" yaml:"used_effective_capacity,omitempty" required:"false" doc:"Used effective capacity in bytes"`
	UsedEffectiveCapacityTb float32 `json:"used_effective_capacity_tb,omitempty" yaml:"used_effective_capacity_tb,omitempty" required:"false" doc:"Used effective capacity in TB"`
	UsedInodes int64 `json:"used_inodes,omitempty" yaml:"used_inodes,omitempty" required:"false" doc:"Number of directories and unique files under the path"`
	UsedLimitedCapacity int64 `json:"used_limited_capacity,omitempty" yaml:"used_limited_capacity,omitempty" required:"false" doc:""`
	UserQuotas []string `json:"user_quotas,omitempty" yaml:"user_quotas,omitempty" required:"false" doc:"An array of user quota rule objects. A user quota rule overrides a default user quota rule for the specified user."`
	
}

// -----------------------------------------------------
// RESOURCE METHODS
// -----------------------------------------------------

// Quota represents a typed resource for quota operations
type Quota struct {
	Untyped *vast_client.VMSRest
}

// Get retrieves a single quota with typed request/response
func (r *Quota) Get(req *QuotaSearchParams) (*QuotaResponseBody, error) {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		return nil, err
	}

	record, err := r.Untyped.Quotas.Get(params)
	if err != nil {
		return nil, err
	}

	var response QuotaResponseBody
	if err := record.Fill(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

// GetWithContext retrieves a single quota with typed request/response using provided context
func (r *Quota) GetWithContext(ctx context.Context, req *QuotaSearchParams) (*QuotaResponseBody, error) {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		return nil, err
	}
	
	record, err := r.Untyped.Quotas.GetWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	var response QuotaResponseBody
	if err := record.Fill(&response); err != nil {
		return nil, err
	}
	
	return &response, nil
}

// GetById retrieves a single quota by ID
func (r *Quota) GetById(id any) (*QuotaResponseBody, error) {
	record, err := r.Untyped.Quotas.GetById(id)
	if err != nil {
		return nil, err
	}

	var response QuotaResponseBody
	if err := record.Fill(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

// GetByIdWithContext retrieves a single quota by ID using provided context
func (r *Quota) GetByIdWithContext(ctx context.Context, id any) (*QuotaResponseBody, error) {
	record, err := r.Untyped.Quotas.GetByIdWithContext(ctx, id)
	if err != nil {
		return nil, err
	}

	var response QuotaResponseBody
	if err := record.Fill(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

// List retrieves multiple quotas with typed request/response
func (r *Quota) List(req *QuotaSearchParams) ([]*QuotaResponseBody, error) {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		return nil, err
	}
	
	recordSet, err := r.Untyped.Quotas.List(params)
	if err != nil {
		return nil, err
	}

	var response []*QuotaResponseBody
	if err := recordSet.Fill(&response); err != nil {
		return nil, err
	}
	
	return response, nil
}

// ListWithContext retrieves multiple quotas with typed request/response using provided context
func (r *Quota) ListWithContext(ctx context.Context, req *QuotaSearchParams) ([]*QuotaResponseBody, error) {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		return nil, err
	}
	
	recordSet, err := r.Untyped.Quotas.ListWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	var response []*QuotaResponseBody
	if err := recordSet.Fill(&response); err != nil {
		return nil, err
	}
	
	return response, nil
}


// Create creates a new quota with typed request/response
func (r *Quota) Create(req *QuotaRequestBody) (*QuotaResponseBody, error) {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		return nil, err
	}

	record, err := r.Untyped.Quotas.Create(params)
	if err != nil {
		return nil, err
	}

	var response QuotaResponseBody
	if err := record.Fill(&response); err != nil {
		return nil, err
	}
	
	return &response, nil
}

// CreateWithContext creates a new quota with typed request/response using provided context
func (r *Quota) CreateWithContext(ctx context.Context, req *QuotaRequestBody) (*QuotaResponseBody, error) {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		return nil, err
	}

	record, err := r.Untyped.Quotas.CreateWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	var response QuotaResponseBody
	if err := record.Fill(&response); err != nil {
		return nil, err
	}
	
	return &response, nil
}


// Update updates an existing quota with typed request/response
func (r *Quota) Update(id any, req *QuotaRequestBody) (*QuotaResponseBody, error) {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		return nil, err
	}

	record, err := r.Untyped.Quotas.Update(id, params)
	if err != nil {
		return nil, err
	}

	var response QuotaResponseBody
	if err := record.Fill(&response); err != nil {
		return nil, err
	}
	
	return &response, nil
}

// UpdateWithContext updates an existing quota with typed request/response using provided context
func (r *Quota) UpdateWithContext(ctx context.Context, id any, req *QuotaRequestBody) (*QuotaResponseBody, error) {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		return nil, err
	}

	record, err := r.Untyped.Quotas.UpdateWithContext(ctx, id, params)
	if err != nil {
		return nil, err
	}

	var response QuotaResponseBody
	if err := record.Fill(&response); err != nil {
		return nil, err
	}
	
	return &response, nil
}



// Delete deletes a quota with search parameters
func (r *Quota) Delete(req *QuotaSearchParams) error {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		return err
	}
	_, err = r.Untyped.Quotas.Delete(params, nil)
	if err != nil {
		return err
	}
	return nil
}

// DeleteWithContext deletes a quota with search parameters using provided context
func (r *Quota) DeleteWithContext(ctx context.Context, req *QuotaSearchParams) error {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		return err
	}
	_, err = r.Untyped.Quotas.DeleteWithContext(ctx, params, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

// DeleteById deletes a quota by ID
func (r *Quota) DeleteById(id any) error {
	_, err := r.Untyped.Quotas.DeleteById(id, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

// DeleteByIdWithContext deletes a quota by ID using provided context
func (r *Quota) DeleteByIdWithContext(ctx context.Context, id any) error {
	_, err := r.Untyped.Quotas.DeleteByIdWithContext(ctx, id, nil, nil)
	if err != nil {
		return err
	}
	return nil
}


// Ensure ensures a quota exists with typed response
func (r *Quota) Ensure(searchParams *QuotaSearchParams, body *QuotaRequestBody) (*QuotaResponseBody, error) {
	searchParamsConverted, err := vast_client.NewParamsFromStruct(searchParams)
	if err != nil {
		return nil, err
	}
	bodyConverted, err := vast_client.NewParamsFromStruct(body)
	if err != nil {
		return nil, err
	}
	
	record, err := r.Untyped.Quotas.Ensure(searchParamsConverted, bodyConverted)
	if err != nil {
		return nil, err
	}

	var response QuotaResponseBody
	if err := record.Fill(&response); err != nil {
		return nil, err
	}
	
	return &response, nil
}

// EnsureWithContext ensures a quota exists with typed response using provided context
func (r *Quota) EnsureWithContext(ctx context.Context, searchParams *QuotaSearchParams, body *QuotaRequestBody) (*QuotaResponseBody, error) {
	searchParamsConverted, err := vast_client.NewParamsFromStruct(searchParams)
	if err != nil {
		return nil, err
	}
	bodyConverted, err := vast_client.NewParamsFromStruct(body)
	if err != nil {
		return nil, err
	}
	
	record, err := r.Untyped.Quotas.EnsureWithContext(ctx, searchParamsConverted, bodyConverted)
	if err != nil {
		return nil, err
	}

	var response QuotaResponseBody
	if err := record.Fill(&response); err != nil {
		return nil, err
	}
	
	return &response, nil
}

// EnsureByName ensures a quota exists by name with typed response
func (r *Quota) EnsureByName(name string, body *QuotaRequestBody) (*QuotaResponseBody, error) {
	bodyConverted, err := vast_client.NewParamsFromStruct(body)
	if err != nil {
		return nil, err
	}
	
	record, err := r.Untyped.Quotas.EnsureByName(name, bodyConverted)
	if err != nil {
		return nil, err
	}

	var response QuotaResponseBody
	if err := record.Fill(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

// EnsureByNameWithContext ensures a quota exists by name with typed response using provided context
func (r *Quota) EnsureByNameWithContext(ctx context.Context, name string, body *QuotaRequestBody) (*QuotaResponseBody, error) {
	bodyConverted, err := vast_client.NewParamsFromStruct(body)
	if err != nil {
		return nil, err
	}
	
	record, err := r.Untyped.Quotas.EnsureByNameWithContext(ctx, name, bodyConverted)
	if err != nil {
		return nil, err
	}

	var response QuotaResponseBody
	if err := record.Fill(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

// Exists checks if a quota exists
func (r *Quota) Exists(req *QuotaSearchParams) (bool, error) {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		return false, err
	}
	return r.Untyped.Quotas.Exists(params)
}

// ExistsWithContext checks if a quota exists using provided context
func (r *Quota) ExistsWithContext(ctx context.Context, req *QuotaSearchParams) (bool, error) {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		return false, err
	}
	return r.Untyped.Quotas.ExistsWithContext(ctx, params)
}

// MustExists checks if a quota exists and panics if not
func (r *Quota) MustExists(req *QuotaSearchParams) bool {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		panic(err)
	}
	return r.Untyped.Quotas.MustExists(params)
}

// MustExistsWithContext checks if a quota exists and panics if not using provided context
func (r *Quota) MustExistsWithContext(ctx context.Context, req *QuotaSearchParams) bool {
	params, err := vast_client.NewParamsFromStruct(req)
	if err != nil {
		panic(err)
	}
	return r.Untyped.Quotas.MustExistsWithContext(ctx, params)
}


