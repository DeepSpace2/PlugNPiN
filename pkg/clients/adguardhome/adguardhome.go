package adguardhome

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/deepspace2/plugnpin/pkg/clients/common"
	"github.com/deepspace2/plugnpin/pkg/metrics"
)

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
		Client: http.Client{
			Transport: common.NewInstrumentedRoundTripper(metrics.ADGUARD_HOME, metrics.ObserveApiRequestDuration),
		},
		baseURL: fmt.Sprintf("%v/control", baseURL),
	}
}

func (ad *Client) GetDnsRewrites() (DnsRewrites, error) {
	dnsRewritesResponseString, _, err := common.Get(&ad.Client, ad.baseURL+"/rewrite/list", headers)
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

func (ad *Client) AddDnsRewrites(domains []string, ip string) (numOfAddedRewrites int, err error) {
	existingRecords, err := ad.GetDnsRewrites()
	if err != nil {
		return 0, err
	}

	for _, domain := range domains {
		d := DomainName(domain)
		_, exists := existingRecords[d]

		if exists {
			continue
		}

		payload, err := json.Marshal(DnsRewrite{Answer: ip, Domain: domain, Enabled: true})
		if err != nil {
			return 0, err
		}
		payloadString := string(payload)
		_, statusCode, err := common.Post(&ad.Client, ad.baseURL+"/rewrite/add", headers, &payloadString)
		if err != nil {
			return 0, err
		}

		if statusCode == 401 {
			return 0, errors.New("Unauthorized")
		}

		if statusCode < 400 {
			numOfAddedRewrites += 1
		}
	}

	return numOfAddedRewrites, nil
}

func (ad *Client) DeleteDnsRewrites(domains []string, ip string) (numOfDeletedRewrites int, err error) {
	for _, domain := range domains {
		payload, err := json.Marshal(DnsRewrite{Answer: ip, Domain: domain, Enabled: true})
		if err != nil {
			return 0, err
		}
		payloadString := string(payload)
		_, statusCode, err := common.Post(&ad.Client, ad.baseURL+"/rewrite/delete", headers, &payloadString)
		if err != nil {
			return 0, err
		}

		if statusCode == 401 {
			return 0, errors.New("Unauthorized")
		}

		if statusCode < 400 {
			numOfDeletedRewrites += 1
		}
	}

	return numOfDeletedRewrites, nil
}
