package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vast-data/go-vast-client/core"
)

func TestUntypedVMSRest_Accessors(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	rest, err := NewUntypedVMSRest(testVMSConfig(t, server))
	if err != nil {
		t.Fatalf("NewUntypedVMSRest: %v", err)
	}

	if rest.GetSession() == nil {
		t.Fatal("expected session")
	}
	if rest.GetResourceMap() == nil {
		t.Fatal("expected resource map")
	}
	if _, ok := rest.GetResourceMap()["User"]; !ok {
		t.Fatal("expected User in resource map")
	}
}

func TestTypedVMSRest_GetResourceMap(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	typed, err := NewTypedVMSRest(testVMSConfig(t, server))
	if err != nil {
		t.Fatalf("NewTypedVMSRest: %v", err)
	}
	if typed.GetResourceMap() == nil {
		t.Fatal("expected resource map from typed client")
	}
}

func TestTypedVMSRest_ResourcesWired(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	typed, err := NewTypedVMSRest(testVMSConfig(t, server))
	if err != nil {
		t.Fatalf("NewTypedVMSRest: %v", err)
	}
	if typed.Users == nil {
		t.Fatal("expected Users resource")
	}
	if typed.Users.GetResourceType() != "User" {
		t.Fatalf("resource type = %q", typed.Users.GetResourceType())
	}
	if typed.Clusters == nil || typed.Quotas == nil || typed.Views == nil {
		t.Fatal("expected multiple typed resources to be wired")
	}
	if typed.GetSession() == nil {
		t.Fatal("expected session")
	}
	if typed.String() == "" {
		t.Fatal("expected non-empty String()")
	}
}

func TestNewTypedVMSRest_InvalidConfig(t *testing.T) {
	if _, err := NewTypedVMSRest(&core.VMSConfig{}); err == nil {
		t.Fatal("expected error for invalid config")
	}
}
