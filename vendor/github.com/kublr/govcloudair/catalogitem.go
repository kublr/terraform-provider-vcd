/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	"net/url"

	"github.com/kublr/govcloudair/types/v56"
	"github.com/pkg/errors"
)

type CatalogItem struct {
	CatalogItem *types.CatalogItem
	c           *Client
}

func NewCatalogItem(c *Client) *CatalogItem {
	return &CatalogItem{
		CatalogItem: new(types.CatalogItem),
		c:           c,
	}
}

func (ci *CatalogItem) GetVAppTemplate() (VAppTemplate, error) {
	u, err := url.ParseRequestURI(ci.CatalogItem.Entity.HREF)
	if err != nil {
		return VAppTemplate{}, errors.Wrapf(err, "cannot parse url: %s", ci.CatalogItem.Entity.HREF)
	}

	req := ci.c.NewRequest(map[string]string{}, "GET", *u, nil)
	resp, err := checkResp(ci.c.Http.Do(req))
	if err != nil {
		return VAppTemplate{}, errors.Wrapf(err, "cannot execute request: %s", ci.CatalogItem.Entity.HREF)
	}
	defer resp.Body.Close()

	cat := NewVAppTemplate(ci.c)
	if err = decodeBody(resp, cat.VAppTemplate); err != nil {
		return VAppTemplate{}, errors.Wrapf(err, "cannot unmarshal response: %s", ci.CatalogItem.Entity.HREF)
	}

	return *cat, nil
}

func (ci *CatalogItem) GetMedia() (Media, error) {
	entityType := ci.CatalogItem.Entity.Type
	if entityType != types.MimeMedia {
		return Media{}, errors.Errorf("wrong entity type: %s", entityType)
	}

	entityUrl, err := url.ParseRequestURI(ci.CatalogItem.Entity.HREF)
	if err != nil {
		return Media{}, errors.Wrapf(err, "cannot parse url: %s", ci.CatalogItem.Entity.HREF)
	}

	req := ci.c.NewRequest(map[string]string{}, "GET", *entityUrl, nil)
	resp, err := checkResp(ci.c.Http.Do(req))
	if err != nil {
		return Media{}, errors.Wrapf(err, "cannot execute request: %s", ci.CatalogItem.Entity.HREF)
	}

	media := NewMedia(ci.c)
	if err = decodeBody(resp, media.Media); err != nil {
		return Media{}, errors.Wrapf(err, "cannot unmarshal response: %s", ci.CatalogItem.Entity.HREF)
	}

	return *media, nil
}

func (ci *CatalogItem) Delete() error {
	link := ci.CatalogItem.Link.ForType("", types.RelRemove)
	if link == nil {
		return errors.Errorf("object does not have a link: ret=%s", types.RelRemove)
	}

	u, err := url.ParseRequestURI(link.HREF)
	if err != nil {
		return errors.Wrapf(err, "cannot parse url: %s", link.HREF)
	}

	req := ci.c.NewRequest(map[string]string{}, "DELETE", *u, nil)
	resp, err := checkResp(ci.c.Http.Do(req))
	if err != nil {
		return errors.Wrapf(err, "cannot execute request: %s", link.HREF)
	}
	defer resp.Body.Close()

	return nil
}
