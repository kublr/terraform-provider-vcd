package vcd

import (
	"net/url"
	"sync"

	govcd "github.com/kublr/govcloudair" // Forked from vmware/govcloudair
	"github.com/pkg/errors"
)

type Config struct {
	User            string
	Password        string
	Org             string
	Href            string
	VDC             string
	MaxRetryTimeout int
	InsecureFlag    bool
	ApiVersion      string
}

type VCDClient struct {
	*govcd.VCDClient

	Org    govcd.Org
	OrgVdc govcd.Vdc

	Mutex           sync.Mutex
	MaxRetryTimeout int
	InsecureFlag    bool
}

func (c *Config) Client() (*VCDClient, error) {
	u, err := url.ParseRequestURI(c.Href)
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot parse URL: %s", c.Href)
	}

	client := govcd.NewVCDClient(*u, c.InsecureFlag, c.ApiVersion)
	err = client.Authenticate(c.User, c.Password, c.Org)
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot authenticate in vCD: orgName=%s, userName=%s", c.Org, c.User)
	}

	org, err := client.GetOrg()
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot retrieve Org: orgName=%s", c.Org)
	}

	vdc, err := org.FindVDC(c.VDC)
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot retrieve VDC: vdcName=%s", c.VDC)
	}

	return &VCDClient{
		client,
		org,
		vdc,
		sync.Mutex{},
		c.MaxRetryTimeout,
		c.InsecureFlag,
	}, nil
}
