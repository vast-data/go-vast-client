package client

import (
	"context"
	"fmt"
	"net/http"

	vastclient "github.com/vast-data/go-vast-client"
)

// VastVersionUnavailable is stored on a profile when Versions is forbidden (e.g. tenant admin).
const VastVersionUnavailable = "<n/a>"

// FetchVastVersion returns the cluster version string for display on a profile.
// A 403 from Versions is treated as non-fatal and returns VastVersionUnavailable —
// auth already succeeded when the REST client was created.
func FetchVastVersion(ctx context.Context, rest *vastclient.VMSRest) (string, error) {
	if rest == nil {
		return "", fmt.Errorf("rest client is nil")
	}
	version, err := rest.Versions.GetVersionWithContext(ctx)
	if err != nil {
		if vastclient.ExpectStatusCodes(err, http.StatusForbidden) {
			return VastVersionUnavailable, nil
		}
		return "", err
	}
	return version.String(), nil
}
