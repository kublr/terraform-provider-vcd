/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	"fmt"
	"net/url"

	"github.com/kublr/govcloudair/types/v56"
	"github.com/pkg/errors"
)

type Org struct {
	Org *types.Org
	c   *Client
}

func NewOrg(c *Client) *Org {
	return &Org{
		Org: new(types.Org),
		c:   c,
	}
}

func (o *Org) Refresh() error {
	u, err := url.ParseRequestURI(o.Org.HREF)
	if err != nil {
		return errors.Wrapf(err, "cannot parse url: %s", o.Org.HREF)
	}

	req := o.c.NewRequest(map[string]string{}, "GET", *u, nil)
	resp, err := checkResp(o.c.Http.Do(req))
	if err != nil {
		return errors.Wrapf(err, "cannot execute request: %s", o.Org.HREF)
	}

	newOrg := &types.Org{}
	if err = decodeBody(resp, newOrg); err != nil {
		return errors.Wrap(err, "cannot unmarshal response")
	}

	o.Org = newOrg
	return nil
}

func (o *Org) FindCatalog(catalogName string) (Catalog, error) {
	link := o.Org.Link.ForName(catalogName, types.MimeCatalog, types.RelDown)
	if link == nil {
		return Catalog{}, fmt.Errorf("cannot find Catalog endpoint: name=%s, type=%s, rel=%s", catalogName, types.MimeCatalog, types.RelDown)
	}

	u, err := url.ParseRequestURI(link.HREF)
	if err != nil {
		return Catalog{}, errors.Wrapf(err, "cannot parse url: %s", link.HREF)
	}

	req := o.c.NewRequest(map[string]string{}, "GET", *u, nil)
	resp, err := checkResp(o.c.Http.Do(req))
	if err != nil {
		return Catalog{}, errors.Wrapf(err, "cannot execute request: %s", link.HREF)
	}
	defer resp.Body.Close()

	catalog := NewCatalog(o.c)
	if err = decodeBody(resp, catalog.Catalog); err != nil {
		return Catalog{}, errors.Wrap(err, "cannot unmarshal response")
	}

	return *catalog, nil
}

func (o *Org) FindVDC(vdcName string) (Vdc, error) {
	link := o.Org.Link.ForName(vdcName, types.MimeVDC, types.RelDown)
	if link == nil {
		return Vdc{}, fmt.Errorf("cannot find VDC endpoint: name=%s, type=%s, rel=%s", vdcName, types.MimeVDC, types.RelDown)
	}

	u, err := url.ParseRequestURI(link.HREF)
	if err != nil {
		return Vdc{}, errors.Wrapf(err, "cannot parse url: %s", link.HREF)
	}

	req := o.c.NewRequest(map[string]string{}, "GET", *u, nil)
	resp, err := checkResp(o.c.Http.Do(req))
	if err != nil {
		return Vdc{}, errors.Wrapf(err, "cannot execute request: %s", link.HREF)
	}
	defer resp.Body.Close()

	vdc := NewVdc(o.c)
	if err = decodeBody(resp, vdc.Vdc); err != nil {
		return Vdc{}, errors.Wrap(err, "cannot unmarshal response")
	}

	return *vdc, nil
}
