package pihole

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

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

func NewClient(baseURL string) *Client {
	return &Client{
		Client: http.Client{
			Transport: common.NewInstrumentedRoundTripper(metrics.PI_HOLE, metrics.ObserveApiRequestDuration),
		},
		baseURL: fmt.Sprintf("%v/api", baseURL),
		sid:     "",
	}
}

func (p *Client) Login(password string) error {
	loginPayload := fmt.Sprintf(`{"password": "%v"}`, password)
	loginResponseString, statusCode, err := common.Post(&p.Client, p.baseURL+"/auth", headers, &loginPayload)
	if err != nil {
		return err
	}
	var resp loginResponse
	json.Unmarshal([]byte(loginResponseString), &resp)

	if statusCode >= 400 || resp.Session.Sid == "" {
		return errors.New(resp.Session.Message)
	}

	p.password = password
	p.sid = resp.Session.Sid
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
		log.Error("Missing Pi-Hole session ID")
		os.Exit(1)
	}
	headers["X-FTL-SID"] = p.sid
	configResponseString, _, err := common.Get(&p.Client, p.baseURL+"/config", headers)
	if err != nil {
		return nil, err
	}
	var resp configResponse
	json.Unmarshal([]byte(configResponseString), &resp)

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
		return numOfAddedDnsRecords, err
	}

	modified := false
	for _, domain := range domains {
		d := DomainName(domain)
		if _, exists := existingRecords[d]; !exists {
			existingRecords[d] = IP(ip)
			numOfAddedDnsRecords += 1
			modified = true
		}
	}

	if !modified {
		return numOfAddedDnsRecords, nil
	}

	rawRecordsSlice := []string{}
	for domain, ip := range existingRecords {
		rawRecordsSlice = append(rawRecordsSlice, dnsRecordToRaw(domain, ip))
	}

	payload := updateDnsRecordsPayload{}
	payload.Config.DNS.Hosts = rawRecordsSlice

	payloadString, err := json.Marshal(payload)
	if err != nil {
		return numOfAddedDnsRecords, err
	}

	if p.sid == "" {
		log.Error("Missing Pi-Hole session ID")
		os.Exit(1)
	}
	headers["X-FTL-SID"] = p.sid
	resp, statusCode, err := common.Patch(&p.Client, p.baseURL+"/config", headers, string(payloadString))
	if err != nil {
		return numOfAddedDnsRecords, err
	}
	if statusCode == 401 {
		p.refreshAuth()
		return p.AddDnsRecords(domains, ip)
	}
	if statusCode >= 400 {
		var errorResponse ErrorResponse
		json.Unmarshal([]byte(resp), &errorResponse)
		return numOfAddedDnsRecords, fmt.Errorf("%v. %v", errorResponse.Error.Message, errorResponse.Error.Hint)
	}

	return numOfAddedDnsRecords, nil
}

func (p *Client) DeleteDnsRecords(domains []string) (numOfDeletedDnsRecords int, err error) {
	existingRecords, err := p.GetDnsRecords()
	if err != nil {
		return numOfDeletedDnsRecords, err
	}

	modified := false
	for _, domain := range domains {
		d := DomainName(domain)
		if _, exists := existingRecords[d]; exists {
			delete(existingRecords, d)
			numOfDeletedDnsRecords += 1
			modified = true
		}
	}

	if !modified {
		return numOfDeletedDnsRecords, nil
	}

	rawRecordsSlice := []string{}
	for domain, ip := range existingRecords {
		rawRecordsSlice = append(rawRecordsSlice, dnsRecordToRaw(domain, ip))
	}

	payload := updateDnsRecordsPayload{}
	payload.Config.DNS.Hosts = rawRecordsSlice

	payloadString, err := json.Marshal(payload)
	if err != nil {
		return numOfDeletedDnsRecords, err
	}

	if p.sid == "" {
		log.Error("Missing Pi-Hole session ID")
		os.Exit(1)
	}
	headers["X-FTL-SID"] = p.sid
	resp, statusCode, err := common.Patch(&p.Client, p.baseURL+"/config", headers, string(payloadString))
	if err != nil {
		return numOfDeletedDnsRecords, err
	}
	if statusCode == 401 {
		p.refreshAuth()
		return p.DeleteDnsRecords(domains)
	}
	if statusCode >= 400 {
		var errorResponse ErrorResponse
		json.Unmarshal([]byte(resp), &errorResponse)
		return numOfDeletedDnsRecords, fmt.Errorf("%v. %v", errorResponse.Error.Message, errorResponse.Error.Hint)
	}

	return numOfDeletedDnsRecords, nil
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
		log.Error("Missing Pi-Hole session ID")
		os.Exit(1)
	}
	headers["X-FTL-SID"] = p.sid
	configResponseString, _, err := common.Get(&p.Client, p.baseURL+"/config", headers)
	if err != nil {
		return nil, err
	}
	var resp configResponse
	json.Unmarshal([]byte(configResponseString), &resp)

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
		return numOfAddedCNameRecords, err
	}

	modified := false
	for _, domain := range domains {
		d := DomainName(domain)
		if _, exists := existingRecords[d]; !exists {
			existingRecords[d] = Target(target)
			numOfAddedCNameRecords += 1
			modified = true
		}
	}

	if !modified {
		return numOfAddedCNameRecords, nil
	}

	rawRecordsSlice := []string{}
	for domain, target := range existingRecords {
		rawRecordsSlice = append(rawRecordsSlice, cNameRecordToRaw(domain, target))
	}

	payload := updateCNameRecordsPayload{}
	payload.Config.DNS.CnameRecords = rawRecordsSlice

	payloadString, err := json.Marshal(payload)
	if err != nil {
		return numOfAddedCNameRecords, err
	}

	if p.sid == "" {
		log.Error("Missing Pi-Hole session ID")
		os.Exit(1)
	}
	headers["X-FTL-SID"] = p.sid
	resp, statusCode, err := common.Patch(&p.Client, p.baseURL+"/config", headers, string(payloadString))
	if err != nil {
		return numOfAddedCNameRecords, err
	}
	if statusCode == 401 {
		p.refreshAuth()
		return p.AddCNameRecords(domains, target)
	}
	if statusCode >= 400 {
		var errorResponse ErrorResponse
		json.Unmarshal([]byte(resp), &errorResponse)
		return numOfAddedCNameRecords, fmt.Errorf("%v. %v", errorResponse.Error.Message, errorResponse.Error.Hint)
	}

	return numOfAddedCNameRecords, nil
}

func (p *Client) DeleteCNameRecords(domains []string) (numOfDeletedCNameRecords int, err error) {
	existingRecords, err := p.getCNameRecords()
	if err != nil {
		return numOfDeletedCNameRecords, err
	}

	modified := false
	for _, domain := range domains {
		d := DomainName(domain)
		if _, exists := existingRecords[d]; exists {
			delete(existingRecords, d)
			numOfDeletedCNameRecords += 1
			modified = true
		}
	}

	if !modified {
		return numOfDeletedCNameRecords, nil
	}

	rawRecordsSlice := []string{}
	for domain, target := range existingRecords {
		rawRecordsSlice = append(rawRecordsSlice, cNameRecordToRaw(domain, target))
	}

	payload := updateCNameRecordsPayload{}
	payload.Config.DNS.CnameRecords = rawRecordsSlice

	payloadString, err := json.Marshal(payload)
	if err != nil {
		return numOfDeletedCNameRecords, err
	}

	if p.sid == "" {
		log.Error("Missing Pi-Hole session ID")
		os.Exit(1)
	}
	headers["X-FTL-SID"] = p.sid
	resp, statusCode, err := common.Patch(&p.Client, p.baseURL+"/config", headers, string(payloadString))
	if err != nil {
		return numOfDeletedCNameRecords, err
	}
	if statusCode == 401 {
		p.refreshAuth()
		return p.DeleteCNameRecords(domains)
	}
	if statusCode >= 400 {
		var errorResponse ErrorResponse
		json.Unmarshal([]byte(resp), &errorResponse)
		return numOfDeletedCNameRecords, fmt.Errorf("%v. %v", errorResponse.Error.Message, errorResponse.Error.Hint)
	}

	return numOfDeletedCNameRecords, nil
}

func (p *Client) refreshAuth() {
	log.Info("Refreshing Pi-Hole authentication")
	p.Login(p.password)
}
