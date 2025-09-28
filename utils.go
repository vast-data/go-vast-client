package vast_client

import (
	"fmt"
	"net"
)


func toInt(val any) (int64, error) {
	var idInt int64
	switch v := val.(type) {
	case int64:
		idInt = v
	case float64:
		idInt = int64(v)
	case int:
		idInt = int64(v)
	default:
		return 0, fmt.Errorf("unexpected type for id field: %T", v)
	}
	return idInt, nil
}

func toRecord(m map[string]interface{}) (Record, error) {
	converted := Record{}
	for k, v := range m {
		converted[k] = v
	}
	return converted, nil
}

func toRecordSet(list []map[string]any) (RecordSet, error) {
	records := make(RecordSet, 0, len(list))
	for _, item := range list {
		rec, err := toRecord(item)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, nil
}

// contains checks if a string is present in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func generateIPRange(ipRanges [][2]string) ([]string, error) {
	ips := []string{}
	for _, r := range ipRanges {
		start := net.ParseIP(r[0]).To4()
		end := net.ParseIP(r[1]).To4()
		if start == nil || end == nil {
			return nil, fmt.Errorf("invalid IP in range: %v", r)
		}
		for ip := start; !ipGreaterThan(ip, end); ip = nextIP(ip) {
			ips = append(ips, ip.String())
		}
	}
	return ips, nil
}

func nextIP(ip net.IP) net.IP {
	newIP := make(net.IP, len(ip))
	copy(newIP, ip)
	for j := len(newIP) - 1; j >= 0; j-- {
		newIP[j]++
		if newIP[j] != 0 {
			break
		}
	}
	return newIP
}

func ipGreaterThan(a, b net.IP) bool {
	for i := 0; i < len(a); i++ {
		if a[i] > b[i] {
			return true
		} else if a[i] < b[i] {
			return false
		}
	}
	return false
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(fmt.Sprintf("must: %v", err))
	}
	return v
}

// buildResourcePathWithID builds a complete resource path with an ID parameter and optional additional segments.
// It takes a resource path (e.g., "/users"), an ID of any type, and optional additional path segments.
// Returns the complete path (e.g., "/users/123/tenant_data" or "/users/uuid/tenant_data").
func buildResourcePathWithID(resourcePath string, id any, additionalSegments ...string) string {
	var path string
	if intId, err := toInt(id); err == nil {
		path = fmt.Sprintf("%s/%d", resourcePath, intId)
	} else {
		path = fmt.Sprintf("%s/%v", resourcePath, id)
	}

	// Append additional segments if provided
	for _, segment := range additionalSegments {
		path += "/" + segment
	}

	return path
}
