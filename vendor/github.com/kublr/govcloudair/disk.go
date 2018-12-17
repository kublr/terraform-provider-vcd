/*
 * Copyright 2018 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/kublr/govcloudair/types/v56"
	"net/http"
	"net/url"
)

// Independent disk
type Disk struct {
	Disk   *types.Disk
	client *Client
}

// Init independent disk struct
func NewDisk(cli *Client) *Disk {
	return &Disk{
		Disk:   new(types.Disk),
		client: cli,
	}
}

// Create an independent disk in VDC
// Reference: vCloud API Programming Guide for Service Providers vCloud API 30.0 PDF Page 102 - 103,
// https://vdc-download.vmware.com/vmwb-repository/dcr-public/1b6cf07d-adb3-4dba-8c47-9c1c92b04857/
// 241956dd-e128-4fcc-8131-bf66e1edd895/vcloud_sp_api_guide_30_0.pdf
func (vdc *Vdc) CreateDisk(diskCreateParams *types.DiskCreateParams) (Task, error) {
	u, err := vdc.Vdc.Link.URLForType(types.MimeDiskCreateParams, types.RelAdd)
	if err != nil {
		return Task{}, err
	}

	// Prepare the request payload
	diskCreateParams.Xmlns = types.NsVCloud

	xmlPayload, err := xml.Marshal(diskCreateParams)
	if err != nil {
		return Task{}, fmt.Errorf("error xml.Marshal: %s", err)
	}

	// Send Request
	req := vdc.c.NewRequest(nil, http.MethodPost, *u, bytes.NewBufferString(xml.Header+string(xmlPayload)))
	req.Header.Add("Content-Type", types.MimeDiskCreateParams)
	resp, err := checkResp(vdc.c.Http.Do(req))
	if err != nil {
		return Task{}, fmt.Errorf("error create disk: %s", err)
	}

	// Decode response
	disk := NewDisk(vdc.c)
	if err = decodeBody(resp, disk.Disk); err != nil {
		return Task{}, fmt.Errorf("error decoding create disk params response: %s", err)
	}

	return Task{
		c:    vdc.c,
		Task: disk.Disk.Tasks.Task[0],
	}, nil
}

// Remove an independent disk
// 1 Delete the independent disk. Make a DELETE request to the URL in the rel="remove" link in the Disk.
// 2 Return task of independent disk deletion.
// Please verify the independent disk is not connected to any VM before calling this function.
// If the independent disk is connected to a VM, the task will be failed.
// Reference: vCloud API Programming Guide for Service Providers vCloud API 30.0 PDF Page 106 - 107,
// https://vdc-download.vmware.com/vmwb-repository/dcr-public/1b6cf07d-adb3-4dba-8c47-9c1c92b04857/
// 241956dd-e128-4fcc-8131-bf66e1edd895/vcloud_sp_api_guide_30_0.pdf
func (d *Disk) Delete() (Task, error) {
	u, err := d.Disk.Link.URLForType(types.MimeEmpty, types.RelRemove)
	if err != nil {
		return Task{}, err
	}

	// Make request
	req := d.client.NewRequest(nil, http.MethodDelete, *u, nil)
	resp, err := checkResp(d.client.Http.Do(req))
	if err != nil {
		return Task{}, fmt.Errorf("error delete disk: %s", err)
	}

	// Decode response
	task := NewTask(d.client)
	if err = decodeBody(resp, task.Task); err != nil {
		return Task{}, fmt.Errorf("error decoding delete disk params response: %s", err)
	}

	// Return the task
	return *task, nil
}

// Find an independent disk by VDC client and disk href
func FindDiskByHREF(client *Client, href string) (*Disk, error) {
	// Parse request URI
	reqUrl, err := url.ParseRequestURI(href)
	if err != nil {
		return nil, fmt.Errorf("error parse URI: %s", err)
	}

	// Send request
	req := client.NewRequest(nil, http.MethodGet, *reqUrl, nil)
	resp, err := checkResp(client.Http.Do(req))
	if err != nil {
		return nil, fmt.Errorf("error find disk: %s", err)
	}

	// Decode response
	disk := NewDisk(client)
	if err = decodeBody(resp, disk.Disk); err != nil {
		return nil, fmt.Errorf("error decoding find disk response: %s", err)
	}

	// Return the disk
	return disk, nil
}

// Find an independent disk by VDC client and disk href
func (vdc *Vdc) FindDiskByHREF(href string) (*Disk, error) {
	return FindDiskByHREF(vdc.c, href)
}

// Find an independent disk by name (if there are several disks with the same name,
// the first one will be returned)
func (vdc *Vdc) FindDiskByName(diskName string) (*Disk, error) {
	for _, resourceEntity := range vdc.Vdc.ResourceEntities {
		for _, entity := range resourceEntity.ResourceEntity {
			if entity.Type == types.MimeDisk && entity.Name == diskName {
				return FindDiskByHREF(vdc.c, entity.HREF)
			}
		}
	}

	return nil, fmt.Errorf("disk '%s' was not found", diskName)
}

// Refresh the disk information by disk href
func (d *Disk) Refresh() error {
	disk, err := FindDiskByHREF(d.client, d.Disk.HREF)
	if err != nil {
		return err
	}

	d.Disk = disk.Disk

	return nil
}

// Update an independent disk
// 1 Verify that the disk is not attached to a virtual machine.
// 2 Use newDiskInfo to change update the independent disk.
// 3 Return task of independent disk update
// Please verify the independent disk is not connected to any VM before calling this function.
// If the independent disk is connected to a VM, the task will be failed.
// Reference: vCloud API Programming Guide for Service Providers vCloud API 30.0 PDF Page 104 - 106,
// https://vdc-download.vmware.com/vmwb-repository/dcr-public/1b6cf07d-adb3-4dba-8c47-9c1c92b04857/
// 241956dd-e128-4fcc-8131-bf66e1edd895/vcloud_sp_api_guide_30_0.pdf
func (d *Disk) Update(newDiskInfo *types.Disk) (Task, error) {
	u, err := d.Disk.Link.URLForType(types.MimeDisk, types.RelEdit)
	if err != nil {
		return Task{}, err
	}

	// Prepare the request payload
	xmlPayload, err := xml.Marshal(&types.Disk{
		Xmlns:          types.NsVCloud,
		Description:    newDiskInfo.Description,
		Size:           newDiskInfo.Size,
		Name:           newDiskInfo.Name,
		StorageProfile: newDiskInfo.StorageProfile,
		Owner:          newDiskInfo.Owner,
	})
	if err != nil {
		return Task{}, fmt.Errorf("error xml.Marshal: %s", err)
	}

	// Send request
	req := d.client.NewRequest(nil, http.MethodPut, *u, bytes.NewBufferString(xml.Header+string(xmlPayload)))
	req.Header.Add("Content-Type", types.MimeDisk)
	resp, err := checkResp(d.client.Http.Do(req))
	if err != nil {
		return Task{}, fmt.Errorf("error find disk: %s", err)
	}

	// Decode response
	task := NewTask(d.client)
	if err = decodeBody(resp, task.Task); err != nil {
		return Task{}, fmt.Errorf("error decoding find disk response: %s", err)
	}

	// Return the task
	return *task, nil
}

// Get a VM that is attached the disk
// An independent disk can be attached to at most one virtual machine.
// If the disk isn't attached to any VM, return empty VM reference and no error.
// Otherwise return the first VM reference and no error.
// Reference: vCloud API Programming Guide for Service Providers vCloud API 30.0 PDF Page 107,
// https://vdc-download.vmware.com/vmwb-repository/dcr-public/1b6cf07d-adb3-4dba-8c47-9c1c92b04857/
// 241956dd-e128-4fcc-8131-bf66e1edd895/vcloud_sp_api_guide_30_0.pdf
func (d *Disk) AttachedVM() (*types.Reference, error) {
	u, err := d.Disk.Link.URLForType(types.MimeVMs, types.RelDown)
	if err != nil {
		return nil, fmt.Errorf("exec link not found")
	}

	// Send request
	req := d.client.NewRequest(nil, http.MethodGet, *u, nil)
	req.Header.Add("Content-Type", types.MimeVMs)
	resp, err := checkResp(d.client.Http.Do(req))
	if err != nil {
		return nil, fmt.Errorf("error attached vms: %s", err)
	}

	// Decode request
	var vms = new(types.Vms)
	if err = decodeBody(resp, vms); err != nil {
		return nil, fmt.Errorf("error decoding find disk response: %s", err)
	}

	// If disk is not attached to any VM
	if vms.VmReference == nil {
		return nil, nil
	}

	// An independent disk can be attached to at most one virtual machine so return the first result of VM reference
	return vms.VmReference, nil
}
