package govcloudair

import (
	"crypto/tls"
	"fmt"
	"github.com/pkg/errors"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/kublr/govcloudair/types/v56"
)

type VCDClient struct {
	Client Client // Client for the underlying VCD instance

	orgHREF     url.URL // vCloud Director OrgRef
	queryHREF   url.URL // HREF for the query API
	sessionHREF url.URL // HREF for the session API
}

type supportedVersions struct {
	VersionInfo struct {
		Version  string `xml:"Version"`
		LoginUrl string `xml:"LoginUrl"`
	} `xml:"VersionInfo"`
}

type session struct {
	Link types.LinkList `xml:"Link"`
}

func (c *VCDClient) vcdloginurl() error {

	s := c.Client.VCDEndpoint
	s.Path += "/versions"

	// No point in checking for errors here
	req := c.Client.NewRequest(map[string]string{}, "GET", s, nil)

	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	supportedVersions := new(supportedVersions)

	err = decodeBody(resp, supportedVersions)

	if err != nil {
		return fmt.Errorf("error decoding versions response: %s", err)
	}

	u, err := url.Parse(supportedVersions.VersionInfo.LoginUrl)
	if err != nil {
		return fmt.Errorf("couldn't find a LoginUrl in versions")
	}
	c.sessionHREF = *u
	return nil
}

func (c *VCDClient) vcdauthorize(user, pass, org string) error {

	if user == "" {
		user = os.Getenv("VCLOUD_USERNAME")
	}

	if pass == "" {
		pass = os.Getenv("VCLOUD_PASSWORD")
	}

	if org == "" {
		org = os.Getenv("VCLOUD_ORG")
	}

	// No point in checking for errors here
	req := c.Client.NewRequest(map[string]string{}, "POST", c.sessionHREF, nil)

	// Set Basic Authentication Header
	req.SetBasicAuth(user+"@"+org, pass)

	// Add the Accept header for vCA
	req.Header.Add(GetVersionHeader(types.ApiVersion))

	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Store the authentication header
	c.Client.VCDToken = resp.Header.Get("x-vcloud-authorization")
	c.Client.VCDAuthHeader = "x-vcloud-authorization"

	session := new(session)
	err = decodeBody(resp, session)

	if err != nil {
		return fmt.Errorf("error decoding session response: %s", err)
	}

	orgLink := session.Link.ForName(org, types.MimeOrg, types.RelDown)
	if orgLink == nil {
		return fmt.Errorf("cannot find a Org endpoint: name=%s type=%s, rel=%s", org, types.MimeOrg, types.RelDown)
	}
	u, err := url.Parse(orgLink.HREF)
	if err != nil {
		return errors.Wrapf(err, "cannot parse url: %s", orgLink.HREF)
	}
	c.orgHREF = *u

	queryLink := session.Link.ForType(types.MimeQueryList, types.RelDown)
	if queryLink == nil {
		return fmt.Errorf("cannot find a Query endpoint: type=%s, rel=%s", types.MimeQueryList, types.RelDown)
	}
	u, err = url.Parse(queryLink.HREF)
	if err != nil {
		return errors.Wrapf(err, "cannot parse url: %s", queryLink.HREF)
	}
	c.queryHREF = *u

	sessionLink := session.Link.ForType("", types.RelRemove)
	if sessionLink == nil {
		return fmt.Errorf("cannot find a LogOut endpoint: rel=%s", types.RelRemove)
	}
	u, err = url.Parse(sessionLink.HREF)
	if err != nil {
		return errors.Wrapf(err, "cannot parse url: %s", sessionLink.HREF)
	}
	c.sessionHREF = *u

	return nil
}

func NewVCDClient(vcdEndpoint url.URL, insecure bool, apiVersion string) *VCDClient {
	return &VCDClient{
		Client: Client{
			APIVersion:  apiVersion,
			VCDEndpoint: vcdEndpoint,
			Http: http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: insecure,
					},
					Proxy:               http.ProxyFromEnvironment,
					TLSHandshakeTimeout: 120 * time.Second,
				},
			},
		},
	}
}

// Authenticate is an helper function that performs a login in vCloud Director.
func (c *VCDClient) Authenticate(username, password, org string) error {

	// LoginUrl
	err := c.vcdloginurl()
	if err != nil {
		return fmt.Errorf("error finding LoginUrl: %s", err)
	}
	// Authorize
	err = c.vcdauthorize(username, password, org)
	if err != nil {
		return fmt.Errorf("error authorizing: %s", err)
	}

	return nil
}

// Disconnect performs a disconnection from the vCloud Director API endpoint.
func (c *VCDClient) Disconnect() error {
	if c.Client.VCDToken == "" && c.Client.VCDAuthHeader == "" {
		return fmt.Errorf("cannot disconnect, client is not authenticated")
	}

	req := c.Client.NewRequest(map[string]string{}, "DELETE", c.sessionHREF, nil)

	// Add the Accept header for vCA
	req.Header.Add(GetVersionHeader(types.ApiVersion90))

	// Set Authorization Header
	req.Header.Add(c.Client.VCDAuthHeader, c.Client.VCDToken)

	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return fmt.Errorf("error processing session delete for vCloud Director: %s", err)
	}
	defer resp.Body.Close()

	return nil
}

// GetOrg returns Org object named c.orgName
func (c *VCDClient) GetOrg() (Org, error) {

	req := c.Client.NewRequest(map[string]string{}, "GET", c.orgHREF, nil)
	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return Org{}, errors.Wrapf(err, "cannot execute request: %s", c.orgHREF.String())
	}
	defer resp.Body.Close()
	org := NewOrg(&c.Client)
	if err = decodeBody(resp, org.Org); err != nil {
		return Org{}, errors.Wrapf(err, "cannot unmarshal response")
	}
	return *org, nil
}

// GetAdminOrg returns AdminOrg object linked from Org
// AdminOrg used for create, update and delete operations
func (c *VCDClient) GetAdminOrg() (AdminOrg, error) {
	org, err := c.GetOrg()
	if err != nil {
		return AdminOrg{}, errors.Wrapf(err, "cannot get org: %s", c.orgHREF.String())
	}

	adminOrgHREF, err := org.Org.Link.URLForType(types.MimeAdminOrg, types.RelAlternate)
	if err != nil {
		return AdminOrg{}, err
	}

	req := c.Client.NewRequest(map[string]string{}, "GET", *adminOrgHREF, nil)
	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return AdminOrg{}, errors.Wrapf(err, "cannot execute request: %s", (*adminOrgHREF).String())
	}
	defer resp.Body.Close()
	adminOrg := NewAdminOrg(&c.Client)
	if err = decodeBody(resp, adminOrg.AdminOrg); err != nil {
		return AdminOrg{}, errors.Wrapf(err, "cannot unmarshal response")
	}
	return *adminOrg, nil
}
