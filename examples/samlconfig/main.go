package main

import (
	"fmt"
	client "github.com/vast-data/go-vast-client"
)

func main() {
	config := &client.VMSConfig{
		Host:     "v95", // replace with your VAST address
		Username: "admin",
		Password: "123456",
	}

	rest, err := client.NewVMSRest(config)
	if err != nil {
		panic(err)
	}

	vmsId := 1
	idps, err := rest.Vms.GetConfiguredIdPs(vmsId)
	if err != nil {
		panic(err)
	}

	if len(idps) == 0 {
		fmt.Println("No SAML IdPs configured.")
		return
	}

	idpName := idps[0]

	// Example 1: Get SAML configuration
	fmt.Println("=== Getting SAML Configuration ===")
	samlConfig, err := rest.SamlConfigs.GetConfig(vmsId, idpName)
	if err != nil {
		panic(fmt.Errorf("failed to get SAML config: %w", err))
	}
	fmt.Printf("SAML Configuration: %s\n", samlConfig.PrettyTable())

	// Example 2: Update SAML configuration
	fmt.Println("\n=== Updating SAML Configuration ===")
	updateParams := client.Params{
		"saml_settings": map[string]interface{}{
			"encrypt_assertion":                  true,
			"force_authn":                        true,
			"idp_entityid":                       "https://my-idp.com/entity",
			"idp_metadata_url":                   "https://my-idp.com/metadata",
			"want_assertions_or_response_signed": true,
		},
	}

	result, err := rest.SamlConfigs.UpdateConfig(vmsId, idpName, updateParams)
	if err != nil {
		panic(fmt.Errorf("failed to update SAML config: %w", err))
	}
	fmt.Printf("Update result: %s\n", result)

	// Example 4: Delete SAML configuration
	fmt.Println("\n=== Deleting SAML Configuration ===")
	deleteResult, err := rest.SamlConfigs.DeleteConfig(vmsId, idpName)
	if err != nil {
		panic(fmt.Errorf("failed to delete SAML config: %w", err))
	}
	fmt.Printf("Delete result: %s\n", deleteResult)
}
