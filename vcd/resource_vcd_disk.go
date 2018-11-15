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
			"description": {
				Type:     schema.TypeString,
				Optional: true,
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
			"bus_type": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"bus_sub_type": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"storage_profile": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
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
		Description: d.Get("description").(string),
		Size:        int(diskSize),
		Iops:        &diskIops,
		BusType:     d.Get("bus_type").(string),
		BusSubType:  d.Get("bus_sub_type").(string),
	}

	storageProfileName := d.Get("storage_profile").(string)
	if storageProfileName != "" {
		storageProfile, err := vcdClient.OrgVdc.FindStorageProfileReference(storageProfileName)
		if err != nil {
			return err
		}

		diskCreateParamsDisk.StorageProfile = &storageProfile
	} else {
		diskCreateParamsDisk.StorageProfile = nil
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

	if err != nil {
		return fmt.Errorf("Error completing tasks: %#v", err)
	}

	d.SetId(d.Get("name").(string))

	return resourceVcdDiskRead(d, meta)
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
		Description: d.Get("description").(string),
		Size:        int(diskSize),
		Iops:        &diskIops,
		BusType:     d.Get("bus_type").(string),
		BusSubType:  d.Get("bus_sub_type").(string),
	}

	storageProfileName := d.Get("storage_profile").(string)
	if storageProfileName != "" {
		storageProfile, err := vcdClient.OrgVdc.FindStorageProfileReference(storageProfileName)
		if err != nil {
			return err
		}

		diskNewParams.StorageProfile = &storageProfile
	} else {
		diskNewParams.StorageProfile = nil
	}

	log.Printf("[INFO] Update disk '%s'", diskName)

	err = retryCall(vcdClient.MaxRetryTimeout, func() *resource.RetryError {
		task, err := disk.Update(diskNewParams)
		if err != nil {
			return resource.NonRetryableError(fmt.Errorf("Error updating disk '%s': %#v", diskName, err))
		}

		return resource.RetryableError(task.WaitTaskCompletion())
	})

	if err != nil {
		return fmt.Errorf("Error completing tasks: %#v", err)
	}

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
	d.Set("description", disk.Disk.Description)
	d.Set("size", disk.Disk.Size)
	d.Set("iops", disk.Disk.Iops)
	d.Set("bus_type", disk.Disk.BusType)
	d.Set("bus_sub_type", disk.Disk.BusSubType)
	if disk.Disk.StorageProfile != nil {
		d.Set("storage_profile", disk.Disk.StorageProfile.Name)
	}

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

	if err != nil {
		return fmt.Errorf("Error completing tasks: %#v", err)
	}

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
