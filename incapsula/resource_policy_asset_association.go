package incapsula

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"log"
	"strings"
)

func resourcePolicyAssetAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourcePolicyAssetAssociationCreate,
		Read:   resourcePolicyAssetAssociationRead,
		Update: nil,
		Delete: resourcePolicyAssetAssociationDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		CustomizeDiff: validateUniqueResource,
		Schema: map[string]*schema.Schema{
			// Required Arguments
			"policy_id": {
				Description: "The Policy ID for the asset association.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"asset_id": {
				Description: "The Asset ID for the asset association. Only type of asset supported at the moment is site.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"asset_type": {
				Description: "The Policy type for the asset association. Only value at the moment is `WEBSITE`.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			// Optional Arguments
			"account_id": {
				Description: "The Asset's Account ID",
				Type:        schema.TypeInt,
				Computed:    true,
				Optional:    true,
				ForceNew:    true,
			},
		},
	}
}

func resourcePolicyAssetAssociationCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*Client)

	policyID := d.Get("policy_id").(string)
	assetID := d.Get("asset_id").(string)
	assetType := d.Get("asset_type").(string)
	currentAccountId := d.Get("account_id").(int)

	err := client.AddPolicyAssetAssociation(policyID, assetID, assetType, &currentAccountId)

	if err != nil {
		log.Printf("[ERROR] Could not create Incapsula policy asset association: policy ID (%s) - asset ID (%s) - asset type (%s) - %s\n", policyID, assetID, assetType, err)
		return err
	}

	// Generate synthetic ID
	syntheticID := fmt.Sprintf("%s/%s/%s", policyID, assetID, assetType)
	d.SetId(syntheticID)
	log.Printf("[INFO] Created Incapsula policy asset association with ID: %s - policy ID (%s) - asset ID (%s) - asset type (%s)\n", syntheticID, policyID, assetID, assetType)

	return resourcePolicyAssetAssociationRead(d, m)
}

func resourcePolicyAssetAssociationRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*Client)

	policyID := strings.Split(d.Id(), "/")[0]
	assetID := strings.Split(d.Id(), "/")[1]
	assetType := strings.Split(d.Id(), "/")[2]
	currentAccountId := getCurrentAccountId(d, client.accountStatus)
	if currentAccountId != nil {
		log.Printf("[INFO] Trying to read Incapsula Policy Asset Association: %s-%s-%s for account %d\n", policyID, assetID, assetType, *currentAccountId)
	} else {
		log.Printf("[INFO] Trying to read Incapsula Policy Asset Association: %s-%s-%s\n", policyID, assetID, assetType)
	}
	var isAssociated, err = client.isPolicyAssetAssociated(policyID, assetID, assetType, currentAccountId)

	if err != nil {
		log.Printf("[ERROR] Could not read Incapsula Policy Asset Association: %s-%s-%s, err: %s\n", policyID, assetID, assetType, err)
		return err
	}

	if !isAssociated {
		log.Printf("[ERROR] Could not find Incapsula Policy Asset Association: %s-%s-%s\n", policyID, assetID, assetType)
		d.SetId("")
		return nil
	}

	log.Printf("[INFO] Successfully read Policy Asset Association exist: %s-%s-%s\n", policyID, assetID, assetType)
	syntheticID := fmt.Sprintf("%s/%s/%s", policyID, assetID, assetType)

	d.Set("asset_id", assetID)
	d.Set("asset_type", assetType)
	d.Set("policy_id", policyID)
	if currentAccountId != nil {
		d.Set("account_id", *currentAccountId)
	}
	d.SetId(syntheticID)

	return nil
}

func resourcePolicyAssetAssociationDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*Client)

	policyID := d.Get("policy_id").(string)
	assetID := d.Get("asset_id").(string)
	assetType := d.Get("asset_type").(string)
	currentAccountId := getCurrentAccountId(d, client.accountStatus)
	if currentAccountId != nil {
		log.Printf("[INFO] Trying to delete Incapsula Policy Asset Association: %s-%s-%s for account %d\n", policyID, assetID, assetType, *currentAccountId)
	} else {
		log.Printf("[INFO] Trying to delete Incapsula Policy Asset Association: %s-%s-%s\n", policyID, assetID, assetType)
	}
	err := client.DeletePolicyAssetAssociation(policyID, assetID, assetType, currentAccountId)

	if err != nil {
		return err
	}

	// Set the ID to empty
	// Implicitly clears the resource
	d.SetId("")

	return nil
}

func validateUniqueResource(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
	rawPlan := d.GetRawState()
	if rawPlan.IsNull() {
		return nil
	}
	client := m.(*Client)
	policyGetResponse, _ := client.GetPolicy(d.Get("policy_id").(string), nil)
	log.Printf("[DEBUG] TEST RESURCE 1 POLICY ID: %d - POLICY TYPE: %s\n", policyGetResponse.Value.ID, policyGetResponse.Value.PolicyType)
	if policyGetResponse.Value.PolicyType == "WAF_RULES" {
		var mySet map[string]struct{}
		mySet = make(map[string]struct{})
		log.Printf("[DEBUG] TEST RESURCE 2 POLICY GetRawState: %s - \n", rawPlan.AsValueMap())
		log.Printf("[DEBUG] TEST RESURCE 3 POLICY GetRawPlan: %s - \n", d.GetRawPlan().AsValueMap())
		log.Printf("[DEBUG] TEST RESURCE 4 POLICY GetRawConfig: %s - \n", d.GetRawConfig().AsValueMap())
		for _, resource := range rawPlan.AsValueMap() {
			if fmt.Sprintf("%v", resource.Type) == "incapsula_policy_asset_association" {
				policyID := fmt.Sprintf("%v", resource.GetAttr("policy_id"))

				policyGetResponse, err := client.GetPolicy(policyID, nil)

				if err != nil {
					log.Printf("[ERROR] Could not get Incapsula policy: %s - %s\n", policyID, err)
					return err
				}

				log.Printf("[DEBUG] TEST RESURCE 3 POLICY POLICY TYPE: %s - \n", policyGetResponse.Value.PolicyType)

				if policyGetResponse.Value.PolicyType == "WAF_RULES" {
					assetId := fmt.Sprintf("%v", resource.GetAttr("asset_id"))
					log.Printf("[DEBUG] TEST RESURCE 5 POLICY POLICY ASSET: %s ASSETS LIST: %s - \n", assetId, mySet)
					if _, exists := mySet[assetId]; exists {
						return fmt.Errorf("site %s has more than one WAF Policy assigned", assetId)
					} else {
						mySet[assetId] = struct{}{}
					}
				}
			}

		}
	}
	return nil
}

func validateUniqueResource2(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
	rawState := d.GetProviderMeta().(*terraform.InstanceState)
	if rawState == nil {
		return nil
	}

	client := m.(*Client)
	policyGetResponse, _ := client.GetPolicy(d.Get("policy_id").(string), nil)
	log.Printf("[DEBUG] TEST RESURCE 1 POLICY ID: %d - POLICY TYPE: %s\n", policyGetResponse.Value.ID, policyGetResponse.Value.PolicyType)
	if policyGetResponse.Value.PolicyType == "WAF_RULES" {
		var mySet map[string]struct{}
		mySet = make(map[string]struct{})
		log.Printf("[DEBUG] TEST RESURCE 2 POLICY GetRawState: %s - \n", rawState.Attributes)
		for _, resource := range rawState.Attributes {
			if resource.Type == "incapsula_policy_asset_association" {
				policyID := resource.Attributes["policy_id"].(string)

				policyGetResponse, err := client.GetPolicy(policyID, nil)
				if err != nil {
					log.Printf("[ERROR] Could not get Incapsula policy: %s - %s\n", policyID, err)
					return err
				}

				log.Printf("[DEBUG] TEST RESURCE 3 POLICY POLICY TYPE: %s - \n", policyGetResponse.Value.PolicyType)
				if policyGetResponse.Value.PolicyType == "WAF_RULES" {
					assetId := resource.Attributes["asset_id"].(string)
					log.Printf("[DEBUG] TEST RESURCE 5 POLICY POLICY ASSET: %s ASSETS LIST: %s - \n", assetId, mySet)
					if _, exists := mySet[assetId]; exists {
						return fmt.Errorf("site %s has more than one WAF Policy assigned", assetId)
					} else {
						mySet[assetId] = struct{}{}
					}
				}
			}
		}
	}
	return nil
}

func validateUniqueResource3(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
	rawState := d.State().RootModule().Resources
	if rawState == nil {
		return nil
	}

	client := m.(*Client)
	var mySet map[string]struct{}
	mySet = make(map[string]struct{})

	for _, resource := range rawState {
		if resource.Type == "incapsula_policy_asset_association" {
			policyID := resource.Primary.Attributes["policy_id"]

			policyGetResponse, err := client.GetPolicy(policyID, nil)
			if err != nil {
				log.Printf("[ERROR] Could not get Incapsula policy: %s - %s\n", policyID, err)
				return err
			}

			if policyGetResponse.Value.PolicyType == "WAF_RULES" {
				assetId := resource.Primary.Attributes["asset_id"]
				if _, exists := mySet[assetId]; exists {
					return fmt.Errorf("site %s has more than one WAF Policy assigned", assetId)
				} else {
					mySet[assetId] = struct{}{}
				}
			}
		}
	}
	return nil
}
