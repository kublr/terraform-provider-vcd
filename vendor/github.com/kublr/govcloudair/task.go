/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	"net/url"
	"time"

	"github.com/kublr/govcloudair/types/v56"
	"github.com/pkg/errors"
)

type Task struct {
	Task *types.Task
	c    *Client
}

func NewTask(c *Client) *Task {
	return &Task{
		Task: new(types.Task),
		c:    c,
	}
}

func (t *Task) Refresh() error {
	u, err := url.ParseRequestURI(t.Task.HREF)
	if err != nil {
		return errors.Wrapf(err, "cannot parse url: %s", t.Task.HREF)
	}

	req := t.c.NewRequest(map[string]string{}, "GET", *u, nil)
	resp, err := checkResp(t.c.Http.Do(req))
	if err != nil {
		return errors.Wrapf(err, "cannot execute request: %s", t.Task.HREF)
	}
	defer resp.Body.Close()

	newTask := &types.Task{}
	if err = decodeBody(resp, newTask); err != nil {
		return errors.Wrapf(err, "cannot unmarshal response: %s", t.Task.HREF)
	}

	t.Task = newTask
	return nil
}

func (t *Task) WaitTaskCompletion() error {
	for {
		err := t.Refresh()
		if err != nil {
			return err
		}

		// If task is not in a waiting status we're done, check if there's an error and return it.
		if t.Task.Status != "queued" && t.Task.Status != "preRunning" && t.Task.Status != "running" {
			if t.Task.Status != "success" {
				return errors.Errorf("task %s did not completed successfully: %s", t.Task.Name, t.Task.Description)
			}

			return nil
		}

		time.Sleep(1 * time.Second)
	}
}

func (t *Task) Cancel() error {
	u, err := url.ParseRequestURI(t.Task.Link.HREF)
	if err != nil {
		return errors.Wrapf(err, "cannot parse url: %s", t.Task.Link.HREF)
	}

	req := t.c.NewRequest(map[string]string{}, "POST", *u, nil)
	resp, err := checkResp(t.c.Http.Do(req))
	if err != nil {
		return errors.Wrapf(err, "cannot execute request: %s", t.Task.Link.HREF)
	}
	defer resp.Body.Close()

	return nil
}
