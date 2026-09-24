package openapi_schema_test

import (
	"testing"

	"github.com/vast-data/go-vast-client/openapi_schema"
)

func TestGetResponseModelSchema_DataEngineUnwrapsDataItems(t *testing.T) {
	schema, err := openapi_schema.GetResponseModelSchema(
		openapi_schema.PackDataEngine,
		"GET",
		"mtls-authentication-credentials",
	)
	if err != nil {
		t.Fatalf("GetResponseModelSchema: %v", err)
	}
	if schema == nil || schema.Value == nil {
		t.Fatal("expected resolved item schema")
	}
	// Item model must be the credential object, not the list envelope.
	if _, ok := schema.Value.Properties["pagination"]; ok {
		t.Fatal("schema still looks like list envelope (has pagination)")
	}
	if _, ok := schema.Value.Properties["data"]; ok {
		t.Fatal("schema still looks like list envelope (has data)")
	}
	if _, ok := schema.Value.Properties["name"]; !ok {
		t.Fatalf("expected credential field name on item schema, props=%v", schema.Value.Properties)
	}
}

func TestGetResponseModelSchemaUnresolved_DataEngineDataItemsRef(t *testing.T) {
	schemaRef, err := openapi_schema.GetResponseModelSchemaUnresolved(
		openapi_schema.PackDataEngine,
		"GET",
		"mtls-authentication-credentials",
	)
	if err != nil {
		t.Fatalf("GetResponseModelSchemaUnresolved: %v", err)
	}
	name := openapi_schema.IsDirectComponentReference(schemaRef)
	if name != "MTLSAuthenticationCredentials" {
		t.Fatalf("expected items $ref MTLSAuthenticationCredentials, got %q (ref=%q)", name, schemaRef.Ref)
	}
}
