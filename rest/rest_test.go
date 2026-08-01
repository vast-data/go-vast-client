package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vast-data/go-vast-client/core"
)

type mockRESTSession struct {
	config *core.VMSConfig
	auth   core.Authenticator
}

func (m *mockRESTSession) Get(context.Context, string, core.Params, []http.Header) (core.Renderable, error) {
	return nil, nil
}
func (m *mockRESTSession) Post(context.Context, string, core.Params, []http.Header) (core.Renderable, error) {
	return nil, nil
}
func (m *mockRESTSession) Put(context.Context, string, core.Params, []http.Header) (core.Renderable, error) {
	return nil, nil
}
func (m *mockRESTSession) Patch(context.Context, string, core.Params, []http.Header) (core.Renderable, error) {
	return nil, nil
}
func (m *mockRESTSession) Delete(context.Context, string, core.Params, []http.Header) (core.Renderable, error) {
	return nil, nil
}
func (m *mockRESTSession) GetConfig() *core.VMSConfig { return m.config }
func (m *mockRESTSession) GetAuthenticator() core.Authenticator {
	return m.auth
}

func parseTestServerAddress(addr string) (host string, port uint64) {
	lastColon := strings.LastIndex(addr, ":")
	if lastColon == -1 {
		return addr, 443
	}
	host = addr[:lastColon]
	portNum, _ := strconv.ParseUint(addr[lastColon+1:], 10, 64)
	return host, portNum
}

func testVMSConfig(t *testing.T, server *httptest.Server) *core.VMSConfig {
	t.Helper()
	host, port := parseTestServerAddress(server.Listener.Addr().String())
	timeout := time.Minute
	return &core.VMSConfig{
		Host:           host,
		Port:           port,
		ApiToken:       "test-token",
		SslVerify:      false,
		Timeout:        &timeout,
		MaxConnections: 5,
		ApiVersion:     "latest",
		UserAgent:      "go-vast-client-test/1.0",
	}
}

func TestNewUntypedVMSRest_InvalidConfig(t *testing.T) {
	_, err := NewUntypedVMSRest(&core.VMSConfig{})
	if err == nil {
		t.Fatal("expected error for empty config")
	}
}

func TestNewUntypedVMSRest_RegistersResources(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	rest, err := NewUntypedVMSRest(testVMSConfig(t, server))
	if err != nil {
		t.Fatalf("NewUntypedVMSRest: %v", err)
	}

	if rest.Users == nil {
		t.Fatal("Users resource not initialized")
	}
	if _, ok := rest.resourceMap["User"]; !ok {
		t.Fatal("User not registered in resource map")
	}
	if got := rest.GetCtx(); got == nil {
		t.Fatal("expected default context")
	}
}

func TestNewUntypedVMSRest_UsesProvidedContext(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := testVMSConfig(t, server)
	ctx := context.WithValue(context.Background(), struct{}{}, "test")
	cfg.Context = ctx

	rest, err := NewUntypedVMSRest(cfg)
	if err != nil {
		t.Fatalf("NewUntypedVMSRest: %v", err)
	}
	if rest.GetCtx() != ctx {
		t.Fatal("expected provided context to be set")
	}
}

func TestUntypedVMSRest_SetCtx(t *testing.T) {
	rest := &UntypedVMSRest{}
	ctx := context.WithValue(context.Background(), struct{}{}, "updated")
	rest.SetCtx(ctx)
	if rest.GetCtx() != ctx {
		t.Fatal("SetCtx did not update context")
	}
}

func TestUntypedVMSRest_String(t *testing.T) {
	cfg := &core.VMSConfig{Host: "vms.example.com"}
	tests := []struct {
		name string
		auth core.Authenticator
		want string
	}{
		{
			name: "api token",
			auth: &core.ApiRTokenAuthenticator{Tenant: "tenant-a"},
			want: "vms.example.com [type=api-token;tenant=tenant-a]",
		},
		{
			name: "api token without tenant",
			auth: &core.ApiRTokenAuthenticator{},
			want: "vms.example.com [type=api-token]",
		},
		{
			name: "basic auth",
			auth: &core.BaseAuthAuthenticator{Username: "admin", Tenant: "tenant-b"},
			want: "vms.example.com [type=basic-auth;user=admin;tenant=tenant-b]",
		},
		{
			name: "jwt",
			auth: &core.JWTAuthenticator{Username: "svc", Tenant: "tenant-c"},
			want: "vms.example.com [type=bearer-token;user=svc;tenant=tenant-c]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rest := &UntypedVMSRest{
				Session: &mockRESTSession{config: cfg, auth: tt.auth},
			}
			if got := rest.String(); got != tt.want {
				t.Fatalf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUntypedVMSRest_String_PanicsOnUnknownAuthenticator(t *testing.T) {
	rest := &UntypedVMSRest{
		Session: &mockRESTSession{
			config: &core.VMSConfig{Host: "vms.example.com"},
			auth:   nil,
		},
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for unknown authenticator")
		}
	}()
	_ = rest.String()
}

func TestNewTypedVMSRest_WiresResources(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	typed, err := NewTypedVMSRest(testVMSConfig(t, server))
	if err != nil {
		t.Fatalf("NewTypedVMSRest: %v", err)
	}

	if typed.Users == nil {
		t.Fatal("typed Users resource not initialized")
	}
	if typed.Untyped == nil {
		t.Fatal("typed client missing untyped backing client")
	}
	if typed.GetSession() == nil {
		t.Fatal("expected session from typed client")
	}

	ctx := context.WithValue(context.Background(), struct{}{}, "typed")
	typed.SetCtx(ctx)
	if typed.GetCtx() != ctx {
		t.Fatal("typed SetCtx did not propagate to untyped client")
	}
	if _, ok := typed.GetResourceMap()["User"]; !ok {
		t.Fatal("typed client resource map missing User")
	}
	if typed.String() == "" {
		t.Fatal("typed String() returned empty value")
	}
}
