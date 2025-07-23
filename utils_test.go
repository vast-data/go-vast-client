package vast_client

import (
	"fmt"
	"net"
	"reflect"
	"testing"
)

func TestToInt(t *testing.T) {
	tests := []struct {
		name    string
		val     any
		want    int64
		wantErr bool
	}{
		{
			name:    "int64 value",
			val:     int64(123),
			want:    123,
			wantErr: false,
		},
		{
			name:    "float64 value",
			val:     float64(123.0),
			want:    123,
			wantErr: false,
		},
		{
			name:    "int value",
			val:     int(123),
			want:    123,
			wantErr: false,
		},
		{
			name:    "float64 with decimals",
			val:     float64(123.7),
			want:    123,
			wantErr: false,
		},
		{
			name:    "string value - should error",
			val:     "123",
			want:    0,
			wantErr: true,
		},
		{
			name:    "nil value - should error",
			val:     nil,
			want:    0,
			wantErr: true,
		},
		{
			name:    "bool value - should error",
			val:     true,
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toInt(tt.val)
			if (err != nil) != tt.wantErr {
				t.Errorf("toInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("toInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToRecord(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		want Record
	}{
		{
			name: "empty map",
			m:    map[string]interface{}{},
			want: Record{},
		},
		{
			name: "simple map",
			m: map[string]interface{}{
				"id":   123,
				"name": "test",
			},
			want: Record{
				"id":   123,
				"name": "test",
			},
		},
		{
			name: "complex map",
			m: map[string]interface{}{
				"id":     int64(123),
				"name":   "test",
				"active": true,
				"data":   []string{"a", "b", "c"},
			},
			want: Record{
				"id":     int64(123),
				"name":   "test",
				"active": true,
				"data":   []string{"a", "b", "c"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toRecord(tt.m)
			if err != nil {
				t.Errorf("toRecord() error = %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toRecord() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToRecordSet(t *testing.T) {
	tests := []struct {
		name    string
		list    []map[string]any
		want    RecordSet
		wantErr bool
	}{
		{
			name: "empty list",
			list: []map[string]any{},
			want: RecordSet{},
		},
		{
			name: "single record",
			list: []map[string]any{
				{"id": 1, "name": "test1"},
			},
			want: RecordSet{
				{"id": 1, "name": "test1"},
			},
		},
		{
			name: "multiple records",
			list: []map[string]any{
				{"id": 1, "name": "test1"},
				{"id": 2, "name": "test2"},
			},
			want: RecordSet{
				{"id": 1, "name": "test1"},
				{"id": 2, "name": "test2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toRecordSet(tt.list)
			if (err != nil) != tt.wantErr {
				t.Errorf("toRecordSet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toRecordSet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		item  string
		want  bool
	}{
		{
			name:  "item found",
			slice: []string{"a", "b", "c"},
			item:  "b",
			want:  true,
		},
		{
			name:  "item not found",
			slice: []string{"a", "b", "c"},
			item:  "d",
			want:  false,
		},
		{
			name:  "empty slice",
			slice: []string{},
			item:  "a",
			want:  false,
		},
		{
			name:  "exact match",
			slice: []string{"test"},
			item:  "test",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contains(tt.slice, tt.item); got != tt.want {
				t.Errorf("contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateIPRange(t *testing.T) {
	tests := []struct {
		name     string
		ipRanges [][2]string
		want     []string
		wantErr  bool
	}{
		{
			name:     "single IP range",
			ipRanges: [][2]string{{"192.168.1.1", "192.168.1.3"}},
			want:     []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"},
			wantErr:  false,
		},
		{
			name:     "single IP",
			ipRanges: [][2]string{{"192.168.1.1", "192.168.1.1"}},
			want:     []string{"192.168.1.1"},
			wantErr:  false,
		},
		{
			name:     "invalid IP in range",
			ipRanges: [][2]string{{"invalid", "192.168.1.3"}},
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "empty range",
			ipRanges: [][2]string{},
			want:     []string{},
			wantErr:  false,
		},
		{
			name: "multiple ranges",
			ipRanges: [][2]string{
				{"192.168.1.1", "192.168.1.2"},
				{"10.0.0.1", "10.0.0.1"},
			},
			want:    []string{"192.168.1.1", "192.168.1.2", "10.0.0.1"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generateIPRange(tt.ipRanges)
			if (err != nil) != tt.wantErr {
				t.Errorf("generateIPRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("generateIPRange() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNextIP(t *testing.T) {
	tests := []struct {
		name string
		ip   net.IP
		want net.IP
	}{
		{
			name: "increment last octet",
			ip:   net.ParseIP("192.168.1.1").To4(),
			want: net.ParseIP("192.168.1.2").To4(),
		},
		{
			name: "overflow last octet",
			ip:   net.ParseIP("192.168.1.255").To4(),
			want: net.ParseIP("192.168.2.0").To4(),
		},
		{
			name: "overflow multiple octets",
			ip:   net.ParseIP("192.168.255.255").To4(),
			want: net.ParseIP("192.169.0.0").To4(),
		},
		{
			name: "overflow all octets",
			ip:   net.ParseIP("255.255.255.255").To4(),
			want: net.ParseIP("0.0.0.0").To4(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nextIP(tt.ip)
			if !got.Equal(tt.want) {
				t.Errorf("nextIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIpGreaterThan(t *testing.T) {
	tests := []struct {
		name string
		a    net.IP
		b    net.IP
		want bool
	}{
		{
			name: "a greater than b",
			a:    net.ParseIP("192.168.1.2").To4(),
			b:    net.ParseIP("192.168.1.1").To4(),
			want: true,
		},
		{
			name: "a less than b",
			a:    net.ParseIP("192.168.1.1").To4(),
			b:    net.ParseIP("192.168.1.2").To4(),
			want: false,
		},
		{
			name: "a equal to b",
			a:    net.ParseIP("192.168.1.1").To4(),
			b:    net.ParseIP("192.168.1.1").To4(),
			want: false,
		},
		{
			name: "different octets",
			a:    net.ParseIP("192.169.0.0").To4(),
			b:    net.ParseIP("192.168.1.1").To4(),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ipGreaterThan(tt.a, tt.b); got != tt.want {
				t.Errorf("ipGreaterThan() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMust(t *testing.T) {
	t.Run("returns value when no error", func(t *testing.T) {
		result := must("test", nil)
		if result != "test" {
			t.Errorf("must() = %v, want test", result)
		}
	})

	t.Run("panics when error provided", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("must() should have panicked")
			}
		}()
		must("test", fmt.Errorf("test error"))
	})

	t.Run("works with different types", func(t *testing.T) {
		intResult := must(42, nil)
		if intResult != 42 {
			t.Errorf("must() = %v, want 42", intResult)
		}

		boolResult := must(true, nil)
		if !boolResult {
			t.Errorf("must() = %v, want true", boolResult)
		}
	})
}
