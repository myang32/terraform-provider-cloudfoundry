package resources

import (
	"code.cloudfoundry.org/cli/cf/models"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client"
)

type CfStackResource struct{}

func (c CfStackResource) DataSourceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
		"guid": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
		"first": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
		},
	}
}
func (c CfStackResource) DataSourceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(cf_client.Client)
	if d.Get("first").(bool) {
		stacks, err := client.Stack().FindAll()
		if err != nil {
			return err
		}
		c.flattenStack(d, stacks[0])
		return nil
	}
	stackId := d.Get("guid").(string)
	name := d.Get("name").(string)
	if name == "" && stackId == "" {
		return fmt.Errorf("You must set param 'name' or 'stack_id' if the param 'first' is to false.")
	}
	var stack models.Stack
	var err error
	if stackId != "" {
		stack, err = client.Stack().FindByGUID(stackId)
	} else {
		stack, err = client.Stack().FindByName(name)
	}
	if err != nil {
		return err
	}
	c.flattenStack(d, stack)
	return nil
}
func (c CfStackResource) flattenStack(d *schema.ResourceData, s models.Stack) {
	d.SetId(s.GUID)
	d.Set("name", s.Name)
}
