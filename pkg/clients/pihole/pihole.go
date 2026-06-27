package pihole

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/deepspace2/plugnpin/pkg/clients/common"
	"github.com/deepspace2/plugnpin/pkg/logging"
	"github.com/deepspace2/plugnpin/pkg/metrics"
)

var log = logging.GetLogger("pihole")

type Client struct {
	http.Client
	baseURL  string
	password string
	sid      string
}

var headers map[string]string = map[string]string{
	"accept":       "application/json",
	"content-type": "application/json",
}

func NewClient(baseURL, password string) *Client {
	return &Client{
		Client: http.Client{
			Transport: common.NewInstrumentedRoundTripper(metrics.PI_HOLE, metrics.ObserveApiRequestDuration),
		},
		baseURL:  fmt.Sprintf("%v/api", baseURL),
		password: password,
		sid:      "",
	}
}

func (p *Client) Login() error {
	loginPayload := fmt.Sprintf(`{"password": "%v"}`, p.password)
	loginResponseString, statusCode, err := common.Post(&p.Client, p.baseURL+"/auth", headers, &loginPayload)
	if err != nil {
		return err
	}
	var resp loginResponse
	err = json.Unmarshal([]byte(loginResponseString), &resp)
	if err != nil {
		return err
	}

	if statusCode >= 400 || resp.Session.Sid == "" {
		return errors.New(resp.Session.Message)
	}

	p.sid = resp.Session.Sid
	return nil
}

func (p *Client) Logout() error {
	if p.sid == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	headers["X-FTL-SID"] = p.sid
	_, statusCode, err := common.DeleteWithContext(ctx, &p.Client, p.baseURL+"/auth", headers)
	if err != nil {
		return err
	}
	p.sid = ""
	if statusCode >= 400 {
		log.Warn("Pi-Hole logout returned non-success status", "status", statusCode)
	}
	return nil
}

func rawDnsRecordToRecord(rawDnsRecord string) (DomainName, IP, error) {
	splitRawDnsRecord := strings.Split(rawDnsRecord, " ")
	if len(splitRawDnsRecord) == 2 {
		ip := IP(splitRawDnsRecord[0])
		domain := DomainName(splitRawDnsRecord[1])
		return domain, ip, nil
	} else {
		return "", "", fmt.Errorf("got bad raw dns host entry from pihole: %v", rawDnsRecord)
	}
}

func dnsRecordToRaw(domain DomainName, ip IP) string {
	return fmt.Sprintf("%v %v", ip, domain)
}

func (p *Client) GetDnsRecords() (DnsRecords, error) {
	if p.sid == "" {
		return nil, errMissingSessionId
	}

	headers["X-FTL-SID"] = p.sid
	configResponseString, _, err := common.Get(&p.Client, p.baseURL+"/config", headers)
	if err != nil {
		return nil, err
	}

	var resp configResponse
	err = json.Unmarshal([]byte(configResponseString), &resp)
	if err != nil {
		return nil, err
	}

	dnsRecords := DnsRecords{}
	for _, rawDnsRecords := range resp.Config.DNS.Hosts {
		domain, ip, err := rawDnsRecordToRecord(rawDnsRecords)
		if err != nil {
			return nil, err
		}
		dnsRecords[domain] = ip
	}
	return dnsRecords, nil
}

func (p *Client) AddDnsRecords(domains []string, ip string) (numOfAddedDnsRecords int, err error) {
	existingRecords, err := p.GetDnsRecords()
	if err != nil {
		return 0, err
	}

	addedDomains := []string{}
	for _, domain := range domains {
		d := DomainName(domain)
		if _, exists := existingRecords[d]; !exists {
			existingRecords[d] = IP(ip)
			addedDomains = append(addedDomains, domain)
		}
	}

	if len(addedDomains) == 0 {
		return 0, nil
	}

	rawRecordsSlice := []string{}
	for domain, ip := range existingRecords {
		rawRecordsSlice = append(rawRecordsSlice, dnsRecordToRaw(domain, ip))
	}

	payload := updateDnsRecordsPayload{}
	payload.Config.DNS.Hosts = rawRecordsSlice

	payloadString, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	if p.sid == "" {
		return 0, errMissingSessionId
	}
	headers["X-FTL-SID"] = p.sid
	resp, statusCode, err := common.Patch(&p.Client, p.baseURL+"/config", headers, string(payloadString))
	if err != nil {
		return 0, err
	}

	if statusCode == 401 {
		if err = p.refreshAuth(); err != nil {
			return 0, errors.Join(errAuthRefreshFailed, err)
		}
		return p.AddDnsRecords(domains, ip)
	}

	if statusCode >= 400 {
		var errorResponse ErrorResponse
		err = json.Unmarshal([]byte(resp), &errorResponse)
		if err != nil {
			return 0, err
		}

		return 0, fmt.Errorf("%v. %v", errorResponse.Error.Message, errorResponse.Error.Hint)
	}

	return len(addedDomains), nil
}

func (p *Client) DeleteDnsRecords(domains []string) (numOfDeletedDnsRecords int, err error) {
	existingRecords, err := p.GetDnsRecords()
	if err != nil {
		return 0, err
	}

	deletedDomains := []string{}
	for _, domain := range domains {
		d := DomainName(domain)
		if _, exists := existingRecords[d]; exists {
			delete(existingRecords, d)
			deletedDomains = append(deletedDomains, domain)
		}
	}

	if len(deletedDomains) == 0 {
		return 0, nil
	}

	rawRecordsSlice := []string{}
	for domain, ip := range existingRecords {
		rawRecordsSlice = append(rawRecordsSlice, dnsRecordToRaw(domain, ip))
	}

	payload := updateDnsRecordsPayload{}
	payload.Config.DNS.Hosts = rawRecordsSlice

	payloadString, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	if p.sid == "" {
		return 0, errMissingSessionId
	}
	headers["X-FTL-SID"] = p.sid
	resp, statusCode, err := common.Patch(&p.Client, p.baseURL+"/config", headers, string(payloadString))
	if err != nil {
		return 0, err
	}

	if statusCode == 401 {
		if err = p.refreshAuth(); err != nil {
			return 0, errors.Join(errAuthRefreshFailed, err)
		}
		return p.DeleteDnsRecords(domains)
	}

	if statusCode >= 400 {
		var errorResponse ErrorResponse
		err = json.Unmarshal([]byte(resp), &errorResponse)
		if err != nil {
			return 0, err
		}
		return 0, fmt.Errorf("%v. %v", errorResponse.Error.Message, errorResponse.Error.Hint)
	}

	return len(deletedDomains), nil
}

func rawCNameRecordToRecord(rawCNameRecord string) (DomainName, Target, error) {
	splitRawCNameRecord := strings.Split(rawCNameRecord, ",")
	if len(splitRawCNameRecord) == 2 {
		domain := DomainName(splitRawCNameRecord[0])
		target := Target(splitRawCNameRecord[1])
		return domain, target, nil
	} else {
		return "", "", fmt.Errorf("got bad raw CNAME record from pihole: %v", rawCNameRecord)
	}
}

func cNameRecordToRaw(domain DomainName, target Target) string {
	return fmt.Sprintf("%v,%v", domain, target)
}

func (p *Client) getCNameRecords() (CNameRecords, error) {
	if p.sid == "" {
		return nil, errMissingSessionId
	}

	headers["X-FTL-SID"] = p.sid
	configResponseString, _, err := common.Get(&p.Client, p.baseURL+"/config", headers)
	if err != nil {
		return nil, err
	}

	var resp configResponse
	err = json.Unmarshal([]byte(configResponseString), &resp)
	if err != nil {
		return nil, err
	}

	cNameRecords := CNameRecords{}
	for _, rawCNameRecord := range resp.Config.DNS.CnameRecords {
		domain, target, err := rawCNameRecordToRecord(rawCNameRecord.(string))
		if err != nil {
			return nil, err
		}
		cNameRecords[domain] = target
	}

	return cNameRecords, nil
}

func (p *Client) AddCNameRecords(domains []string, target string) (numOfAddedCNameRecords int, err error) {
	existingRecords, err := p.getCNameRecords()
	if err != nil {
		return 0, err
	}

	addedDomains := []string{}
	for _, domain := range domains {
		d := DomainName(domain)
		if _, exists := existingRecords[d]; !exists {
			existingRecords[d] = Target(target)
			addedDomains = append(addedDomains, domain)
		}
	}

	if len(addedDomains) == 0 {
		return 0, nil
	}

	rawRecordsSlice := []string{}
	for domain, target := range existingRecords {
		rawRecordsSlice = append(rawRecordsSlice, cNameRecordToRaw(domain, target))
	}

	payload := updateCNameRecordsPayload{}
	payload.Config.DNS.CnameRecords = rawRecordsSlice

	payloadString, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	if p.sid == "" {
		return 0, errMissingSessionId
	}

	headers["X-FTL-SID"] = p.sid
	resp, statusCode, err := common.Patch(&p.Client, p.baseURL+"/config", headers, string(payloadString))
	if err != nil {
		return 0, err
	}

	if statusCode == 401 {
		if err = p.refreshAuth(); err != nil {
			return 0, errors.Join(errAuthRefreshFailed, err)
		}
		return p.AddCNameRecords(domains, target)
	}

	if statusCode >= 400 {
		var errorResponse ErrorResponse
		err = json.Unmarshal([]byte(resp), &errorResponse)
		if err != nil {
			return 0, err
		}

		return 0, fmt.Errorf("%v. %v", errorResponse.Error.Message, errorResponse.Error.Hint)
	}

	return len(addedDomains), nil
}

func (p *Client) DeleteCNameRecords(domains []string) (numOfDeletedCNameRecords int, err error) {
	existingRecords, err := p.getCNameRecords()
	if err != nil {
		return 0, err
	}

	deletedDomains := []string{}
	for _, domain := range domains {
		d := DomainName(domain)
		if _, exists := existingRecords[d]; exists {
			delete(existingRecords, d)
			deletedDomains = append(deletedDomains, domain)
		}
	}

	if len(deletedDomains) == 0 {
		return 0, nil
	}

	rawRecordsSlice := []string{}
	for domain, target := range existingRecords {
		rawRecordsSlice = append(rawRecordsSlice, cNameRecordToRaw(domain, target))
	}

	payload := updateCNameRecordsPayload{}
	payload.Config.DNS.CnameRecords = rawRecordsSlice

	payloadString, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	if p.sid == "" {
		return 0, errMissingSessionId
	}

	headers["X-FTL-SID"] = p.sid
	resp, statusCode, err := common.Patch(&p.Client, p.baseURL+"/config", headers, string(payloadString))
	if err != nil {
		return 0, err
	}

	if statusCode == 401 {
		if err = p.refreshAuth(); err != nil {
			return 0, errors.Join(errAuthRefreshFailed, err)
		}
		return p.DeleteCNameRecords(domains)
	}

	if statusCode >= 400 {
		var errorResponse ErrorResponse
		err = json.Unmarshal([]byte(resp), &errorResponse)
		if err != nil {
			return 0, err
		}

		return 0, fmt.Errorf("%v. %v", errorResponse.Error.Message, errorResponse.Error.Hint)
	}

	return len(deletedDomains), nil
}

func (p *Client) refreshAuth() error {
	log.Info("Refreshing Pi-Hole authentication")
	if err := p.Logout(); err != nil {
		log.Warn("Failed to logout old Pi-Hole session", "error", err)
	}
	return p.Login()
}
