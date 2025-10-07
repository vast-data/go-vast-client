package untyped

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"reflect"

	"github.com/vast-data/go-vast-client/core"
)

type Monitor struct {
	*core.VastResource
}

// convertMonitorParams converts params for monitor queries, specifically handling
// prop_list as multiple query parameters instead of comma-separated values
func convertMonitorParams(params core.Params) core.Params {
	if params == nil {
		return params
	}

	// Check if prop_list exists and is a slice/array
	propList, exists := params["prop_list"]
	if !exists {
		return params
	}

	rv := reflect.ValueOf(propList)
	if !rv.IsValid() || (rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array) {
		return params
	}

	// prop_list is a slice/array, so we need special handling
	// We'll build the query string manually using url.Values
	values := url.Values{}

	// Add all other params first
	for k, v := range params {
		if k == "prop_list" {
			continue // handle separately
		}
		values.Set(k, fmt.Sprint(v))
	}

	// Add each prop_list element as a separate parameter
	n := rv.Len()
	for i := 0; i < n; i++ {
		values.Add("prop_list", fmt.Sprint(rv.Index(i).Interface()))
	}

	// Return a special marker params that will be handled differently
	// We'll encode the query string and pass it through
	return core.Params{
		"@preencoded_query": values.Encode(),
	}
}

// MonitorAdHocQueryWithContext_GET
// method: GET
// url: /monitors/ad_hoc_query/
// summary: Query Analytics with Ad Hoc Query Parameters
func (m *Monitor) MonitorAdHocQueryWithContext_GET(ctx context.Context, params core.Params) (core.Record, error) {
	// Convert params to handle prop_list properly
	convertedParams := convertMonitorParams(params)

	// Check if we have a pre-encoded query string
	if preencoded, ok := convertedParams["@preencoded_query"]; ok {
		// Build URL manually with pre-encoded query
		resourcePath := "/monitors/ad_hoc_query/?" + preencoded.(string)

		// Use session.Get directly instead of core.Request
		if ctx == nil {
			ctx = context.Background()
		}
		result, err := m.Session().Get(ctx, resourcePath, nil, nil)
		if err != nil {
			return nil, err
		}

		// The result should already be a Record
		if record, ok := result.(core.Record); ok {
			return record, nil
		}
		return nil, fmt.Errorf("unexpected response type: %T", result)
	}

	// Fallback to normal behavior if no conversion was needed
	resourcePath := "/monitors/ad_hoc_query/"
	result, err := core.Request[core.Record](ctx, m, http.MethodGet, resourcePath, convertedParams, nil)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// MonitorAdHocQuery_GET
// method: GET
// url: /monitors/ad_hoc_query/
// summary: Query Analytics with Ad Hoc Query Parameters
func (m *Monitor) MonitorAdHocQuery_GET(params core.Params) (core.Record, error) {
	return m.MonitorAdHocQueryWithContext_GET(m.Rest.GetCtx(), params)
}
