package vast_client

import (
	"strings"
	"testing"
)

func TestClientVersion(t *testing.T) {
	version := ClientVersion()

	// Version should not be empty
	if version == "" {
		t.Error("ClientVersion() should not return empty string")
	}

	// Version should not contain newlines (should be trimmed)
	if strings.Contains(version, "\n") || strings.Contains(version, "\r") {
		t.Error("ClientVersion() should not contain newline characters")
	}

	// Version should have some reasonable content
	if len(version) < 3 {
		t.Errorf("ClientVersion() = %v, seems too short for a version", version)
	}
}
