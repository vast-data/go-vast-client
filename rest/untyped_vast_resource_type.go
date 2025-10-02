package rest

import (
	"github.com/vast-data/go-vast-client/core"
	"github.com/vast-data/go-vast-client/resources/untyped"
)

type UntypedVastResourceType interface {
	core.Dummy |
		untyped.Version |
		untyped.Quota |
		untyped.View |
		untyped.VipPool |
		untyped.User |
		untyped.UserKey |
		untyped.Snapshot |
		untyped.BlockHost |
		untyped.Volume |
		untyped.VTask |
		untyped.BlockHostMapping |
		untyped.Cnode |
		untyped.QosPolicy |
		untyped.Dns |
		untyped.ViewPolicy |
		untyped.Group |
		untyped.Nis |
		untyped.Tenant |
		untyped.Ldap |
		untyped.S3LifeCycleRule |
		untyped.ActiveDirectory |
		untyped.S3Policy |
		untyped.ProtectedPath |
		untyped.GlobalSnapshotStream |
		untyped.ReplicationPeers |
		untyped.ProtectionPolicy |
		untyped.S3replicationPeers |
		untyped.Realm |
		untyped.Role |
		untyped.NonLocalUser |
		untyped.NonLocalGroup |
		untyped.NonLocalUserKey |
		untyped.ApiToken |
		untyped.KafkaBroker |
		untyped.Manager |
		untyped.Folder |
		untyped.EventDefinition |
		untyped.EventDefinitionConfig |
		untyped.BGPConfig |
		untyped.Vms |
		untyped.Topic |
		untyped.LocalProvider |
		untyped.LocalS3Key |
		untyped.EncryptionGroup |
		untyped.SamlConfig |
		untyped.Kerberos |
		untyped.Cluster |
		untyped.SupportedDrivers |
		untyped.Rack |
		untyped.Fan |
		untyped.Nic |
		untyped.NicPort |
		untyped.IamRole |
		untyped.Oidc |
		untyped.Vip
}
