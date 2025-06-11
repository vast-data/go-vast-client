package vast_client

import (
	"fmt"
	"net"
)

const ApplicationJson = "application/json"

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
	var ips []string
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
