/*
 * Copyright 2016 Skyscape Cloud Services.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	"github.com/pkg/errors"

	"github.com/kublr/govcloudair/types/v56"
)

type Results struct {
	Results *types.QueryResultRecordsType
	c       *Client
}

func NewResults(c *Client) *Results {
	return &Results{
		Results: new(types.QueryResultRecordsType),
		c:       c,
	}
}

func (c *VCDClient) Query(params map[string]string) (Results, error) {
	req := c.Client.NewRequest(params, "GET", c.queryHREF, nil)
	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return Results{}, errors.Wrapf(err, "cannot execute request: %s", c.queryHREF.String())
	}
	defer resp.Body.Close()

	results := NewResults(&c.Client)
	if err = decodeBody(resp, results.Results); err != nil {
		return Results{}, errors.Wrap(err, "cannot unmarshal response")
	}

	return *results, nil
}
