package artifactory

import (
	"context"
	"fmt"
	"net/http"

	"github.com/atlassian/go-artifactory/v2/artifactory"
	"github.com/atlassian/go-artifactory/v2/artifactory/v1"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArtifactoryGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceGroupCreate,
		Read:   resourceGroupRead,
		Update: resourceGroupUpdate,
		Delete: resourceGroupDelete,
		Exists: resourceGroupExists,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"auto_join": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"admin_privileges": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"realm": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateLowerCase,
			},
			"realm_attributes": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func unmarshalGroup(s *schema.ResourceData) (*v1.Group, error) {
	d := &ResourceData{s}

	group := new(v1.Group)

	group.Name = d.getStringRef("name", false)
	group.Description = d.getStringRef("description", false)
	group.AutoJoin = d.getBoolRef("auto_join", false)
	group.AdminPrivileges = d.getBoolRef("admin_privileges", false)
	group.Realm = d.getStringRef("realm", false)
	group.RealmAttributes = d.getStringRef("realm_attributes", false)

	// Validator
	if group.AdminPrivileges != nil && group.AutoJoin != nil && *group.AdminPrivileges && *group.AutoJoin {
		return nil, fmt.Errorf("error: auto_join cannot be true if admin_privileges is true")
	}

	return group, nil
}

func resourceGroupCreate(d *schema.ResourceData, m interface{}) error {
	c := m.(*artifactory.Artifactory)

	group, err := unmarshalGroup(d)

	if err != nil {
		return err
	}

	_, err = c.V1.Security.CreateOrReplaceGroup(context.Background(), *group.Name, group)

	if err != nil {
		return err
	}

	d.SetId(*group.Name)
	return resourceGroupRead(d, m)
}

func resourceGroupRead(d *schema.ResourceData, m interface{}) error {
	c := m.(*artifactory.Artifactory)

	group, resp, err := c.V1.Security.GetGroup(context.Background(), d.Id())

	// If we 404 it is likely the resources was externally deleted
	// If the ID is updated to blank, this tells Terraform the resource no longer exist
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	} else if err != nil {
		return err
	}

	hasErr := false
	logError := cascadingErr(&hasErr)
	logError(d.Set("name", group.Name))
	logError(d.Set("description", group.Description))
	logError(d.Set("auto_join", group.AutoJoin))
	logError(d.Set("admin_privileges", group.AdminPrivileges))
	logError(d.Set("realm", group.Realm))
	logError(d.Set("realm_attributes", group.RealmAttributes))
	if hasErr {
		return fmt.Errorf("failed to marshal group")
	}
	return nil
}

func resourceGroupUpdate(d *schema.ResourceData, m interface{}) error {
	c := m.(*artifactory.Artifactory)
	group, err := unmarshalGroup(d)
	if err != nil {
		return err
	}
	_, err = c.V1.Security.UpdateGroup(context.Background(), d.Id(), group)
	if err != nil {
		return err
	}

	d.SetId(*group.Name)
	return resourceGroupRead(d, m)
}

func resourceGroupDelete(d *schema.ResourceData, m interface{}) error {
	c := m.(*artifactory.Artifactory)
	group, err := unmarshalGroup(d)
	if err != nil {
		return err
	}

	_, resp, err := c.V1.Security.DeleteGroup(context.Background(), *group.Name)

	if err != nil && resp.StatusCode == http.StatusNotFound {
		return nil
	}

	return err
}

func resourceGroupExists(d *schema.ResourceData, m interface{}) (bool, error) {
	c := m.(*artifactory.Artifactory)

	groupName := d.Id()
	_, resp, err := c.V1.Security.GetGroup(context.Background(), groupName)

	return err == nil && resp.StatusCode != http.StatusNotFound, err
}
