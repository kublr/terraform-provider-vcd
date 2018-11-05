/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/kublr/govcloudair/types/v56"
)

type VM struct {
	VM *types.VM
	c  *Client
}

func NewVM(c *Client) *VM {
	return &VM{
		VM: new(types.VM),
		c:  c,
	}
}

func (v *VM) GetStatus() (string, error) {
	err := v.Refresh()
	if err != nil {
		return "", fmt.Errorf("error refreshing VM: %v", err)
	}
	return types.VAppStatuses[v.VM.Status], nil
}

func (v *VM) Refresh() error {

	if v.VM.HREF == "" {
		return fmt.Errorf("cannot refresh VM, Object is empty")
	}

	u, _ := url.ParseRequestURI(v.VM.HREF)

	req := v.c.NewRequest(map[string]string{}, "GET", *u, nil)

	resp, err := checkResp(v.c.Http.Do(req))
	if err != nil {
		return fmt.Errorf("error retrieving task: %s", err)
	}
	defer resp.Body.Close()

	// Empty struct before a new unmarshal, otherwise we end up with duplicate
	// elements in slices.
	v.VM = &types.VM{}

	if err = decodeBody(resp, v.VM); err != nil {
		return fmt.Errorf("error decoding task response VM: %s", err)
	}

	// The request was successful
	return nil
}

func (v *VM) Reconfigure() (Task, error) {
	// err := v.Refresh()
	// if err != nil {
	// 	return Task{}, fmt.Errorf("error refreshing VM before running customization: %v", err)
	// }

	// WORKAROUND for XML namespace support in go, see bottom of types.go
	// github.com/kublr/govcloudair/types/v56
	// v.CorrectAddressOnParentForNetworkHardware()

	v.SetXMLNamespaces()

	ovfVirtualHardwareSection := v.VM.VirtualHardwareSection.ConvertToOVF()
	virtualHardwareSection := v.VM.VirtualHardwareSection
	v.VM.VirtualHardwareSection = nil
	v.VM.OVFVirtualHardwareSection = ovfVirtualHardwareSection

	output, err := xml.MarshalIndent(v.VM, "  ", "    ")
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}

	v.VM.VirtualHardwareSection = virtualHardwareSection

	v.VM.OVFVirtualHardwareSection = nil

	log.Printf("[DEBUG] Reconfigured VM: \n%s", output)

	return ExecuteRequest(string(output),
		v.VM.HREF+"/action/reconfigureVm",
		"POST",
		"application/vnd.vmware.vcloud.vm+xml",
		v.c)
}

func (v *VM) PowerOn() (Task, error) {
	return ExecuteRequest("",
		v.VM.HREF+"/power/action/powerOn",
		"POST",
		"application/vnd.vmware.vcloud.vm+xml",
		v.c)
}

func (v *VM) PowerOff() (Task, error) {
	return ExecuteRequest("",
		v.VM.HREF+"/power/action/powerOff",
		"POST",
		"application/vnd.vmware.vcloud.vm+xml",
		v.c)
}

func (v *VM) Shutdown() (Task, error) {
	return ExecuteRequest("",
		v.VM.HREF+"/power/action/shutdown",
		"POST",
		"application/vnd.vmware.vcloud.vm+xml",
		v.c)
}

func (v *VM) Undeploy(action types.UndeployPowerAction) (Task, error) {

	vu := &types.UndeployVAppParams{
		Xmlns:               "http://www.vmware.com/vcloud/v1.5",
		UndeployPowerAction: action,
	}

	output, err := xml.MarshalIndent(vu, "  ", "    ")
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}

	return ExecuteRequest(string(output),
		v.VM.HREF+"/action/undeploy",
		"POST",
		"application/vnd.vmware.vcloud.undeployVAppParams+xml",
		v.c)
}

func (v *VM) getVirtualHardwareItemsByResourceType(resourceType int) ([]*types.VirtualHardwareItem, error) {
	err := v.Refresh()
	if err != nil {
		return nil, fmt.Errorf("error refreshing VM before running customization: %v", err)
	}

	var items []*types.VirtualHardwareItem

	for _, item := range v.VM.VirtualHardwareSection.Item {
		if item.ResourceType == resourceType {
			items = append(items, item)
		}
	}

	return items, nil
}

func (v *VM) GetCPUCount() (int, error) {
	items, err := v.getVirtualHardwareItemsByResourceType(types.ResourceTypeProcessor)
	if err != nil {
		return 0, err
	}

	// The amount of CPU items must be one
	if len(items) != 1 {
		return 0, fmt.Errorf("error: Did not find any CPU on the given vm (%s)", v.VM.Name)
	}

	return items[0].VirtualQuantity, nil
}

func (v *VM) GetMemoryCount() (int, error) {
	items, err := v.getVirtualHardwareItemsByResourceType(types.ResourceTypeMemory)
	if err != nil {
		return 0, err
	}

	// The amount of memory items must be one
	if len(items) != 1 {
		return 0, fmt.Errorf("error: Did not find any Memory on the given vm (%s)", v.VM.Name)
	}

	return items[0].VirtualQuantity, nil
}

// func (v *VM) CorrectAddressOnParentForNetworkHardware() error {
// 	for index := range v.VM.VirtualHardwareSection.Item {
// 		if v.VM.VirtualHardwareSection.Item[index].ResourceType == types.ResourceTypeEthernet {
// 			v.VM.VirtualHardwareSection.Item[index].AddressOnParent = v.VM.VirtualHardwareSection.Item[index].InstanceID
// 		}
// 	}
// 	return nil
// }

func (v *VM) SetXMLNamespaces() {
	v.VM.Xmlns = types.XMLNamespaceXMLNS
	v.VM.Vcloud = types.XMLNamespaceVCloud
	v.VM.Ovf = types.XMLNamespaceOVF
	v.VM.Vmw = types.XMLNamespaceVMW
	v.VM.Xsi = types.XMLNamespaceXSI
	v.VM.Rasd = types.XMLNamespaceRASD
	v.VM.Vssd = types.XMLNamespaceVSSD
}

func (v *VM) SetCPUCount(count int) {
	for index := range v.VM.VirtualHardwareSection.Item {
		if v.VM.VirtualHardwareSection.Item[index].ResourceType == types.ResourceTypeProcessor {
			v.VM.VirtualHardwareSection.Item[index].ElementName = strconv.Itoa(count) + " virtual CPU(s)"
			v.VM.VirtualHardwareSection.Item[index].VirtualQuantity = count
		}
	}

	// Needs item list WITHOUT cpu item
	// item := &types.VirtualHardwareItem{
	// 	AllocationUnits: "hertz * 10^6",
	// 	Description:     "Number of Virtual CPUs",
	// 	ElementName:     strconv.Itoa(count) + " virtual CPU(s)",
	// 	ResourceType:    types.ResourceTypeProcessor,
	// 	VirtualQuantity: count,
	// 	CoresPerSocket:  1,
	// }

	// v.VM.VirtualHardwareSection.Item = append(v.VM.VirtualHardwareSection.Item, item)
}

func (v *VM) SetMemoryCount(count int) {
	for index := range v.VM.VirtualHardwareSection.Item {
		if v.VM.VirtualHardwareSection.Item[index].ResourceType == types.ResourceTypeMemory {
			v.VM.VirtualHardwareSection.Item[index].ElementName = strconv.Itoa(count) + " MB of memory"
			v.VM.VirtualHardwareSection.Item[index].VirtualQuantity = count
		}
	}

	// Needs item list WITHOUT memory item
	// item := &types.VirtualHardwareItem{
	// 	AllocationUnits: "byte * 2^20",
	// 	Description:     "Memory Size",
	// 	ElementName:     strconv.Itoa(count) + " MB of memory",
	// 	ResourceType:    types.ResourceTypeMemory,
	// 	VirtualQuantity: count,
	// }

	// v.VM.VirtualHardwareSection.Item = append(v.VM.VirtualHardwareSection.Item, item)
}

func (v *VM) SetNestedHypervisor(value bool) {
	v.VM.NestedHypervisorEnabled = value
}

func (v *VM) SetNestedHypervisorWithRequest(value bool) (Task, error) {
	url := ""
	if value {
		url = "/action/enableNestedHypervisor"
	} else {
		url = "/action/disableNestedHypervisor"
	}

	return ExecuteRequest("",
		v.VM.HREF+url,
		"POST",
		"application/vnd.vmware.vcloud.vm+xml",
		v.c)
}

func (v *VM) SetStorageProfile(storageProfile types.Reference) {
	v.VM.StorageProfile = &storageProfile
}

// func (v *VM) SetStorageProfileWithRequest(storageProfile types.Reference) (Task, error) {

// 	return Task{}, nil
// }

func (v *VM) SetInitscript(value string) {
	v.VM.GuestCustomizationSection.CustomizationScript = value
}

func (v *VM) SetNetworkConnectionSection(networks *types.NetworkConnectionSection) {
	v.VM.NetworkConnectionSection = networks
}

func (v *VM) SetAdminPasswordAuto(value bool) {
	v.VM.GuestCustomizationSection.AdminPasswordAuto = value
}

func (v *VM) SetAdminPassword(value string) {
	v.VM.GuestCustomizationSection.AdminPassword = value
}

func (v *VM) SetName(value string) {
	v.VM.Name = value
}

func (v *VM) SetHostName(value string) {
	v.VM.GuestCustomizationSection.ComputerName = value
}

func (v *VM) SetDescription(value string) {
	v.VM.Description = value
}

func (v *VM) SetNeedsCustomization(value bool) {
	v.VM.NeedsCustomization = value
}

func (v *VM) RemoveVirtualHardwareItemByResourceType(type_ types.ResourceType) {
	preservedItems := make([]*types.VirtualHardwareItem, 0)
	for _, item := range v.VM.VirtualHardwareSection.Item {
		if item.ResourceType != type_ {
			preservedItems = append(preservedItems, item)
		}
	}
	v.VM.VirtualHardwareSection.Item = preservedItems
}

// Attach or detach an independent disk
// Use the disk/action/attach or disk/action/detach links in a Vm to attach or detach an independent disk.
// Reference: vCloud API Programming Guide for Service Providers vCloud API 30.0 PDF Page 164 - 165,
// https://vdc-download.vmware.com/vmwb-repository/dcr-public/1b6cf07d-adb3-4dba-8c47-9c1c92b04857/
// 241956dd-e128-4fcc-8131-bf66e1edd895/vcloud_sp_api_guide_30_0.pdf
func (vm *VM) attachOrDetachDisk(diskParams *types.DiskAttachOrDetachParams, rel string) (Task, error) {
	var err error
	var execLink *types.Link
	for _, link := range vm.VM.Link {
		if link.Rel == rel && link.Type == types.MimeDiskAttachOrDetachParams {
			execLink = link
		}
	}

	reqUrl, err := url.ParseRequestURI(execLink.HREF)

	diskParams.Xmlns = types.NsVCloud

	xmlPayload, err := xml.Marshal(diskParams)
	if err != nil {
		return Task{}, fmt.Errorf("error marshal xml: %s", err)
	}

	// Send request
	req := vm.c.NewRequest(nil, http.MethodPost, *reqUrl, bytes.NewBufferString(xml.Header+string(xmlPayload)))

	req.Header.Add("Content-Type", execLink.Type)

	resp, err := checkResp(vm.c.Http.Do(req))
	if err != nil {
		return Task{}, fmt.Errorf("error attach or detach disk: %s", err)
	}

	// Decode response
	task := NewTask(vm.c)
	if err = decodeBody(resp, task.Task); err != nil {
		return Task{}, fmt.Errorf("error decoding Task response: %s", err)
	}

	// The request was successful
	return *task, nil
}

// Attach an independent disk
// Call attachOrDetachDisk with disk and types.RelDiskAttach to attach an independent disk.
// Please verify the independent disk is not connected to any VM before calling this function.
// If the independent disk is connected to a VM, the task will be failed.
// Reference: vCloud API Programming Guide for Service Providers vCloud API 30.0 PDF Page 164 - 165,
// https://vdc-download.vmware.com/vmwb-repository/dcr-public/1b6cf07d-adb3-4dba-8c47-9c1c92b04857/
// 241956dd-e128-4fcc-8131-bf66e1edd895/vcloud_sp_api_guide_30_0.pdf
func (vm *VM) AttachDisk(diskParams *types.DiskAttachOrDetachParams) (Task, error) {
	return vm.attachOrDetachDisk(diskParams, types.RelDiskAttach)
}

// Detach an independent disk
// Call attachOrDetachDisk with disk and types.RelDiskDetach to detach an independent disk.
// Please verify the independent disk is connected the VM before calling this function.
// If the independent disk is not connected to the VM, the task will be failed.
// Reference: vCloud API Programming Guide for Service Providers vCloud API 30.0 PDF Page 164 - 165,
// https://vdc-download.vmware.com/vmwb-repository/dcr-public/1b6cf07d-adb3-4dba-8c47-9c1c92b04857/
// 241956dd-e128-4fcc-8131-bf66e1edd895/vcloud_sp_api_guide_30_0.pdf
func (vm *VM) DetachDisk(diskParams *types.DiskAttachOrDetachParams) (Task, error) {
	return vm.attachOrDetachDisk(diskParams, types.RelDiskDetach)
}
