/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/url"
	"strconv"

	"github.com/kublr/govcloudair/types/v56"
)

// AdminCatalog is a admin view of a vCloud Director Catalog
type AdminCatalog struct {
	AdminCatalog *types.AdminCatalog
	c            *Client
}

func NewAdminCatalog(client *Client) *AdminCatalog {
	return &AdminCatalog{
		AdminCatalog: new(types.AdminCatalog),
		c:            client,
	}
}

// Updates the Catalog definition from current Catalog struct contents.
// Update automatically performs a refresh with the admin catalog it gets back from the rest api
func (adminCatalog *AdminCatalog) Update() error {
	vcomp := &types.AdminCatalog{
		Xmlns:       "http://www.vmware.com/vcloud/v1.5",
		Name:        adminCatalog.AdminCatalog.Name,
		Description: adminCatalog.AdminCatalog.Description,
		IsPublished: adminCatalog.AdminCatalog.IsPublished,
	}
	adminCatalogHREF, err := url.ParseRequestURI(adminCatalog.AdminCatalog.HREF)
	if err != nil {
		return fmt.Errorf("error parsing admin catalog's href: %v", err)
	}
	output, err := xml.MarshalIndent(vcomp, "  ", "    ")
	if err != nil {
		return fmt.Errorf("error marshalling xml data for update %v", err)
	}
	xmlData := bytes.NewBufferString(xml.Header + string(output))
	req := adminCatalog.c.NewRequest(map[string]string{}, "PUT", *adminCatalogHREF, xmlData)
	req.Header.Add("Content-Type", types.MimeAdminCatalog)
	resp, err := checkResp(adminCatalog.c.Http.Do(req))
	if err != nil {
		return fmt.Errorf("error updating catalog: %s : %s", err, adminCatalogHREF.Path)
	}
	defer resp.Body.Close()
	catalog := &types.AdminCatalog{}
	if err = decodeBody(resp, catalog); err != nil {
		return fmt.Errorf("error decoding update response: %s", err)
	}
	adminCatalog.AdminCatalog = catalog
	return nil
}

// Deletes the Catalog, returning an error if the vCD call fails.
func (adminCatalog *AdminCatalog) Delete(force, recursive bool) error {
	url, err := adminCatalog.AdminCatalog.Link.URLForType("", types.RelRemove)
	if err != nil {
		return err
	}

	req := adminCatalog.c.NewRequest(map[string]string{
		"force":     strconv.FormatBool(force),
		"recursive": strconv.FormatBool(recursive),
	}, "DELETE", *url, nil)

	_, err = checkResp(adminCatalog.c.Http.Do(req))
	if err != nil {
		return fmt.Errorf("error deleting Catalog %s: %s", adminCatalog.AdminCatalog.ID, err)
	}
	return nil
}
