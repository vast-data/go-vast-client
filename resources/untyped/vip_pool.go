package untyped

import (
	"context"
	"fmt"
	"net"

	"github.com/vast-data/go-vast-client/core"
)

type VipPool struct {
	*core.VastResource
}

func (v *VipPool) IpRangeForWithContext(ctx context.Context, name string) ([]string, error) {
	result, err := v.GetWithContext(ctx, core.Params{"name": name})
	if err != nil {
		return nil, err
	}
	var ipRanges struct {
		IpRanges [][2]string `json:"ip_ranges"`
	}
	if err = result.Fill(&ipRanges); err != nil {
		return nil, err
	}
	return generateIPRange(ipRanges.IpRanges)
}

func (v *VipPool) IpRangeFor(name string) ([]string, error) {
	return v.IpRangeForWithContext(v.Rest.GetCtx(), name)
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
