package openapi_schema_test

import (
	"testing"

	"github.com/vast-data/go-vast-client/openapi_schema"
)

func TestDataEnginePackLoads(t *testing.T) {
	paths, err := openapi_schema.GetAllPaths(openapi_schema.PackDataEngine)
	if err != nil {
		t.Fatalf("GetAllPaths(dataengine): %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("expected DataEngine paths, got none")
	}
	for _, want := range []string{
		"/functions",
		"/pipelines",
		"/container-registries",
		"/kubernetes-clusters",
		"/triggers/schedule",
		"/triggers/element",
	} {
		if _, ok := paths[want]; !ok {
			// Some specs use trailing slash variants.
			if _, ok2 := paths[want+"/"]; !ok2 {
				t.Errorf("missing DataEngine path %s (have %d paths)", want, len(paths))
			}
		}
	}
}

func TestVMSPackStillLoads(t *testing.T) {
	paths, err := openapi_schema.GetAllPaths(openapi_schema.PackVMS)
	if err != nil {
		t.Fatalf("GetAllPaths(vms): %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("expected VMS paths, got none")
	}
}
