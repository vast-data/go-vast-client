//go:build ignore
// +build ignore

// pack_oas3 packs an already-OpenAPI-3 document into api.json inside a .tar.gz.
// Used when the input is OpenAPI 3.x (e.g. DataEngine provisioning.yaml) and
// swagger2→v3 conversion must be skipped.
//
// Env:
//
//	INPUT_PATH  - path to OpenAPI 3 YAML or JSON (required)
//	OUTPUT_DIR  - directory for api.json and the tarball (default /tmp/apiconv)
//	OUTPUT_TAR  - tarball filename inside OUTPUT_DIR (default api.tar.gz)
package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

func main() {
	inputPath := os.Getenv("INPUT_PATH")
	if inputPath == "" {
		log.Fatal("❌ INPUT_PATH is required")
	}
	outputDir := os.Getenv("OUTPUT_DIR")
	if outputDir == "" {
		outputDir = "/tmp/apiconv"
	}
	tarName := os.Getenv("OUTPUT_TAR")
	if tarName == "" {
		tarName = "api.tar.gz"
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("❌ Failed to create output directory: %v", err)
	}

	doc, err := loadOpenAPI3(inputPath)
	if err != nil {
		log.Fatalf("❌ Failed to load OpenAPI 3 document: %v", err)
	}

	// Inline external $refs (e.g. common.yaml) so the embedded pack is self-contained.
	doc.InternalizeRefs(context.Background(), nil)
	log.Println("✅ Internalized external $refs into components")

	jsonOut := filepath.Join(outputDir, "api.json")
	f, err := os.Create(jsonOut)
	if err != nil {
		log.Fatalf("❌ Failed to create api.json: %v", err)
	}
	enc := json.NewEncoder(f)
	// Compact JSON to match convert_to_v3.go output style.
	if err := enc.Encode(doc); err != nil {
		_ = f.Close()
		log.Fatalf("❌ Failed to write api.json: %v", err)
	}
	if err := f.Close(); err != nil {
		log.Fatalf("❌ Failed to close api.json: %v", err)
	}
	log.Println("✅ Saved OpenAPI v3 to", jsonOut)

	tarOut := filepath.Join(outputDir, tarName)
	if err := writeTarGz(tarOut, jsonOut, "api.json"); err != nil {
		log.Fatalf("❌ Failed to create tarball: %v", err)
	}
	log.Println("✅ Created", tarOut)
}

func loadOpenAPI3(path string) (*openapi3.T, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".json") {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return loader.LoadFromData(data)
	}

	// LoadFromFile resolves relative external $refs (e.g. common.yaml) against the file path.
	doc, err := loader.LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("load %s (external refs allowed): %w", path, err)
	}
	return doc, nil
}

func writeTarGz(tarPath, filePath, nameInTar string) error {
	tarFile, err := os.Create(tarPath)
	if err != nil {
		return err
	}
	defer tarFile.Close()

	gzw := gzip.NewWriter(tarFile)
	defer gzw.Close()
	tw := tar.NewWriter(gzw)
	defer tw.Close()

	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	hdr, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	hdr.Name = nameInTar
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	_, err = tw.Write(content)
	return err
}
