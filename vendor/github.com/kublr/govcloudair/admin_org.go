/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/url"

	"github.com/kublr/govcloudair/types/v56"
	"github.com/pkg/errors"
)

// AdminOrg gives an admin representation of an org.
// Administrators can delete and update orgs with an admin org object.
// AdminOrg includes all members of the Org element, and adds several
// elements that can be viewed and modified only by system administrators.
type AdminOrg struct {
	AdminOrg *types.AdminOrg
	c        *Client
}

// NewAdminOrg create an instance of AdminOrg
func NewAdminOrg(cli *Client) *AdminOrg {
	return &AdminOrg{
		AdminOrg: new(types.AdminOrg),
		c:        cli,
	}
}

// Refresh pull changes of AdminOrg
func (adminOrg *AdminOrg) Refresh() error {
	if *adminOrg == (AdminOrg{}) {
		return fmt.Errorf("cannot refresh, Object is empty")
	}
	adminOrgHREF, _ := url.ParseRequestURI(adminOrg.AdminOrg.HREF)
	req := adminOrg.c.NewRequest(map[string]string{}, "GET", *adminOrgHREF, nil)
	resp, err := checkResp(adminOrg.c.Http.Do(req))
	if err != nil {
		return fmt.Errorf("error performing request: %s", err)
	}
	defer resp.Body.Close()
	// Empty struct before a new unmarshal, otherwise we end up with duplicate
	// elements in slices.
	unmarshalledAdminOrg := &types.AdminOrg{}
	if err = decodeBody(resp, unmarshalledAdminOrg); err != nil {
		return fmt.Errorf("error decoding org response: %s", err)
	}
	adminOrg.AdminOrg = unmarshalledAdminOrg
	// The request was successful
	return nil
}

func (adminOrg *AdminOrg) FindAdminCatalog(catalogName string) (AdminCatalog, error) {
	ref := adminOrg.AdminOrg.Catalogs.Catalog.ForName(catalogName)

	if ref == nil {
		return AdminCatalog{}, errors.Errorf("cannot find catalog: %s", catalogName)
	}

	catalogURL, err := url.ParseRequestURI(ref.HREF)
	if err != nil {
		return AdminCatalog{}, fmt.Errorf("error decoding catalog url: %s", err)
	}
	req := adminOrg.c.NewRequest(map[string]string{}, "GET", *catalogURL, nil)
	resp, err := checkResp(adminOrg.c.Http.Do(req))
	if err != nil {
		return AdminCatalog{}, fmt.Errorf("error retrieving catalog: %s", err)
	}
	adminCatalog := NewAdminCatalog(adminOrg.c)
	if err = decodeBody(resp, adminCatalog.AdminCatalog); err != nil {
		return AdminCatalog{}, fmt.Errorf("error decoding catalog response: %s", err)
	}
	// The request was successful
	return *adminCatalog, nil

}

// CreateCatalog creates a catalog with given name and description under the
// the given organization. Returns a Task
//
func (adminOrg *AdminOrg) CreateCatalog(Name, Description string) (Task, error) {
	vcomp := &types.AdminCatalog{
		Xmlns:       "http://www.vmware.com/vcloud/v1.5",
		Name:        Name,
		Description: Description,
	}
	output, _ := xml.MarshalIndent(vcomp, "  ", "    ")
	xmlData := bytes.NewBufferString(xml.Header + string(output))

	catalogHREF, err := adminOrg.AdminOrg.Link.URLForType(types.MimeAdminCatalog, types.RelAdd)
	if err != nil {
		return Task{}, err
	}

	req := adminOrg.c.NewRequest(map[string]string{}, "POST", *catalogHREF, xmlData)
	req.Header.Add("Content-Type", types.MimeAdminCatalog)

	resp, err := checkResp(adminOrg.c.Http.Do(req))
	if err != nil {
		return Task{}, fmt.Errorf("error creating catalog: %s : %s", err, catalogHREF.Path)
	}
	defer resp.Body.Close()

	catalog := NewAdminCatalog(adminOrg.c)
	if err = decodeBody(resp, catalog.AdminCatalog); err != nil {
		return Task{}, fmt.Errorf("error decoding task response: %s", err)
	}

	if catalog.AdminCatalog.Tasks != nil && (*catalog.AdminCatalog.Tasks).Task != nil {
		for _, task := range (*catalog.AdminCatalog.Tasks).Task {
			if task.Type == types.MimeTask {
				result := NewTask(adminOrg.c)
				result.Task = task
				return *result, nil
			}
		}
	}

	return Task{}, fmt.Errorf("error creating catalog: no task found with type : %s", types.MimeTask)

}
