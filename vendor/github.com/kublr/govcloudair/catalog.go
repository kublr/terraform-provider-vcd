/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	"bytes"
	"encoding/xml"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/kublr/govcloudair/types/v56"
	"github.com/pkg/errors"
)

type Catalog struct {
	Catalog *types.Catalog
	c       *Client
}

func NewCatalog(c *Client) *Catalog {
	return &Catalog{
		Catalog: new(types.Catalog),
		c:       c,
	}
}

func (c *Catalog) Refresh() error {
	catalogUrl, err := url.ParseRequestURI(c.Catalog.HREF)
	if err != nil {
		return errors.Wrapf(err, "cannot parse url: %s", c.Catalog.HREF)
	}

	req := c.c.NewRequest(map[string]string{}, "GET", *catalogUrl, nil)
	resp, err := checkResp(c.c.Http.Do(req))
	if err != nil {
		return errors.Wrapf(err, "cannot execute request: %s", c.Catalog.HREF)
	}

	newCatalog := &types.Catalog{}
	if err = decodeBody(resp, newCatalog); err != nil {
		return errors.Wrapf(err, "cannot unmarshal response: %s", c.Catalog.HREF)
	}

	c.Catalog = newCatalog
	return nil
}

func (c *Catalog) HasCatalogItem(catalogItemName string) bool {
	for _, cis := range c.Catalog.CatalogItems {
		ref := cis.CatalogItem.ForName(catalogItemName)
		if ref != nil {
			return true
		}
	}

	return false
}

func (c *Catalog) FindCatalogItem(catalogItemName string) (CatalogItem, error) {
	var ref *types.Reference
	for _, cis := range c.Catalog.CatalogItems {
		ref = cis.CatalogItem.ForName(catalogItemName)
		if ref != nil {
			break
		}
	}
	if ref == nil {
		return CatalogItem{}, errors.Errorf("cannot find catalog item: %s", catalogItemName)
	}

	u, err := url.ParseRequestURI(ref.HREF)
	if err != nil {
		return CatalogItem{}, errors.Wrapf(err, "cannot parse url: %s", ref.HREF)
	}

	req := c.c.NewRequest(map[string]string{}, "GET", *u, nil)
	resp, err := checkResp(c.c.Http.Do(req))
	if err != nil {
		return CatalogItem{}, errors.Wrapf(err, "cannot execute request: %s", ref.HREF)
	}
	defer resp.Body.Close()

	cat := NewCatalogItem(c.c)
	if err = decodeBody(resp, cat.CatalogItem); err != nil {
		return CatalogItem{}, errors.Wrapf(err, "cannot unmarshal response: %s", ref.HREF)
	}

	return *cat, nil
}

func (c *Catalog) UploadMedia(mediaName string, reader io.Reader) (Task, error) {
	mediaType := strings.TrimPrefix(filepath.Ext(mediaName), ".")
	if mediaType == "" {
		mediaType = "floppy"
	}

	var mediaBuffer bytes.Buffer
	mediaSize, err := io.Copy(&mediaBuffer, reader)
	if err != nil {
		return Task{}, errors.Wrap(err, "cannot read media content")
	}

	bodyXml, err := xml.MarshalIndent(types.Media{
		Xmlns:     types.NsVCloud,
		Name:      mediaName,
		ImageType: mediaType,
		Size:      mediaSize,
	}, "", "  ")
	if err != nil {
		return Task{}, err
	}
	body := bytes.NewBufferString(xml.Header + string(bodyXml))

	link := c.Catalog.Link.ForType(types.MimeMedia, types.RelAdd)
	if link == nil {
		return Task{}, errors.Errorf("object does not have a link: type=%s, ret=%s", types.MimeMedia, types.RelAdd)
	}

	u, err := url.ParseRequestURI(link.HREF)
	if err != nil {
		return Task{}, errors.Wrapf(err, "cannot parse url: %s", link.HREF)
	}

	req := c.c.NewRequest(map[string]string{}, "POST", *u, body)
	req.Header.Add("Content-Type", types.MimeMedia)
	resp, err := checkResp(c.c.Http.Do(req))
	if err != nil {
		return Task{}, errors.Wrapf(err, "cannot execute request: %s", link.HREF)
	}
	defer resp.Body.Close()

	catalogItem := NewCatalogItem(c.c)
	if err = decodeBody(resp, catalogItem.CatalogItem); err != nil {
		return Task{}, errors.Wrapf(err, "cannot unmarshal response: %s", link.HREF)
	}

	media, err := catalogItem.GetMedia()
	if err != nil {
		return Task{}, err
	}

	return media.Upload(&mediaBuffer, mediaSize)
}
