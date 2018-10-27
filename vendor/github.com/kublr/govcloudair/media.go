/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	"io"
	"net/url"
	"strconv"

	"github.com/kublr/govcloudair/types/v56"
	"github.com/pkg/errors"
)

type Media struct {
	Media *types.Media
	c     *Client
}

func NewMedia(c *Client) *Media {
	return &Media{
		Media: new(types.Media),
		c:     c,
	}
}

func (m *Media) Refresh() error {
	mediaUrl, err := url.ParseRequestURI(m.Media.HREF)
	if err != nil {
		return errors.Wrapf(err, "cannot parse url: %s", m.Media.HREF)
	}

	req := m.c.NewRequest(map[string]string{}, "GET", *mediaUrl, nil)
	resp, err := checkResp(m.c.Http.Do(req))
	if err != nil {
		return errors.Wrapf(err, "cannot execute request: %s", m.Media.HREF)
	}

	newMedia := &types.Media{}
	if err = decodeBody(resp, newMedia); err != nil {
		return errors.Wrapf(err, "cannot unmarshal response: %s", m.Media.HREF)
	}

	m.Media = newMedia
	return nil
}

func (m *Media) EnableDownload() (Task, error) {
	link := m.Media.Link.ForType("", types.RelEnable)
	if link == nil {
		return Task{}, errors.Errorf("object does not have a link: ret=%s", types.RelEnable)
	}

	return ExecuteRequest("", link.HREF, "POST", "", m.c)
}

// Download  media content.
// Client is responsible for calling Read & Close
func (m *Media) Download() (io.ReadCloser, error) {
	if m.Media.Files == nil || len(m.Media.Files.File) < 1 {
		return nil, errors.New("media does not have any files")
	}

	link := m.Media.Files.File[0].Link.ForType("", types.RelDownloadDefault)
	if link == nil {
		return nil, errors.Errorf("object does not have a link: ret=%s", types.RelDownloadDefault)
	}

	mediaDownloadUrl, err := url.ParseRequestURI(link.HREF)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot parse url: %s", link.HREF)
	}

	req := m.c.NewRequest(map[string]string{}, "GET", *mediaDownloadUrl, nil)
	resp, err := checkResp(m.c.Http.Do(req))
	if err != nil {
		return nil, errors.Wrapf(err, "cannot execute request: %s", link.HREF)
	}

	return resp.Body, nil
}

// Upload media content.
// Client is responsible for cleaning in case of upload failure
func (m *Media) Upload(reader io.Reader, size int64) (Task, error) {
	link := m.Media.Files.File[0].Link.ForType("", types.RelUploadDefault)
	if link == nil {
		return Task{}, errors.Errorf("object does not have a link: ret=%s", types.RelUploadDefault)
	}

	mediaUploadUrl, err := url.ParseRequestURI(link.HREF)
	if err != nil {
		return Task{}, errors.Wrapf(err, "cannot parse url: %s", link.HREF)
	}

	task := Task{
		c:    m.c,
		Task: m.Media.Tasks.Task[0],
	}

	req := m.c.NewRequest(map[string]string{}, "PUT", *mediaUploadUrl, reader)
	req.Header.Add("Content-Length", strconv.FormatInt(size, 10))

	resp, err := checkResp(m.c.Http.Do(req))
	if err != nil {
		return task, errors.Wrapf(err, "cannot execute request: %s", link.HREF)
	}
	defer resp.Body.Close()

	return task, nil
}
