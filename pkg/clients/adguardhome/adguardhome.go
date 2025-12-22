package adguardhome

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/deepspace2/plugnpin/pkg/clients"
	"github.com/deepspace2/plugnpin/pkg/logging"
)

var log = logging.GetLogger()

type Client struct {
	http.Client
	baseURL string
}

var headers map[string]string = map[string]string{
	"accept":        "application/json",
	"content-type":  "application/json",
	"authorization": "",
}

func NewClient(baseURL, username, password string) *Client {
	base64encodedCredentials := base64.StdEncoding.EncodeToString([]byte(fmt.Appendf([]byte{}, "%s:%s", username, password)))
	headers["authorization"] = fmt.Sprintf("Basic %s", base64encodedCredentials)
	return &Client{
		http.Client{},
		fmt.Sprintf("%v/control", baseURL),
	}
}

func (ad *Client) getDnsRewrites() (DnsRewrites, error) {
	dnsRewritesResponseString, _, err := clients.Get(&ad.Client, ad.baseURL+"/rewrite/list", headers)
	if err != nil {
		return nil, err
	}
	var resp []DnsRewrite
	json.Unmarshal([]byte(dnsRewritesResponseString), &resp)

	dnsRewrites := DnsRewrites{}
	for _, rawDnsRewrite := range resp {
		dnsRewrites[DomainName(rawDnsRewrite.Domain)] = IP(rawDnsRewrite.Answer)
	}
	return dnsRewrites, nil
}

func (ad *Client) AddDnsRewrite(domain, ip string) error {
	existingRecords, err := ad.getDnsRewrites()
	if err != nil {
		return err
	}
	d := DomainName(domain)
	_, exists := existingRecords[d]

	if exists {
		return nil
	}

	payload, err := json.Marshal(DnsRewrite{Answer: ip, Domain: domain, Enabled: true})
	if err != nil {
		return err
	}
	payloadString := string(payload)
	_, statusCode, err := clients.Post(&ad.Client, ad.baseURL+"/rewrite/add", headers, &payloadString)
	if err != nil {
		return err
	}

	if statusCode == 401 {
		return errors.New("Unauthorized")
	}

	return nil
}

func (ad *Client) DeleteDnsRewrite(domain, ip string) error {
	payload, err := json.Marshal(DnsRewrite{Answer: ip, Domain: domain, Enabled: true})
	if err != nil {
		return err
	}
	payloadString := string(payload)
	_, statusCode, err := clients.Post(&ad.Client, ad.baseURL+"/rewrite/delete", headers, &payloadString)
	if err != nil {
		return err
	}

	if statusCode == 401 {
		return errors.New("Unauthorized")
	}

	return nil
}
