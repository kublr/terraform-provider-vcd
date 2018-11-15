package vcd

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/pkg/errors"
	"log"
)

func resourceVcdCatalog() *schema.Resource {
	return &schema.Resource{
		Create: resourceVcdCatalogCreate,
		Update: resourceVcdCatalogUpdate,
		Read:   resourceVcdCatalogRead,
		Delete: resourceVcdCatalogDelete,

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
		},
	}
}

func resourceVcdCatalogCreate(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)
	// See if catalog exists
	catalog, err := vcdClient.Org.FindCatalog(d.Get("name").(string))
	log.Printf("[TRACE] Looking for existing catalog, found %#v", catalog)
	if err != nil {
		log.Printf("[TRACE] No catalog found, preparing creation")
		adminOrg, err := vcdClient.GetAdminOrg()
		if err != nil {
			return errors.Wrap(err, "Unable to create Catalog because error during getting AdminOrg")
		}
		err = retryCall(vcdClient.MaxRetryTimeout, func() *resource.RetryError {
			task, err := adminOrg.CreateCatalog(d.Get("name").(string), d.Get("description").(string))
			if err != nil {
				return resource.NonRetryableError(fmt.Errorf("error creating Catalog: %#v", err))
			}
			return resource.RetryableError(task.WaitTaskCompletion())
		})

		if err != nil {
			return fmt.Errorf("Error completing tasks: %#v", err)
		}

		err = vcdClient.Org.Refresh()
		if err != nil {
			return fmt.Errorf("error refreshing org: %#v", err)
		}
		catalog, err = vcdClient.Org.FindCatalog(d.Get("name").(string))
		if err != nil {
			return fmt.Errorf("error refreshing just created catalog: %#v", err)
		}
	}

	d.SetId(d.Get("name").(string))

	return nil
}

func resourceVcdCatalogUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Println("[TRACE] resourceVcdCatalogUpdate")
	vcdClient := meta.(*VCDClient)
	log.Printf("[TRACE] Updating state from Org")
	err := vcdClient.Org.Refresh()
	if err != nil {
		return fmt.Errorf("error refreshing Org: %#v", err)
	}
	adminOrg, err := vcdClient.GetAdminOrg()
	if err != nil {
		return errors.Wrap(err, "Unable to update Catalog because error during getting AdminOrg")
	}

	// Should be fetched by ID/HREF
	adminCatalog, err := adminOrg.FindAdminCatalog(d.Id())
	if err != nil {
		log.Printf("[DEBUG] Unable to find catalog. Removing from tfstate")
		d.SetId("")
		return nil
	}
	adminCatalog.AdminCatalog.Description = d.Get("description").(string)
	err = adminCatalog.Update()
	if err != nil {
		log.Printf("[DEBUG] Unable to updage catalog")
		return err
	}
	return nil
}

func resourceVcdCatalogRead(d *schema.ResourceData, meta interface{}) error {
	log.Println("[TRACE] resourceVcdCatalogRead")
	vcdClient := meta.(*VCDClient)
	log.Printf("[TRACE] Updating state from Org")
	err := vcdClient.Org.Refresh()
	if err != nil {
		return fmt.Errorf("error refreshing org: %#v", err)
	}

	// Should be fetched by ID/HREF
	catalog, err := vcdClient.Org.FindCatalog(d.Id())
	if err != nil {
		log.Printf("[DEBUG] Unable to find catalog. Removing from tfstate")
		d.SetId("")
		return nil
	}
	d.Set("description", catalog.Catalog.Description)
	return nil
}

func resourceVcdCatalogDelete(d *schema.ResourceData, meta interface{}) error {
	log.Println("[TRACE] resourceVcdCatalogDelete")
	vcdClient := meta.(*VCDClient)
	log.Printf("[TRACE] Updating state from VCD")
	err := vcdClient.Org.Refresh()
	if err != nil {
		return fmt.Errorf("error refreshing org: %#v", err)
	}
	adminOrg, err := vcdClient.GetAdminOrg()
	if err != nil {
		return errors.Wrap(err, "Unable to delete Catalog because error during getting AdminOrg")
	}

	// Should be fetched by ID/HREF
	adminCatalog, err := adminOrg.FindAdminCatalog(d.Id())
	if err != nil {
		log.Printf("[DEBUG] Unable to find catalog")
		return err
	}

	err = adminCatalog.Delete(true, true)
	if err != nil {
		log.Printf("[DEBUG] Unable to delete catalog: %s", err.Error())
		return err
	}
	// Wait until catalog really deleted
	err = retryCall(vcdClient.MaxRetryTimeout, func() *resource.RetryError {
		adminOrg.Refresh()
		_, err := adminOrg.FindAdminCatalog(d.Id())
		if err != nil {
			return nil
		}
		log.Printf("[DEBUG] Waiting until catalog %s deleted", d.Id())
		return resource.RetryableError(errors.Errorf("Catalog %s is not deleted yet", d.Id()))
	})

	if err != nil {
		return fmt.Errorf("Error completing tasks: %#v", err)
	}

	return nil
}
