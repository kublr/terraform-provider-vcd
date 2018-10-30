package vcd

import (
	"fmt"
	"github.com/pkg/errors"
)

func findDefaultStorageProfile(vcdClient *VCDClient) (string, error) {
	queryParams := map[string]string{
		"type":          "orgVdcStorageProfile",
		"format":        "records",
		"filter":        fmt.Sprintf("(vdcName==%s;isDefaultStorageProfile==true)", vcdClient.OrgVdc.Vdc.Name),
		"filterEncoded": "true",
	}

	query, err := vcdClient.Query(queryParams)
	if err != nil {
		return "", errors.Wrapf(err, "cannot execute query: %v", queryParams)
	}

	records := query.Results.OrgVdcStorageProfileRecord
	if len(records) < 1 {
		return "", fmt.Errorf("no storage profiles found: vdcName%s", vcdClient.OrgVdc.Vdc.Name)
	}

	return records[0].Name, nil
}
