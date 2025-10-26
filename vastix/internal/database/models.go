package database

import (
	"fmt"
	"time"
	"vastix/internal/client"

	vast_client "github.com/vast-data/go-vast-client"
	"gorm.io/gorm"
)

// Profile represents connection information for server access
type Profile struct {
	gorm.Model

	// Alias is a user-friendly name for the profile (max 20 characters)
	Alias string `json:"alias" gorm:"size:20"`

	// Endpoint is the IP or URL connection to server
	Endpoint string `json:"endpoint" gorm:"not null"`

	// Port for the connection (default: 443 for HTTPS)
	Port int64 `json:"port" gorm:"default:443"`

	// Username for authentication (can be empty if using token)
	Username string `json:"username"`

	// Password for authentication (can be empty if using token)
	Password string `json:"password"`

	// Token is an alternative to username/password authentication
	Token string `json:"token"`

	// Tenant is optional field for tenant-based connections
	Tenant string `json:"tenant"`

	// SSLVerify indicates whether to verify SSL certificates
	SSLVerify bool `json:"ssl_verify"`

	// Active indicates if this profile is currently active (only one can be active)
	// Note: Database constraint ensures only one profile can be active at a time
	Active bool `json:"active" gorm:"default:false;index:idx_active_unique,where:active = true"`

	// VastVersion is the version of the Vast cluster
	VastVersion string `json:"vast_version"`

	// ApiVersion selects REST API version (required)
	ApiVersion string `json:"api_version" gorm:"not null"`
}

func (p *Profile) ProfileName() string {
	palias := p.Alias
	pname := p.Endpoint
	if palias != "" {
		pname = fmt.Sprintf("%s [%s]", palias, pname)
	}
	return pname
}

func (p *Profile) RestClientFromProfile() (*vast_client.VMSRest, error) {
	// Shortcut for getting the global REST client service from profile
	config := client.RestClientConfig{
		Host:       p.Endpoint,
		Port:       p.Port,
		Username:   p.Username,
		Password:   p.Password,
		ApiToken:   p.Token,
		Tenant:     p.Tenant,
		SslVerify:  p.SSLVerify,
		ApiVersion: p.ApiVersion,
	}
	return client.GetGlobalClient(config)

}

// ResourceHistory represents the navigation history of resources
type ResourceHistory struct {
	gorm.Model

	// CurrentResource is the currently active resource type
	CurrentResource string `json:"current_resource" gorm:"not null;size:50"`

	// PreviousResource is the previously active resource type
	PreviousResource string `json:"previous_resource" gorm:"size:50"`
}

// UserKey represents access and secret keys generated for users
type UserKey struct {
	gorm.Model

	// ProfileID is the foreign key reference to the profile from which the key was created
	// This will automatically cascade delete when the profile is deleted
	ProfileID uint `json:"profile_id" gorm:"not null;constraint:OnDelete:CASCADE"`

	// Profile is the relationship to the profile
	Profile Profile `json:"profile" gorm:"foreignKey:ProfileID"`

	// UserID is the ID of the user for whom the key was created
	// For non-local users, this will be 0 since they don't have IDs
	UserID int64 `json:"user_id" gorm:"default:0"`

	// Username is the name of the user for whom the key was created
	Username string `json:"username" gorm:"size:255"`

	// UserUID is the UID of the user for whom the key was created
	UserUID *int64 `json:"user_uid"`

	// NonLocal indicates whether this key is for a non-local user
	// false for regular users, true for non-local users
	NonLocal bool `json:"non_local" gorm:"default:false"`

	// AccessKey is the generated access key
	AccessKey string `json:"access_key" gorm:"not null;size:255"`

	// SecretKey is the generated secret key
	SecretKey string `json:"secret_key" gorm:"not null;size:255"`

	// Optional: Add an index for faster lookups by user_id and profile_id
	// Index is created automatically by GORM for foreign keys
}

// ApiToken represents API tokens stored locally for tracking
type ApiToken struct {
	gorm.Model

	// ProfileID is the foreign key reference to the profile from which the token was created
	ProfileID uint `json:"profile_id" gorm:"not null;constraint:OnDelete:CASCADE"`

	// Profile is the relationship to the profile
	Profile Profile `json:"profile" gorm:"foreignKey:ProfileID"`

	// TokenID is the ID from the VAST system
	TokenID string `json:"token_id" gorm:"not null;size:255;index"`

	// Token is the actual token string
	Token string `json:"token" gorm:"not null;size:255"`

	// Name is the token name
	Name string `json:"name" gorm:"not null;size:255"`

	// Owner is the username of the token owner
	Owner string `json:"owner" gorm:"not null;size:255"`

	// OwnerID is the ID of the owner from the VAST system
	OwnerID uint `json:"owner_id" gorm:"not null"`

	// ExpireDate is when the token expires
	ExpireDate *time.Time `json:"expire_date"`

	// VastCreated is the creation timestamp from VAST (overrides gorm.Model CreatedAt)
	VastCreated time.Time `json:"vast_created" gorm:"not null"`
}

// SshConnection represents SSH connection configurations
type SshConnection struct {
	gorm.Model

	// Name is the arbitrary connection name (required, not empty)
	Name string `json:"name" gorm:"not null;size:255"`

	// SshHost is the hostname or IP address for SSH connection (required for non-local)
	SshHost string `json:"ssh_host" gorm:"size:255"`

	// SshUserName is the username for SSH connection (required)
	SshUserName string `json:"ssh_user_name" gorm:"not null;size:255"`

	// SshPassword is the password for SSH connection (optional if SshKey provided)
	SshPassword string `json:"ssh_password"`

	// SshKey is the absolute path to private SSH key (optional if SshPassword provided)
	SshKey string `json:"ssh_key"`

	// SshPort is the SSH port (optional, defaults to 22)
	SshPort int `json:"ssh_port" gorm:"default:22"`
}
