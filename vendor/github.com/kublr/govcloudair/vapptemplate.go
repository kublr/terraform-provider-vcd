/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/kublr/govcloudair/types/v56"
	"github.com/pkg/errors"
	"net/url"
)

type VAppTemplate struct {
	VAppTemplate *types.VAppTemplate
	c            *Client
}

func NewVAppTemplate(c *Client) *VAppTemplate {
	return &VAppTemplate{
		VAppTemplate: new(types.VAppTemplate),
		c:            c,
	}
}

func (v *Vdc) InstantiateVAppTemplate(template *types.InstantiateVAppTemplateParams) error {
	output, err := xml.MarshalIndent(template, "", "  ")
	if err != nil {
		return fmt.Errorf("Error finding VAppTemplate: %#v", err)
	}
	b := bytes.NewBufferString(xml.Header + string(output))

	link := v.Vdc.Link.ForType(types.MimeInstantiateVAppTemplate, types.RelAdd)
	if link == nil {
		return errors.Errorf("cannot find endpoint: type=%s, rel=%s", types.MimeInstantiateVAppTemplate, types.RelAdd)
	}

	u, err := url.Parse(link.HREF)
	if err != nil {
		return errors.Wrapf(err, "cannot parse url: %s", link.HREF)
	}

	req := v.c.NewRequest(map[string]string{}, "POST", *u, b)
	req.Header.Add("Content-Type", types.MimeInstantiateVAppTemplate)

	resp, err := checkResp(v.c.Http.Do(req))
	if err != nil {
		return fmt.Errorf("error instantiating a new template: %s", err)
	}
	defer resp.Body.Close()

	vapptemplate := NewVAppTemplate(v.c)
	if err = decodeBody(resp, vapptemplate.VAppTemplate); err != nil {
		return fmt.Errorf("error decoding orgvdcnetwork response: %s", err)
	}
	task := NewTask(v.c)
	for _, t := range vapptemplate.VAppTemplate.Tasks.Task {
		task.Task = t
		err = task.WaitTaskCompletion()
		if err != nil {
			return fmt.Errorf("Error performing task: %#v", err)
		}
	}
	return nil
}
