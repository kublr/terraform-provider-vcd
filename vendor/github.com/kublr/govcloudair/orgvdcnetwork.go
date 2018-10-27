/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/kublr/govcloudair/types/v56"
	"log"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// OrgVDCNetwork an org vdc network client
type OrgVDCNetwork struct {
	OrgVDCNetwork *types.OrgVDCNetwork
	c             *Client
}

// NewOrgVDCNetwork creates an org vdc network client
func NewOrgVDCNetwork(c *Client) *OrgVDCNetwork {
	return &OrgVDCNetwork{
		OrgVDCNetwork: new(types.OrgVDCNetwork),
		c:             c,
	}
}

func (o *OrgVDCNetwork) Refresh() error {
	if o.OrgVDCNetwork.HREF == "" {
		return fmt.Errorf("cannot refresh, Object is empty")
	}

	u, _ := url.ParseRequestURI(o.OrgVDCNetwork.HREF)

	req := o.c.NewRequest(map[string]string{}, "GET", *u, nil)

	resp, err := checkResp(o.c.Http.Do(req))
	if err != nil {
		return fmt.Errorf("error retrieving task: %s", err)
	}
	defer resp.Body.Close()

	// Empty struct before a new unmarshal, otherwise we end up with duplicate
	// elements in slices.
	o.OrgVDCNetwork = &types.OrgVDCNetwork{}

	if err = decodeBody(resp, o.OrgVDCNetwork); err != nil {
		return fmt.Errorf("error decoding task response: %s", err)
	}

	// The request was successful
	return nil
}

func (o *OrgVDCNetwork) Delete() (Task, error) {
	err := o.Refresh()
	if err != nil {
		return Task{}, fmt.Errorf("Error refreshing network: %s", err)
	}
	pathArr := strings.Split(o.OrgVDCNetwork.HREF, "/")
	s, _ := url.ParseRequestURI(o.OrgVDCNetwork.HREF)
	s.Path = "/api/admin/network/" + pathArr[len(pathArr)-1]

	req := o.c.NewRequest(map[string]string{}, "DELETE", *s, nil)
	resp, err := checkResp(o.c.Http.Do(req))
	if err != nil {
		if ok, _ := regexp.MatchString("is busy, cannot proceed with the operation.$", err.Error()); ok {
			time.Sleep(3 * time.Second)
			return o.Delete()
		}
		return Task{}, fmt.Errorf("error deleting Network: %s", err)
	}
	defer resp.Body.Close()

	task := NewTask(o.c)
	if err = decodeBody(resp, task.Task); err != nil {
		return Task{}, fmt.Errorf("error decoding Task response: %s", err)
	}

	// The request was successful
	return *task, nil
}

func (v *Vdc) CreateOrgVDCNetwork(networkConfig *types.OrgVDCNetwork) error {
	link := v.Vdc.Link.ForType("application/vnd.vmware.vcloud.orgVdcNetwork+xml", types.RelAdd)
	if link == nil {
		return errors.New("cannot find link for add orgVdcNetwork operation")
	}

	u, err := url.ParseRequestURI(link.HREF)
	if err != nil {
		return fmt.Errorf("error decoding vdc response: %s", err)
	}

	output, err := xml.MarshalIndent(networkConfig, "  ", "    ")
	if err != nil {
		return fmt.Errorf("error marshaling OrgVDCNetwork compose: %s", err)
	}

	b := bytes.NewBufferString(xml.Header + string(output))
	log.Printf("[DEBUG] VCD Client configuration: %s", b)

	req := v.c.NewRequest(map[string]string{}, "POST", *u, b)
	req.Header.Add("Content-Type", link.Type)

	resp, err := checkResp(v.c.Http.Do(req))
	if err != nil {
		if ok, _ := regexp.MatchString("is busy, cannot proceed with the operation.$", err.Error()); ok {
			time.Sleep(3 * time.Second)
			return v.CreateOrgVDCNetwork(networkConfig)
		}

		return fmt.Errorf("error instantiating a new OrgVDCNetwork: %s", err)
	}
	defer resp.Body.Close()

	newstuff := NewOrgVDCNetwork(v.c)
	if err = decodeBody(resp, newstuff.OrgVDCNetwork); err != nil {
		return fmt.Errorf("error decoding orgvdcnetwork response: %s", err)
	}

	task := NewTask(v.c)
	for _, t := range newstuff.OrgVDCNetwork.Tasks.Task {
		task.Task = t
		err = task.WaitTaskCompletion()
		if err != nil {
			return fmt.Errorf("error performing task: %#v", err)
		}
	}

	return nil
}
