/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	"fmt"
	"net/url"

	"github.com/kublr/govcloudair/types/v56"
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

func (o *Org) FindCatalog(catalog string) (Catalog, error) {
	link := o.Org.Link.ForName(catalog, types.MimeCatalog, types.RelDown)
	if link == nil {
		return Catalog{}, fmt.Errorf("can't find catalog: %s", catalog)
	}

	u, err := url.ParseRequestURI(link.HREF)
	if err != nil {
		return Catalog{}, fmt.Errorf("error decoding org response: %s", err)
	}

	req := o.c.NewRequest(map[string]string{}, "GET", *u, nil)
	resp, err := checkResp(o.c.Http.Do(req))
	if err != nil {
		return Catalog{}, fmt.Errorf("error retreiving catalog: %s", err)
	}
	defer resp.Body.Close()

	cat := NewCatalog(o.c)
	if err = decodeBody(resp, cat.Catalog); err != nil {
		return Catalog{}, fmt.Errorf("error decoding catalog response: %s", err)
	}

	// The request was successful
	return *cat, nil
}
