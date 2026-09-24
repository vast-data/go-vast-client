package openapi_schema

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
)

// Pack identifies an embedded OpenAPI document.
type Pack string

const (
	// PackVMS is the classic VMS Management OpenAPI pack (api.tar.gz).
	PackVMS Pack = "vms"
	// PackDataEngine is the DataEngine / serverless provisioning OpenAPI pack (dataengine.tar.gz).
	PackDataEngine Pack = "dataengine"
)

var packArchives = map[Pack]string{
	PackVMS:        "api.tar.gz",
	PackDataEngine: "dataengine.tar.gz",
}

type packState struct {
	once sync.Once
	doc  *openapi3.T
	err  error
}

var packs = map[Pack]*packState{
	PackVMS:        {},
	PackDataEngine: {},
}

// loadOpenAPIDoc loads and caches the OpenAPI document for the given pack.
func loadOpenAPIDoc(pack Pack) (*openapi3.T, error) {
	state, ok := packs[pack]
	if !ok {
		return nil, fmt.Errorf("unknown OpenAPI pack %q (known: %s, %s)", pack, PackVMS, PackDataEngine)
	}
	archive, ok := packArchives[pack]
	if !ok {
		return nil, fmt.Errorf("no archive registered for OpenAPI pack %q", pack)
	}

	state.once.Do(func() {
		state.doc, state.err = loadOpenAPIDocFromArchive(archive)
	})
	return state.doc, state.err
}

func loadOpenAPIDocFromArchive(archiveName string) (*openapi3.T, error) {
	data, err := FS.ReadFile(archiveName)
	if err != nil {
		return nil, fmt.Errorf("read embedded %s: %w", archiveName, err)
	}

	gzr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gzip reader (%s): %w", archiveName, err)
	}
	defer func() { _ = gzr.Close() }()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("api.json not found in embedded archive %s", archiveName)
		}
		if err != nil {
			return nil, fmt.Errorf("tar read error (%s): %w", archiveName, err)
		}

		if !strings.HasSuffix(hdr.Name, "api.json") {
			continue
		}

		var buf bytes.Buffer
		const maxOpenAPIJSON = 32 << 20
		if _, err := io.Copy(&buf, io.LimitReader(tr, maxOpenAPIJSON+1)); err != nil { // #nosec G110 -- extract capped; archive is our embed
			return nil, fmt.Errorf("copy api.json from %s: %w", archiveName, err)
		}
		if buf.Len() > maxOpenAPIJSON {
			return nil, fmt.Errorf("api.json in %s exceeds %d byte extract limit", archiveName, maxOpenAPIJSON)
		}

		loader := openapi3.NewLoader()
		doc, err := loader.LoadFromData(buf.Bytes())
		if err != nil {
			return nil, fmt.Errorf("parse api.json from %s: %w", archiveName, err)
		}
		return doc, nil
	}
}
