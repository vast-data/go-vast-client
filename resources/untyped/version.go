package untyped

import (
	"context"
	"strings"

	version "github.com/hashicorp/go-version"
	"github.com/vast-data/go-vast-client/core"
)

type Version struct {
	*core.VastResource
}

func (v *Version) GetVersionWithContext(ctx context.Context) (*version.Version, error) {
	result, err := v.ListWithContext(ctx, core.Params{"status": "success"})
	if err != nil {
		return nil, err
	}
	truncatedVersion, _ := sanitizeVersion(result[0]["sys_version"].(string))
	clusterVersion, err := version.NewVersion(truncatedVersion)
	if err != nil {
		return nil, err
	}
	//We only work with core version
	return clusterVersion.Core(), nil
}

func (v *Version) GetVersion() (*version.Version, error) {
	return v.GetVersionWithContext(v.Rest.GetCtx())
}

func (v *Version) CompareWithWithContext(ctx context.Context, other *version.Version) (int, error) {
	clusterVersion, err := v.GetVersionWithContext(ctx)
	if err != nil {
		return 0, err
	}
	return clusterVersion.Compare(other), nil
}

func (v *Version) CompareWith(other *version.Version) (int, error) {
	return v.CompareWithWithContext(v.Rest.GetCtx(), other)
}

// sanitizeVersion truncates all segments of Cluster Version above core (x.y.z)
// but preserves pre-release identifiers (e.g., 5.3.0-beta.1 stays as-is)
func sanitizeVersion(version string) (string, bool) {
	// First, separate the main version from pre-release and build metadata
	// Format: x.y.z[-prerelease][+buildmetadata]

	// Split on '+' to separate build metadata
	mainAndPrerelease := version
	buildMetadata := ""
	if plusIndex := strings.Index(version, "+"); plusIndex != -1 {
		mainAndPrerelease = version[:plusIndex]
		buildMetadata = version[plusIndex:]
	}

	// Split on '-' to separate pre-release
	mainVersion := mainAndPrerelease
	prerelease := ""
	if dashIndex := strings.Index(mainAndPrerelease, "-"); dashIndex != -1 {
		mainVersion = mainAndPrerelease[:dashIndex]
		prerelease = mainAndPrerelease[dashIndex:]
	}

	// Now split the main version on '.'
	segments := strings.Split(mainVersion, ".")
	truncated := len(segments) > 3 || buildMetadata != ""

	// Take only the first 3 segments of the main version
	var coreVersion string
	if len(segments) <= 3 {
		coreVersion = mainVersion
	} else {
		coreVersion = strings.Join(segments[:3], ".")
	}

	// Reconstruct version with pre-release but without build metadata
	result := coreVersion + prerelease
	return result, truncated
}
