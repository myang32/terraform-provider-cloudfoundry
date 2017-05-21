package resources

import (
	"code.cloudfoundry.org/cli/cf/api/resources"
	"code.cloudfoundry.org/cli/cf/formatters"
	"code.cloudfoundry.org/cli/cf/models"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client"
	"log"
	"strings"
)

type CfQuotaResource struct {
	CfResource
}

func (c CfQuotaResource) resourceObject(d *schema.ResourceData) (interface{}, error) {
	totalMemory, err := c.transformToMegabytes(d.Get("total_memory").(string))
	if err != nil {
		return models.QuotaFields{}, err
	}
	instanceMemory, err := c.transformToMegabytes(d.Get("instance_memory").(string))
	if err != nil {
		return models.QuotaFields{}, err
	}
	if c.isOrgQuota(d) {
		return c.resourceOrgQuotaObject(d, totalMemory, instanceMemory), nil
	}
	return c.resourceSpaceQuotaObject(d, totalMemory, instanceMemory), nil
}
func (c CfQuotaResource) transformToMegabytes(value string) (int64, error) {
	if value == "-1" || value == "" || value == "unlimited" {
		return int64(-1), nil
	}
	return formatters.ToMegabytes(value)
}
func (c CfQuotaResource) transformFromMegabytes(value int64) string {
	if value == int64(-1) {
		return "-1"
	}
	return formatters.InstanceMemoryLimit(value)
}
func (c CfQuotaResource) transformFromBytes(value int64) string {
	inMbString := c.transformFromMegabytes(value)
	if inMbString == "-1" {
		return inMbString
	}
	inMb, _ := formatters.ToMegabytes(inMbString)
	return formatters.ByteSize(inMb * formatters.MEGABYTE)
}
func (c CfQuotaResource) objectToResource(d *schema.ResourceData, quotaGeneric interface{}) {
	if c.isOrgQuota(d) {
		quota := quotaGeneric.(models.QuotaFields)
		d.Set("name", quota.Name)
		d.Set("total_memory", c.transformFromBytes(quota.MemoryLimit))
		d.Set("instance_memory", c.transformFromBytes(quota.InstanceMemoryLimit))
		d.Set("routes", quota.RoutesLimit)
		d.Set("service_instances", quota.ServicesLimit)
		d.Set("app_instances", quota.AppInstanceLimit)
		d.Set("allow_paid_service_plans", quota.NonBasicServicesAllowed)
		d.Set("reserved_route_ports", quota.ReservedRoutePorts)
		d.Set("org_id", "")
		return
	}
	quota := quotaGeneric.(models.SpaceQuota)
	d.Set("name", quota.Name)
	d.Set("total_memory", c.transformFromBytes(quota.MemoryLimit))
	d.Set("instance_memory", c.transformFromBytes(quota.InstanceMemoryLimit))
	d.Set("routes", quota.RoutesLimit)
	d.Set("service_instances", quota.ServicesLimit)
	d.Set("app_instances", quota.AppInstanceLimit)
	d.Set("allow_paid_service_plans", quota.NonBasicServicesAllowed)
	d.Set("reserved_route_ports", quota.ReservedRoutePortsLimit)
	d.Set("org_id", quota.OrgGUID)
}
func (c CfQuotaResource) resourceOrgQuotaObject(d *schema.ResourceData, totalMemory, instanceMemory int64) models.QuotaFields {
	return models.QuotaFields{
		GUID:                    d.Id(),
		Name:                    d.Get("name").(string),
		MemoryLimit:             totalMemory,
		InstanceMemoryLimit:     instanceMemory,
		RoutesLimit:             d.Get("routes").(int),
		ServicesLimit:           d.Get("service_instances").(int),
		AppInstanceLimit:        d.Get("app_instances").(int),
		NonBasicServicesAllowed: d.Get("allow_paid_service_plans").(bool),
		ReservedRoutePorts:      json.Number(d.Get("reserved_route_ports").(string)),
	}
}
func (c CfQuotaResource) resourceSpaceQuotaObject(d *schema.ResourceData, totalMemory, instanceMemory int64) models.SpaceQuota {
	return models.SpaceQuota{
		GUID:                    d.Id(),
		Name:                    d.Get("name").(string),
		MemoryLimit:             totalMemory,
		InstanceMemoryLimit:     instanceMemory,
		RoutesLimit:             d.Get("routes").(int),
		ServicesLimit:           d.Get("service_instances").(int),
		AppInstanceLimit:        d.Get("app_instances").(int),
		NonBasicServicesAllowed: d.Get("allow_paid_service_plans").(bool),
		ReservedRoutePortsLimit: json.Number(d.Get("reserved_route_ports").(string)),
		OrgGUID:                 d.Get("org_id").(string),
	}
}
func (c CfQuotaResource) Create(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	quotaName := d.Get("name").(string)
	quota, err := c.resourceObject(d)
	isOrg := c.isOrgQuota(d)
	if err != nil {
		return err
	}
	if ok, _ := c.Exists(d, meta); ok {
		log.Printf(
			"[INFO] skipping creation of quota %s/%s because it already exists on your Cloud Foundry",
			client.Config().ApiEndpoint,
			quotaName,
		)
		return nil
	}
	if isOrg {
		err = client.Quotas().Create(quota.(models.QuotaFields))
	} else {
		err = client.SpaceQuotas().Create(quota.(models.SpaceQuota))
	}
	if err != nil {
		return err
	}
	_, err = c.Exists(d, meta)
	return err
}
func (c CfQuotaResource) isOrgQuota(d *schema.ResourceData) bool {
	return d.Get("org_id").(string) == ""
}
func (c CfQuotaResource) getQuotaFromCf(client cf_client.Client, d *schema.ResourceData) (interface{}, error) {
	var err error
	quotaGuid := d.Id()

	if c.isOrgQuota(d) {
		res := resources.QuotaResource{}
		err = client.Gateways().CloudControllerGateway.GetResource(
			fmt.Sprintf("%s/v2/quota_definitions/%s?inline-relations-depth=1", client.Config().ApiEndpoint, quotaGuid),
			&res)
		if err != nil {
			return models.QuotaFields{}, err
		}
		return res.ToFields(), nil
	}
	res := resources.SpaceQuotaResource{}
	err = client.Gateways().CloudControllerGateway.GetResource(
		fmt.Sprintf("%s/v2/space_quota_definitions/%s?inline-relations-depth=1", client.Config().ApiEndpoint, quotaGuid),
		&res)
	if err != nil {
		return models.QuotaFields{}, err
	}
	return res.ToModel(), nil

}
func (c CfQuotaResource) Read(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	quotaName := d.Get("name").(string)

	quota, err := c.getQuotaFromCf(client, d)
	if err != nil {
		return err
	}
	if (models.QuotaFields{}) == quota {
		log.Printf(
			"[WARN] removing quota %s/%s from state because it no longer exists in your Cloud Foundry",
			client.Config().ApiEndpoint,
			quotaName,
		)
		d.SetId("")
		return nil
	}
	c.objectToResource(d, quota)
	return nil
}
func (c CfQuotaResource) Update(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	quotaName := d.Get("name").(string)
	quota, err := c.resourceObject(d)
	if err != nil {
		return err
	}
	quotaCf, err := c.getQuotaFromCf(client, d)
	if err != nil {
		return err
	}
	if (models.QuotaFields{}) == quotaCf {
		log.Printf(
			"[WARN] removing quota %s/%s from state because it no longer exists in your Cloud Foundry",
			client.Config().ApiEndpoint,
			quotaName,
		)
		d.SetId("")
		return nil
	}
	if c.isOrgQuota(d) {
		return client.Quotas().Update(quota.(models.QuotaFields))
	}
	return client.SpaceQuotas().Update(quota.(models.SpaceQuota))
}
func (c CfQuotaResource) Delete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	if c.isOrgQuota(d) {
		return client.Quotas().Delete(d.Id())
	}
	return client.SpaceQuotas().Delete(d.Id())
}
func (c CfQuotaResource) Exists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(cf_client.Client)
	name := d.Get("name").(string)
	var quota interface{}
	var err error
	isOrg := c.isOrgQuota(d)
	if isOrg {
		quota, err = client.Quotas().FindByName(name)
	} else {
		quota, err = client.SpaceQuotas().FindByNameAndOrgGUID(name, d.Get("org_id").(string))
	}
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}
	if isOrg {
		d.SetId(quota.(models.QuotaFields).GUID)
	} else {
		d.SetId(quota.(models.SpaceQuota).GUID)
	}
	return true, nil
}
func (c CfQuotaResource) Schema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"total_memory": &schema.Schema{
			Type:     schema.TypeString,
			Default:  "20G",
			Optional: true,
		},
		"instance_memory": &schema.Schema{
			Type:     schema.TypeString,
			Default:  "-1",
			Optional: true,
		},
		"routes": &schema.Schema{
			Type:     schema.TypeInt,
			Default:  2000,
			Optional: true,
		},
		"service_instances": &schema.Schema{
			Type:     schema.TypeInt,
			Default:  200,
			Optional: true,
		},
		"app_instances": &schema.Schema{
			Type:     schema.TypeInt,
			Default:  -1,
			Optional: true,
		},
		"allow_paid_service_plans": &schema.Schema{
			Type:     schema.TypeBool,
			Default:  true,
			Optional: true,
		},
		"reserved_route_ports": &schema.Schema{
			Type:     schema.TypeString,
			Default:  0,
			Optional: true,
		},
		"org_id": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
	}
}
