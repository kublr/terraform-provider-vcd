/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/pkg/errors"
	"log"
	"net/url"
	"os"

	"github.com/kublr/govcloudair/types/v56"
)

type Vdc struct {
	Vdc *types.Vdc
	c   *Client
}

func NewVdc(c *Client) *Vdc {
	return &Vdc{
		Vdc: new(types.Vdc),
		c:   c,
	}
}

func (v *Vdc) Refresh() error {

	if v.Vdc.HREF == "" {
		return fmt.Errorf("cannot refresh, Object is empty")
	}

	u, _ := url.ParseRequestURI(v.Vdc.HREF)

	req := v.c.NewRequest(map[string]string{}, "GET", *u, nil)

	resp, err := checkResp(v.c.Http.Do(req))
	if err != nil {
		return fmt.Errorf("error retreiving Edge Gateway: %s", err)
	}
	defer resp.Body.Close()

	// Empty struct before a new unmarshal, otherwise we end up with duplicate
	// elements in slices.
	unmarshalledVdc := &types.Vdc{}

	if err = decodeBody(resp, unmarshalledVdc); err != nil {
		return fmt.Errorf("error decoding vdc response: %s", err)
	}

	v.Vdc = unmarshalledVdc

	// The request was successful
	return nil
}

func (v *Vdc) FindVDCNetwork(network string) (OrgVDCNetwork, error) {
	var ref *types.Reference
	for _, an := range v.Vdc.AvailableNetworks {
		ref = an.Network.ForName(network)
		if ref != nil {
			break
		}
	}

	if ref == nil {
		return OrgVDCNetwork{}, fmt.Errorf("can't find VDC Network: %s", network)
	}

	u, err := url.ParseRequestURI(ref.HREF)
	if err != nil {
		return OrgVDCNetwork{}, fmt.Errorf("error decoding vdc response: %s", err)
	}

	req := v.c.NewRequest(map[string]string{}, "GET", *u, nil)
	resp, err := checkResp(v.c.Http.Do(req))
	if err != nil {
		return OrgVDCNetwork{}, fmt.Errorf("error retreiving orgvdcnetwork: %s", err)
	}
	defer resp.Body.Close()

	orgnet := NewOrgVDCNetwork(v.c)
	if err = decodeBody(resp, orgnet.OrgVDCNetwork); err != nil {
		return OrgVDCNetwork{}, fmt.Errorf("error decoding orgvdcnetwork response: %s", err)
	}

	// The request was successful
	return *orgnet, nil
}

func (v *Vdc) FindStorageProfileReference(name string) (types.Reference, error) {
	var ref *types.Reference
	for _, sps := range v.Vdc.VdcStorageProfiles {
		ref = sps.VdcStorageProfile.ForName(name)
		if ref != nil {
			break
		}
	}

	if ref == nil {
		return types.Reference{}, fmt.Errorf("can't find VDC Storage_profile: %s", name)
	}

	return *ref, nil
}

// Doesn't work with vCloud API 5.5, only vCloud Air
func (v *Vdc) GetVDCOrg() (Org, error) {
	link := v.Vdc.Link.ForType(types.MimeOrg, types.RelUp)
	if link == nil {
		return Org{}, fmt.Errorf("can't find VDC Org")
	}

	u, err := url.ParseRequestURI(link.HREF)
	if err != nil {
		return Org{}, fmt.Errorf("error decoding vdc response: %s", err)
	}

	req := v.c.NewRequest(map[string]string{}, "GET", *u, nil)
	resp, err := checkResp(v.c.Http.Do(req))
	if err != nil {
		return Org{}, fmt.Errorf("error retreiving org: %s", err)
	}
	defer resp.Body.Close()

	org := NewOrg(v.c)
	if err = decodeBody(resp, org.Org); err != nil {
		return Org{}, fmt.Errorf("error decoding org response: %s", err)
	}

	// The request was successful
	return *org, nil
}

func (v *Vdc) FindEdgeGateway(edgegateway string) (EdgeGateway, error) {
	link := v.Vdc.Link.ForType("application/vnd.vmware.vcloud.query.records+xml", "edgeGateways")
	if link == nil {
		return EdgeGateway{}, fmt.Errorf("can't find Edge Gateway")
	}

	u, err := url.ParseRequestURI(link.HREF)
	if err != nil {
		return EdgeGateway{}, fmt.Errorf("error decoding vdc response: %s", err)
	}

	// Querying the Result list
	req := v.c.NewRequest(map[string]string{}, "GET", *u, nil)
	resp, err := checkResp(v.c.Http.Do(req))
	if err != nil {
		return EdgeGateway{}, fmt.Errorf("error retrieving edge gateway records: %s", err)
	}
	defer resp.Body.Close()

	query := new(types.QueryResultEdgeGatewayRecordsType)
	if err = decodeBody(resp, query); err != nil {
		return EdgeGateway{}, fmt.Errorf("error decoding edge gateway query response: %s", err)
	}

	var href string
	for _, edge := range query.EdgeGatewayRecord {
		if edge.Name == edgegateway {
			href = edge.HREF
		}
	}

	if href == "" {
		return EdgeGateway{}, fmt.Errorf("can't find edge gateway with name: %s", edgegateway)
	}

	u, err = url.ParseRequestURI(href)
	if err != nil {
		return EdgeGateway{}, fmt.Errorf("error decoding edge gateway query response: %s", err)
	}

	// Querying the Result list
	req = v.c.NewRequest(map[string]string{}, "GET", *u, nil)
	resp, err = checkResp(v.c.Http.Do(req))
	if err != nil {
		return EdgeGateway{}, fmt.Errorf("error retrieving edge gateway: %s", err)
	}
	defer resp.Body.Close()

	edge := NewEdgeGateway(v.c)
	if err = decodeBody(resp, edge.EdgeGateway); err != nil {
		return EdgeGateway{}, fmt.Errorf("error decoding edge gateway response: %s", err)
	}

	return *edge, nil
}

func (v *Vdc) ComposeRawVApp(name string) error {
	vcomp := &types.ComposeVAppParams{
		Ovf:     "http://schemas.dmtf.org/ovf/envelope/1",
		Xsi:     "http://www.w3.org/2001/XMLSchema-instance",
		Xmlns:   "http://www.vmware.com/vcloud/v1.5",
		Deploy:  false,
		Name:    name,
		PowerOn: false,
	}

	output, err := xml.MarshalIndent(vcomp, "  ", "    ")
	if err != nil {
		return fmt.Errorf("error marshaling vapp compose: %s", err)
	}

	debug := os.Getenv("GOVCLOUDAIR_DEBUG")

	if debug == "true" {
		fmt.Printf("\n\nXML DEBUG: %s\n\n", string(output))
	}

	b := bytes.NewBufferString(xml.Header + string(output))

	link := v.Vdc.Link.ForType(types.MimeComposeVAppParams, types.RelAdd)
	if link == nil {
		return errors.Errorf("cannot find endpoint: type=%s, rel=%s", types.MimeComposeVAppParams, types.RelAdd)
	}

	u, err := url.Parse(link.HREF)
	if err != nil {
		return errors.Wrapf(err, "cannot parse url: %s", link.HREF)
	}

	req := v.c.NewRequest(map[string]string{}, "POST", *u, b)
	req.Header.Add("Content-Type", types.MimeComposeVAppParams)

	resp, err := checkResp(v.c.Http.Do(req))
	if err != nil {
		return fmt.Errorf("error instantiating a new vApp: %s", err)
	}
	defer resp.Body.Close()

	task := NewTask(v.c)
	if err = decodeBody(resp, task.Task); err != nil {
		return fmt.Errorf("error decoding task response: %s", err)
	}

	err = task.WaitTaskCompletion()
	if err != nil {
		return fmt.Errorf("Error performing task: %#v", err)
	}

	return nil
}

func (v *Vdc) FindVAppByName(vapp string) (VApp, error) {

	err := v.Refresh()
	if err != nil {
		return VApp{}, fmt.Errorf("error refreshing vdc: %s", err)
	}

	var ref *types.ResourceReference
	for _, resents := range v.Vdc.ResourceEntities {
		for _, resent := range resents.ResourceEntity {
			if resent.Name == vapp && resent.Type == "application/vnd.vmware.vcloud.vApp+xml" {
				ref = resent
				break
			}
		}
	}

	if ref == nil {
		return VApp{}, fmt.Errorf("can't find vApp: %s", vapp)
	}

	u, err := url.ParseRequestURI(ref.HREF)
	if err != nil {
		return VApp{}, fmt.Errorf("error decoding vdc response: %s", err)
	}

	// Querying the VApp
	req := v.c.NewRequest(map[string]string{}, "GET", *u, nil)
	resp, err := checkResp(v.c.Http.Do(req))
	if err != nil {
		return VApp{}, fmt.Errorf("error retrieving vApp: %s", err)
	}
	defer resp.Body.Close()

	newvapp := NewVApp(v.c)
	if err = decodeBody(resp, newvapp.VApp); err != nil {
		return VApp{}, fmt.Errorf("error decoding vApp response: %s", err.Error())
	}

	return *newvapp, nil
}

func (v *Vdc) FindVMByName(vapp VApp, vmName string) (VM, error) {
	err := v.Refresh()
	if err != nil {
		return VM{}, fmt.Errorf("error refreshing vdc: %s", err)
	}

	err = vapp.Refresh()
	if err != nil {
		return VM{}, fmt.Errorf("error refreshing vapp: %s", err)
	}

	//vApp Might Not Have Any VMs
	if vapp.VApp.Children == nil {
		return VM{}, fmt.Errorf("VApp Has No VMs")
	}

	log.Printf("[TRACE] Looking for VM: %s", vmName)
	var vm *types.VM
	for _, child := range vapp.VApp.Children.VM {
		if child.Name == vmName {
			vm = child
			break
		}
	}

	if vm == nil {
		return VM{}, fmt.Errorf("can't find vm: %s", vmName)
	}

	u, err := url.ParseRequestURI(vm.HREF)
	if err != nil {
		return VM{}, fmt.Errorf("error decoding vdc response: %s", err)
	}

	// Querying the VApp
	req := v.c.NewRequest(map[string]string{}, "GET", *u, nil)
	resp, err := checkResp(v.c.Http.Do(req))
	if err != nil {
		return VM{}, fmt.Errorf("error retrieving vm: %s", err)
	}
	defer resp.Body.Close()

	newvm := NewVM(v.c)
	if err = decodeBody(resp, newvm.VM); err != nil {
		return VM{}, fmt.Errorf("error decoding vm response: %s", err.Error())
	}

	return *newvm, nil
}

func (v *Vdc) GetVMByHREF(vmhref string) (VM, error) {

	u, err := url.ParseRequestURI(vmhref)

	if err != nil {
		return VM{}, fmt.Errorf("error decoding vm HREF: %s", err)
	}

	// Querying the VApp
	req := v.c.NewRequest(map[string]string{}, "GET", *u, nil)

	resp, err := checkResp(v.c.Http.Do(req))
	if err != nil {
		return VM{}, fmt.Errorf("error retrieving VM: %s", err)
	}
	defer resp.Body.Close()

	newvm := NewVM(v.c)

	if err = decodeBody(resp, newvm.VM); err != nil {
		return VM{}, fmt.Errorf("error decoding VM response: %s", err)
	}

	return *newvm, nil
}

func (v *Vdc) GetVAppByHREF(vmhref string) (VApp, error) {
	u, err := url.ParseRequestURI(vmhref)

	if err != nil {
		return VApp{}, fmt.Errorf("error decoding vm HREF: %s", err)
	}

	// Querying the VApp
	req := v.c.NewRequest(map[string]string{}, "GET", *u, nil)

	resp, err := checkResp(v.c.Http.Do(req))
	if err != nil {
		return VApp{}, fmt.Errorf("error retrieving VApp: %s", err)
	}
	defer resp.Body.Close()

	newVApp := NewVApp(v.c)

	if err = decodeBody(resp, newVApp.VApp); err != nil {
		return VApp{}, fmt.Errorf("error decoding VApp response: %s", err)
	}

	return *newVApp, nil
}
