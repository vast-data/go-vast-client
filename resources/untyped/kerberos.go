package untyped

import (
	"context"
	"net/http"

	"github.com/vast-data/go-vast-client/core"
)

type Kerberos struct {
	*core.VastResource
}

// KerberosKeytabWithContext_PUT
// method: PUT
// url: /kerberos/{id}/keytab/
// summary: Upload keytab for the Kerberos Provider
func (k *Kerberos) KerberosKeytabWithContext_PUT(ctx context.Context, kerberosId any, keytabFile []byte, filename string) (core.Record, error) {
	path := core.BuildResourcePathWithID(k.GetResourcePath(), kerberosId, "keytab")

	// Prepare multipart form data using Params
	body := core.Params{
		"keytab_file": core.FileData{
			Filename: filename,
			Content:  keytabFile,
		},
	}

	// Create headers to indicate multipart/form-data
	headers := []http.Header{{
		core.HeaderContentType: []string{core.ContentTypeMultipartForm},
	}}

	return core.RequestWithHeaders[core.Record](ctx, k, http.MethodPut, path, nil, body, headers)
}

// KerberosKeytab_PUT
// method: PUT
// url: /kerberos/{id}/keytab/
// summary: Upload keytab for the Kerberos Provider
func (k *Kerberos) KerberosKeytab_PUT(kerberosId any, keytabFile []byte, filename string) (core.Record, error) {
	return k.KerberosKeytabWithContext_PUT(k.Rest.GetCtx(), kerberosId, keytabFile, filename)

}
