package main

import (
	"fmt"
	client "github.com/vast-data/go-vast-client"
)

type FolderStatsContainer struct {
	OwningUser      string `json:"owning_user"`
	OwningUID       int    `json:"owning_uid"`
	OwningGroup     string `json:"owning_group"`
	OwningGID       int    `json:"owning_gid"`
	IsDirectory     bool   `json:"is_directory"`
	Children        int    `json:"children"`
	SmbReadonlyFlag bool   `json:"smb_readonly_flag"`
	ATime           string `json:"atime"`
	MTime           string `json:"mtime"`
	CTime           string `json:"ctime"`
}

func main() {
	config := &client.VMSConfig{
		Host:     "v95",   // replace with your VAST IP
		Username: "admin", // Authentication user
		Password: "123456",
	}

	rest, err := client.NewVMSRest(config)
	if err != nil {
		panic(err)
	}

	folderPath := "/example/test/folder/1"
	tenantID := 1

	fmt.Println("=== FOLDER OPERATIONS EXAMPLE ===")

	// --- CREATE FOLDER ---
	fmt.Println("\n1. Creating folder...")
	createParams := client.Params{
		"path":            folderPath,
		"tenant_id":       tenantID,
		"create_dir_mode": 755,
		"inherit_acl":     true,
	}

	createResult, err := rest.Folders.CreateFolder(createParams)
	if err != nil {
		panic(fmt.Errorf("failed to create folder: %w", err))
	}
	fmt.Printf("✓ Folder created successfully\n")
	fmt.Printf("  Result: %s\n", createResult.PrettyTable())

	// --- GET FOLDER STATS ---
	fmt.Println("\n2. Getting folder statistics...")
	statsParams := client.Params{
		"path":      folderPath,
		"tenant_id": tenantID,
	}
	statsResult, err := rest.Folders.StatPath(statsParams)
	if err != nil {
		panic(fmt.Errorf("failed to get folder stats: %w", err))
	}

	var stats FolderStatsContainer
	if err := statsResult.Fill(&stats); err != nil {
		panic(fmt.Errorf("failed to parse folder stats: %w", err))
	}
	fmt.Printf("✓ Folder stats retrieved successfully\n")
	fmt.Printf("  Owner: %s (UID: %d)\n", stats.OwningUser, stats.OwningUID)
	fmt.Printf("  Group: %s (GID: %d)\n", stats.OwningGroup, stats.OwningGID)
	fmt.Printf("  Is Directory: %t\n", stats.IsDirectory)
	fmt.Printf("  Children: %d\n", stats.Children)
	fmt.Printf("  SMB Read-only: %t\n", stats.SmbReadonlyFlag)

	// --- SET READ-ONLY ---
	fmt.Println("\n5. Setting folder as read-only...")
	readOnlyParams := client.Params{
		"path":      folderPath,
		"tenant_id": tenantID,
	}
	setReadOnlyResult, err := rest.Folders.SetReadOnly(readOnlyParams)
	if err != nil {
		panic(fmt.Errorf("failed to set folder as read-only: %w", err))
	}
	fmt.Printf("✓ Folder set as read-only successfully\n")
	fmt.Printf("  Result: %s\n", setReadOnlyResult.PrettyTable())

	// --- GET READ-ONLY STATUS ---
	fmt.Println("\n6. Getting read-only folder information...")
	getReadOnlyParams := client.Params{
		"path":      folderPath,
		"tenant_id": tenantID,
	}
	readOnlyInfo, err := rest.Folders.GetReadOnly(getReadOnlyParams)
	if err != nil {
		panic(fmt.Errorf("failed to get read-only folder info: %w", err))
	}
	fmt.Printf("✓ Read-only information retrieved successfully\n")
	fmt.Printf("  Result: %s\n", readOnlyInfo.PrettyTable())

	// --- DELETE READ-ONLY STATUS ---
	fmt.Println("\n7. Removing read-only status...")
	deleteReadOnlyParams := client.Params{
		"path":      folderPath,
		"tenant_id": tenantID,
	}
	_, err = rest.Folders.DeleteReadOnly(deleteReadOnlyParams)
	if err != nil {
		panic(fmt.Errorf("failed to remove read-only status: %w", err))
	}
	fmt.Printf("✓ Read-only status removed successfully\n")

	// --- VERIFY READ-ONLY REMOVED ---
	fmt.Println("\n8. Verifying read-only status is removed...")
	finalStatsResult, err := rest.Folders.StatPath(statsParams)
	if err != nil {
		panic(fmt.Errorf("failed to get final folder stats: %w", err))
	}

	var finalStats FolderStatsContainer
	if err := finalStatsResult.Fill(&finalStats); err != nil {
		panic(fmt.Errorf("failed to parse final folder stats: %w", err))
	}
	fmt.Printf("✓ Final verification completed\n")
	fmt.Printf("  SMB Read-only: %t\n", finalStats.SmbReadonlyFlag)

	// --- DELETE FOLDER ---
	fmt.Println("\n9. Deleting folder...")
	deleteParams := client.Params{
		"path":      folderPath,
		"tenant_id": tenantID,
	}
	_, err = rest.Folders.DeleteFolder(deleteParams)
	if err != nil {
		panic(fmt.Errorf("failed to delete folder: %w", err))
	}
	fmt.Printf("✓ Folder deleted successfully\n")

	fmt.Println("\n=== ALL FOLDER OPERATIONS COMPLETED SUCCESSFULLY ===")
}
