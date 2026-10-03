package dataengine_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vast-data/go-vast-client/core"
	"github.com/vast-data/go-vast-client/rest"
)

func testDEConfig(t *testing.T, server *httptest.Server) *core.VMSConfig {
	t.Helper()
	addr := server.Listener.Addr().String()
	lastColon := strings.LastIndex(addr, ":")
	host, portStr := addr[:lastColon], addr[lastColon+1:]
	port, _ := strconv.ParseUint(portStr, 10, 64)
	timeout := time.Minute
	return &core.VMSConfig{
		Host:       host,
		Port:       port,
		ApiToken:   "test-token",
		SslVerify:  false,
		Timeout:    &timeout,
		ApiVersion: "latest",
	}
}

func TestElementTriggerListUsesTriggersCollection(t *testing.T) {
	var gotPath, gotType string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotType = r.URL.Query().Get("type")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"guid": "g1", "name": "el", "type": "Element"},
			},
			"pagination": map[string]any{},
		})
	}))
	defer srv.Close()

	parent, err := rest.NewUntypedVMSRest(testDEConfig(t, srv))
	if err != nil {
		t.Fatalf("NewUntypedVMSRest: %v", err)
	}

	recs, err := parent.DataEngine.ElementTriggers.List(core.Params{"tenant_name": "de-lab"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if gotPath != "/api/latest/serverless/triggers/" {
		t.Fatalf("path = %q, want /api/latest/serverless/triggers/", gotPath)
	}
	if gotType != "Element" {
		t.Fatalf("type = %q, want Element", gotType)
	}
	if len(recs) != 1 || recs[0]["name"] != "el" {
		t.Fatalf("unexpected records: %#v", recs)
	}
}

func TestScheduleTriggerGetByIdUsesTriggersGUID(t *testing.T) {
	var gotPath string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(map[string]any{
			"guid": "abc", "name": "sched", "type": "Schedule",
		})
	}))
	defer srv.Close()

	parent, err := rest.NewUntypedVMSRest(testDEConfig(t, srv))
	if err != nil {
		t.Fatalf("NewUntypedVMSRest: %v", err)
	}

	rec, err := parent.DataEngine.ScheduleTriggers.GetById("abc", core.Params{"tenant_name": "de-lab"})
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if gotPath != "/api/latest/serverless/triggers/abc/" && gotPath != "/api/latest/serverless/triggers/abc" {
		t.Fatalf("path = %q, want triggers/abc", gotPath)
	}
	if rec["name"] != "sched" {
		t.Fatalf("unexpected record: %#v", rec)
	}
}

func TestKubernetesSecretListIsEmpty(t *testing.T) {
	parent, err := rest.NewUntypedVMSRest(&core.VMSConfig{
		Host:     "example.invalid",
		Username: "u",
		Password: "p",
	})
	if err != nil {
		t.Fatalf("NewUntypedVMSRest: %v", err)
	}
	recs, err := parent.DataEngine.KubernetesSecrets.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(recs) != 0 {
		t.Fatalf("want empty list, got %#v", recs)
	}
}

func TestElementTriggerCreateStillUsesElementPath(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		_ = json.NewEncoder(w).Encode(map[string]any{
			"guid": "g1", "name": "el", "type": "Element",
		})
	}))
	defer srv.Close()

	parent, err := rest.NewUntypedVMSRest(testDEConfig(t, srv))
	if err != nil {
		t.Fatalf("NewUntypedVMSRest: %v", err)
	}

	_, err = parent.DataEngine.ElementTriggers.Create(
		core.Params{"tenant_name": "de-lab"},
		core.Params{"name": "el", "config": map[string]any{"events": []string{"ObjectCreated:*"}}},
	)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/api/latest/serverless/triggers/element/" && gotPath != "/api/latest/serverless/triggers/element" {
		t.Fatalf("path = %q, want triggers/element", gotPath)
	}
}
