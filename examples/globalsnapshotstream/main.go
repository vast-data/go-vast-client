package main

import (
	"fmt"
	client "github.com/vast-data/go-vast-client"
)

type ViewContainer struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	TenantID int64  `json:"tenant_id"`
}

func main() {
	var view ViewContainer
	config := &client.VMSConfig{
		Host:     "10.27.40.1", // replace with your VAST address
		Username: "admin",
		Password: "123456",
	}

	rest, err := client.NewVMSRest(config)
	if err != nil {
		panic(err)
	}

	// Create view
	viewName := "sourceview"
	fmt.Printf("Ensuring view '%s'...\n", viewName)
	viewCreateParams := client.Params{
		"path":       "/sourceview",
		"create_dir": true,
		"policy_id":  1,
		"protocols":  []string{"NFS"},
	}
	viewResponse, err := rest.Views.EnsureByName(viewName, viewCreateParams)
	if err != nil {
		panic(fmt.Errorf("failed to create view: %w", err))
	}
	if err := viewResponse.Fill(&view); err != nil {
		panic(fmt.Errorf("failed to fill ViewContainer: %w", err))
	}
	fmt.Printf("View created: ID=%d, Name=%s, Path=%s, TenantID=%d\n", view.ID, view.Name, view.Path, view.TenantID)

	// Create snap for view
	snapName := "mysnap"
	fmt.Printf("Ensuring snapshot '%s' for path '%s'...\n", snapName, view.Path)
	snapCreateParams := client.Params{"path": view.Path, "tenant_id": view.TenantID}
	snapResponse, err := rest.Snapshots.EnsureByName(snapName, snapCreateParams)
	if err != nil {
		panic(fmt.Errorf("failed to create snapshot: %w", err))
	}
	fmt.Printf("Snapshot created: ID=%d\n", snapResponse.RecordID())

	// Create GSS
	destPath := "/sourceview"
	gssName := "myGss"
	fmt.Printf("Ensuring Global Snapshot Stream '%s'...\n", gssName)
	gssResponse, err := rest.GlobalSnapshotStreams.EnsureGss(gssName, destPath, snapResponse.RecordID(), view.TenantID, true)
	if err != nil {
		panic(fmt.Errorf("failed to ensure GSS: %w", err))
	}
	fmt.Println("GSS created successfully:")
	fmt.Println(gssResponse.PrettyTable())

	// Delete GSS
	fmt.Printf("Ensuring GSS '%s' is deleted...\n", gssName)
	resp, err := rest.GlobalSnapshotStreams.EnsureGssDeleted(client.Params{"name": gssName})
	if err != nil {
		panic(fmt.Errorf("failed to delete GSS: %w", err))
	}
	fmt.Println("GSS deletion result:")
	fmt.Println(resp.PrettyTable())
}
