package widgets

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"

	vast_client "github.com/vast-data/go-vast-client"
)

type VipPool struct {
	*BaseWidget
}

func NewVipPool(db *database.Service) common.Widget {
	resourceType := "vippools"
	listHeaders := []string{"id", "role", "name", "ip_ranges", "tenant_name"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
		CustomInputs: []common.InputDefinition{
			{
				Name:        "ip_ranges",
				Type:        "array",
				Required:    true,
				Description: "List of IP ranges for the VIP pool",
				Placeholder: "Enter IP ranges (e.g., 10.0.0.1-10.0.0.20, ...)",
				Items: &common.InputDefinition{
					Type: "string",
				},
			},
		},
	}

	extraNav := []common.ExtraWidget{}

	widget := &VipPool{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (VipPool) API(rest *VMSRest) VastResourceAPI {
	return rest.VipPools
}

func (VipPool) BeforeCreate(params vast_client.Params) error {
	// Get ip_ranges from params
	ipRangesRaw, exists := params["ip_ranges"]
	if !exists {
		return nil // ip_ranges is optional, so no error if missing
	}

	// Convert to slice of strings
	var ipRangeStrings []string
	switch v := ipRangesRaw.(type) {
	case []interface{}:
		for _, item := range v {
			if str, ok := item.(string); ok && strings.TrimSpace(str) != "" {
				ipRangeStrings = append(ipRangeStrings, strings.TrimSpace(str))
			}
		}
	case []string:
		for _, str := range v {
			if strings.TrimSpace(str) != "" {
				ipRangeStrings = append(ipRangeStrings, strings.TrimSpace(str))
			}
		}
	case string:
		// Handle single string (shouldn't happen with array input, but just in case)
		if strings.TrimSpace(v) != "" {
			ipRangeStrings = append(ipRangeStrings, strings.TrimSpace(v))
		}
	default:
		return fmt.Errorf("ip_ranges must be an array of strings, got %T", ipRangesRaw)
	}

	if len(ipRangeStrings) == 0 {
		return nil // No ranges provided, that's okay
	}

	// Convert to slice of slices [start_ip, end_ip]
	var ipRanges [][]string
	for i, rangeStr := range ipRangeStrings {
		startIP, endIP, err := parseAndValidateIPRange(rangeStr)
		if err != nil {
			return fmt.Errorf("invalid IP range at index %d (%s): %v", i, rangeStr, err)
		}
		ipRanges = append(ipRanges, []string{startIP, endIP})
	}

	// Replace the original ip_ranges with the converted format
	params["ip_ranges"] = ipRanges
	return nil
}

// parseAndValidateIPRange parses a string like "10.0.0.1-10.0.0.20" and returns start, end IPs with validation
func parseAndValidateIPRange(rangeStr string) (string, string, error) {
	parts := strings.Split(rangeStr, "-")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("IP range must be in format 'start_ip-end_ip', got '%s'", rangeStr)
	}

	startIPStr := strings.TrimSpace(parts[0])
	endIPStr := strings.TrimSpace(parts[1])

	// Validate start IP
	startIP := net.ParseIP(startIPStr)
	if startIP == nil {
		return "", "", fmt.Errorf("invalid start IP address: '%s'", startIPStr)
	}

	// Validate end IP
	endIP := net.ParseIP(endIPStr)
	if endIP == nil {
		return "", "", fmt.Errorf("invalid end IP address: '%s'", endIPStr)
	}

	// Ensure both IPs are the same version (IPv4 or IPv6)
	if (startIP.To4() == nil) != (endIP.To4() == nil) {
		return "", "", fmt.Errorf("start and end IP must be the same version (both IPv4 or both IPv6)")
	}

	// Ensure start IP <= end IP
	if compareIPs(startIP, endIP) > 0 {
		return "", "", fmt.Errorf("start IP (%s) must be less than or equal to end IP (%s)", startIPStr, endIPStr)
	}

	return startIPStr, endIPStr, nil
}

// compareIPs compares two IP addresses, returns -1 if ip1 < ip2, 0 if equal, 1 if ip1 > ip2
func compareIPs(ip1, ip2 net.IP) int {
	// Convert to 16-byte representation for comparison
	ip1_16 := ip1.To16()
	ip2_16 := ip2.To16()

	for i := 0; i < 16; i++ {
		if ip1_16[i] < ip2_16[i] {
			return -1
		}
		if ip1_16[i] > ip2_16[i] {
			return 1
		}
	}
	return 0
}
