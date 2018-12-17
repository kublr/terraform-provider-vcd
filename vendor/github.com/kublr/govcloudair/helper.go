package govcloudair

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/kublr/govcloudair/types/v56"
)

func GetVersionHeader(version types.ApiVersionType) (key, value string) {
	return "Accept", fmt.Sprintf("application/*+xml;version=%s", version)
}

func GetAuthorizationHeader(token string) (key, value string) {
	return "x-vcloud-authorization", token
}

func ExecuteRequest(payload, path, type_, contentType string, client *Client) (Task, error) {
	s, _ := url.ParseRequestURI(path)

	var req *http.Request
	switch type_ {
	case "POST":
		b := bytes.NewBufferString(xml.Header + payload)
		req = client.NewRequest(map[string]string{}, type_, *s, b)
	default:
		req = client.NewRequest(map[string]string{}, type_, *s, nil)
	}

	if contentType != "" {
		req.Header.Add("Content-Type", contentType)
	}

	resp, err := checkResp(client.Http.Do(req))
	if err != nil {
		return Task{}, err
	}
	defer resp.Body.Close()

	task := NewTask(client)
	if err = decodeBody(resp, task.Task); err != nil {
		return Task{}, fmt.Errorf("error decoding Task response: %s", err)
	}

	return *task, nil
}

// Extract ID from an URN.
// 'urn:vcloud:catalog:39867ab4-04e0-4b13-b468-08abcc1de810' will produce '39867ab4-04e0-4b13-b468-08abcc1de810'
func ExtractID(urn string) string {
	parts := strings.Split(urn, ":")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return urn
}
