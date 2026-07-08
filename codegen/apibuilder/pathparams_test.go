package apibuilder

import "testing"

func TestExtractPathParams(t *testing.T) {
	params := ExtractPathParams("/tenants/{tenant_id}/metric_label_values/{id}/")
	if len(params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(params))
	}
	if params[0].Name != "tenant_id" || params[0].GoName != "tenantId" {
		t.Fatalf("unexpected first param: %+v", params[0])
	}
	if params[1].Name != "id" || params[1].GoName != "id" {
		t.Fatalf("unexpected second param: %+v", params[1])
	}
}

func TestAnalyzePathBuild(t *testing.T) {
	single := AnalyzePathBuild("/activedirectory/{id}/refresh/")
	if !single.UseBuildResourcePathWithID {
		t.Fatal("expected BuildResourcePathWithID for single id param")
	}
	if single.ResourcePath != "activedirectory" || len(single.SubPathSegments) != 1 || single.SubPathSegments[0] != "refresh" {
		t.Fatalf("unexpected single-param build info: %+v", single)
	}

	tenant := AnalyzePathBuild("/tenants/{tenant_id}/metric_label_values/bulk/")
	if !tenant.UseBuildResourcePathWithID {
		t.Fatal("expected BuildResourcePathWithID for single tenant_id param")
	}
	if tenant.ResourcePath != "tenants" || len(tenant.SubPathSegments) != 2 {
		t.Fatalf("unexpected tenant build info: %+v", tenant)
	}

	multi := AnalyzePathBuild("/tenants/{tenant_id}/metric_label_values/{id}/")
	if multi.UseBuildResourcePathWithID {
		t.Fatal("expected InterpolatePathTemplate for multiple params")
	}
	if len(multi.PathParams) != 2 {
		t.Fatalf("expected 2 path params, got %+v", multi.PathParams)
	}
}
