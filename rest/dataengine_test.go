package rest_test

import (
	"context"
	"testing"

	"github.com/vast-data/go-vast-client/core"
	"github.com/vast-data/go-vast-client/rest"
	"github.com/vast-data/go-vast-client/rest/dataengine"
)

func TestDataEngineIsNestedVastRest(t *testing.T) {
	parent, err := rest.NewUntypedVMSRest(&core.VMSConfig{
		Host:     "example.invalid",
		Username: "u",
		Password: "p",
	})
	if err != nil {
		t.Fatalf("NewUntypedVMSRest: %v", err)
	}
	if parent.DataEngine == nil {
		t.Fatal("DataEngine nested rest is nil")
	}

	var asVastRest core.VastRest = parent.DataEngine
	_ = asVastRest
	if parent.DataEngine.GetSession() != parent.GetSession() {
		t.Fatal("DataEngine must share parent session")
	}
	if got := parent.DataEngine.GetApiRoot(); got != dataengine.ApiRoot {
		t.Fatalf("GetApiRoot() = %q, want %q", got, dataengine.ApiRoot)
	}
	if parent.DataEngine.Functions == nil {
		t.Fatal("Functions facade is nil")
	}
	if parent.DataEngine.Functions.Rest == nil || parent.DataEngine.Functions.Rest.GetApiRoot() != dataengine.ApiRoot {
		t.Fatal("Functions must inherit apiRoot from nested DataEngine rest")
	}
	if _, ok := parent.DataEngine.GetResourceMap()["Function"]; !ok {
		t.Fatal("Function not registered on DataEngine resource map")
	}
	if _, ok := parent.GetResourceMap()["Function"]; ok {
		t.Fatal("Function must not be registered on parent VMS resource map")
	}
}

func TestTypedDataEngineNestedRest(t *testing.T) {
	typed, err := rest.NewTypedVMSRest(&core.VMSConfig{
		Host:     "example.invalid",
		Username: "u",
		Password: "p",
	})
	if err != nil {
		t.Fatalf("NewTypedVMSRest: %v", err)
	}
	if typed.DataEngine == nil {
		t.Fatal("typed DataEngine is nil")
	}
	var _ core.VastRest = typed.DataEngine
	var _ *dataengine.TypedRest = typed.DataEngine
	if typed.DataEngine.Functions == nil {
		t.Fatal("typed Functions is nil")
	}
	if typed.DataEngine.Untyped != typed.Untyped.DataEngine {
		t.Fatal("typed DataEngine must wrap the same untyped nested rest")
	}
	if got := typed.GetApiRoot(); got != "" {
		t.Fatalf("TypedVMSRest.GetApiRoot() = %q, want empty VMS root", got)
	}
	if got := typed.DataEngine.GetApiRoot(); got != dataengine.ApiRoot {
		t.Fatalf("typed DataEngine.GetApiRoot() = %q, want %q", got, dataengine.ApiRoot)
	}
}

func TestSetCtxFansOutToDataEngine(t *testing.T) {
	type ctxKey string
	parent, err := rest.NewUntypedVMSRest(&core.VMSConfig{
		Host:     "example.invalid",
		Username: "u",
		Password: "p",
	})
	if err != nil {
		t.Fatalf("NewUntypedVMSRest: %v", err)
	}
	ctx := context.WithValue(context.Background(), ctxKey("de-test"), "ok")
	parent.SetCtx(ctx)
	if parent.GetCtx() != ctx {
		t.Fatal("parent ctx not updated")
	}
	if parent.DataEngine.GetCtx() != ctx {
		t.Fatal("DataEngine ctx must follow parent SetCtx")
	}

	typed, err := rest.NewTypedVMSRest(&core.VMSConfig{
		Host:     "example.invalid",
		Username: "u",
		Password: "p",
	})
	if err != nil {
		t.Fatalf("NewTypedVMSRest: %v", err)
	}
	typedCtx := context.WithValue(context.Background(), ctxKey("typed-de"), 1)
	typed.SetCtx(typedCtx)
	if typed.Untyped.DataEngine.GetCtx() != typedCtx {
		t.Fatal("typed SetCtx must fan out to nested DataEngine via Untyped.SetCtx")
	}
}
