package vcd

import (
	"fmt"
	"log"

	"github.com/alecthomas/units"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/kublr/govcloudair/types/v56"
)

func resourceVcdDisk() *schema.Resource {
	return &schema.Resource{
		Create: resourceVcdDiskCreate,
		Update: resourceVcdDiskUpdate,
		Read:   resourceVcdDiskRead,
		Delete: resourceVcdDiskDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"size": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: ValidateDiskSize(),
			},
			"iops": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceVcdDiskCreate(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)

	diskName := d.Get("name").(string)

	// checking if the disk exists
	foundDisk, err := vcdClient.OrgVdc.FindDiskByName(diskName)
	if err == nil {
		return fmt.Errorf("The disk '%s' already exists (HREF: '%s')", diskName, foundDisk.Disk.HREF)
	}

	diskSize, err := units.ParseBase2Bytes(d.Get("size").(string))
	if err != nil {
		return fmt.Errorf("wrong disk size '%s'", d.Get("size").(string))
	}

	diskIops := d.Get("iops").(int)

	diskCreateParamsDisk := &types.Disk{
		Name:        diskName,
		Size:        int(diskSize),
		Iops:        &diskIops,
		Description: d.Get("description").(string),
	}

	diskCreateParams := &types.DiskCreateParams{
		Disk: diskCreateParamsDisk,
	}

	log.Printf("[INFO] Create disk '%s'", diskName)

	err = retryCall(vcdClient.MaxRetryTimeout, func() *resource.RetryError {
		task, err := vcdClient.OrgVdc.CreateDisk(diskCreateParams)
		if err != nil {
			return resource.NonRetryableError(fmt.Errorf("Error creating disk '%s': %#v", diskName, err))
		}

		return resource.RetryableError(task.WaitTaskCompletion())
	})

	d.SetId(d.Get("name").(string))

	return nil
}

func resourceVcdDiskUpdate(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)

	// checking if the disk exists
	disk, err := vcdClient.OrgVdc.FindDiskByName(d.Id())
	if err != nil {
		log.Printf("Disk '%s' does not exists. removing from tfstate", d.Id())
		return fmt.Errorf("Disk '%s' does not exists. removing from tfstate", d.Id())
	}

	diskName := d.Get("name").(string)
	diskSize, err := units.ParseBase2Bytes(d.Get("size").(string))
	if err != nil {
		return fmt.Errorf("wrong disk size '%s'", d.Get("size").(string))
	}

	diskIops := d.Get("iops").(int)

	diskNewParams := &types.Disk{
		Name:        diskName,
		Size:        int(diskSize),
		Iops:        &diskIops,
		Description: d.Get("description").(string),
	}

	log.Printf("[INFO] Update disk '%s'", diskName)

	err = retryCall(vcdClient.MaxRetryTimeout, func() *resource.RetryError {
		task, err := disk.Update(diskNewParams)
		if err != nil {
			return resource.NonRetryableError(fmt.Errorf("Error updating disk '%s': %#v", diskName, err))
		}

		return resource.RetryableError(task.WaitTaskCompletion())
	})

	return resourceVcdDiskRead(d, meta)
}

func resourceVcdDiskRead(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)

	err := vcdClient.OrgVdc.Refresh()
	if err != nil {
		return fmt.Errorf("Error refreshing vdc: %#v", err)
	}

	disk, err := vcdClient.OrgVdc.FindDiskByName(d.Id())
	if err != nil {
		log.Printf("Disk '%s' does not exists. removing from tfstate", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("name", disk.Disk.Name)
	d.Set("size", disk.Disk.Size)
	d.Set("iops", disk.Disk.Iops)
	d.Set("description", disk.Disk.Description)

	return nil
}

func resourceVcdDiskDelete(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)

	disk, err := vcdClient.OrgVdc.FindDiskByName(d.Id())
	if err != nil {
		return fmt.Errorf("The disk '%s' does not exist", d.Id())
	}

	err = retryCall(vcdClient.MaxRetryTimeout, func() *resource.RetryError {
		task, err := disk.Delete()
		if err != nil {
			return resource.NonRetryableError(fmt.Errorf("Error deleting disk '%s': %#v", d.Id(), err))
		}

		return resource.RetryableError(task.WaitTaskCompletion())
	})

	return nil
}

func ValidateDiskSize() schema.SchemaValidateFunc {
	return func(i interface{}, k string) (s []string, es []error) {
		v, ok := i.(string)
		if !ok {
			es = append(es, fmt.Errorf("expected type of %s to be string", k))
			return
		}

		_, err := units.ParseBase2Bytes(v)
		if err != nil {
			es = append(es, fmt.Errorf("expected value: %s to be a valid disk size (i.e 1KB, 2MB, 3GB and etc)", v))
		}

		return
	}
}
