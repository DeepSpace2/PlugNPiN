package pihole

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/deepspace2/plugnpin/pkg/clients"
	"github.com/deepspace2/plugnpin/pkg/logging"
)

var log = logging.GetLogger()

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
		Client:  http.Client{},
		baseURL: fmt.Sprintf("%v/api", baseURL),
		sid:     "",
	}
}

func (p *Client) Login(password string) error {
	loginPayload := fmt.Sprintf(`{"password": "%v"}`, password)
	loginResponseString, statusCode, err := clients.Post(&p.Client, p.baseURL+"/auth", headers, &loginPayload)
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
	configResponseString, _, err := clients.Get(&p.Client, p.baseURL+"/config", headers)
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

func (p *Client) AddDnsRecord(domain, ip string) error {
	existingRecords, err := p.GetDnsRecords()
	if err != nil {
		return err
	}
	d := DomainName(domain)
	_, exists := existingRecords[d]

	if exists {
		return nil
	}

	existingRecords[d] = IP(ip)

	rawRecordsSlice := []string{}
	for domain, ip := range existingRecords {
		rawRecordsSlice = append(rawRecordsSlice, dnsRecordToRaw(domain, ip))
	}

	payload := updateDnsRecordsPayload{}
	payload.Config.DNS.Hosts = rawRecordsSlice

	payloadString, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if p.sid == "" {
		log.Error("Missing Pi-Hole session ID")
		os.Exit(1)
	}
	headers["X-FTL-SID"] = p.sid
	resp, statusCode, err := clients.Patch(&p.Client, p.baseURL+"/config", headers, string(payloadString))
	if err != nil {
		return err
	}
	if statusCode == 401 {
		p.refreshAuth()
		return p.AddDnsRecord(domain, ip)
	}
	if statusCode >= 400 {
		var errorResponse ErrorResponse
		json.Unmarshal([]byte(resp), &errorResponse)
		return fmt.Errorf("%v. %v", errorResponse.Error.Message, errorResponse.Error.Hint)
	}

	return nil
}

func (p *Client) DeleteDnsRecord(domain string) error {
	existingRecords, err := p.GetDnsRecords()
	if err != nil {
		return err
	}
	d := DomainName(domain)
	_, exists := existingRecords[d]

	if !exists {
		return nil
	}

	delete(existingRecords, d)

	rawRecordsSlice := []string{}
	for domain, ip := range existingRecords {
		rawRecordsSlice = append(rawRecordsSlice, dnsRecordToRaw(domain, ip))
	}

	payload := updateDnsRecordsPayload{}
	payload.Config.DNS.Hosts = rawRecordsSlice

	payloadString, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if p.sid == "" {
		log.Error("Missing Pi-Hole session ID")
		os.Exit(1)
	}
	headers["X-FTL-SID"] = p.sid
	resp, statusCode, err := clients.Patch(&p.Client, p.baseURL+"/config", headers, string(payloadString))
	if err != nil {
		return err
	}
	if statusCode == 401 {
		p.refreshAuth()
		return p.DeleteDnsRecord(domain)
	}
	if statusCode >= 400 {
		var errorResponse ErrorResponse
		json.Unmarshal([]byte(resp), &errorResponse)
		return fmt.Errorf("%v. %v", errorResponse.Error.Message, errorResponse.Error.Hint)
	}

	return nil
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
	configResponseString, _, err := clients.Get(&p.Client, p.baseURL+"/config", headers)
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

func (p *Client) AddCNameRecord(domain, target string) error {
	existingRecords, err := p.getCNameRecords()
	if err != nil {
		return err
	}
	d := DomainName(domain)
	_, exists := existingRecords[d]

	if exists {
		return nil
	}

	existingRecords[d] = Target(target)

	rawRecordsSlice := []string{}
	for domain, target := range existingRecords {
		rawRecordsSlice = append(rawRecordsSlice, cNameRecordToRaw(domain, target))
	}

	payload := updateCNameRecordsPayload{}
	payload.Config.DNS.CnameRecords = rawRecordsSlice

	payloadString, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if p.sid == "" {
		log.Error("Missing Pi-Hole session ID")
		os.Exit(1)
	}
	headers["X-FTL-SID"] = p.sid
	resp, statusCode, err := clients.Patch(&p.Client, p.baseURL+"/config", headers, string(payloadString))
	if err != nil {
		return err
	}
	if statusCode == 401 {
		p.refreshAuth()
		return p.AddCNameRecord(domain, target)
	}
	if statusCode >= 400 {
		var errorResponse ErrorResponse
		json.Unmarshal([]byte(resp), &errorResponse)
		return fmt.Errorf("%v. %v", errorResponse.Error.Message, errorResponse.Error.Hint)
	}

	return nil
}

func (p *Client) DeleteCNameRecord(domain, target string) error {
	existingRecords, err := p.getCNameRecords()
	if err != nil {
		return err
	}
	d := DomainName(domain)
	_, exists := existingRecords[d]

	if !exists {
		return nil
	}

	delete(existingRecords, d)

	rawRecordsSlice := []string{}
	for domain, target := range existingRecords {
		rawRecordsSlice = append(rawRecordsSlice, cNameRecordToRaw(domain, target))
	}

	payload := updateCNameRecordsPayload{}
	payload.Config.DNS.CnameRecords = rawRecordsSlice

	payloadString, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if p.sid == "" {
		log.Error("Missing Pi-Hole session ID")
		os.Exit(1)
	}
	headers["X-FTL-SID"] = p.sid
	resp, statusCode, err := clients.Patch(&p.Client, p.baseURL+"/config", headers, string(payloadString))
	if err != nil {
		return err
	}
	if statusCode == 401 {
		p.refreshAuth()
		return p.DeleteCNameRecord(domain, target)
	}
	if statusCode >= 400 {
		var errorResponse ErrorResponse
		json.Unmarshal([]byte(resp), &errorResponse)
		return fmt.Errorf("%v. %v", errorResponse.Error.Message, errorResponse.Error.Hint)
	}

	return nil
}

func (p *Client) refreshAuth() {
	log.Info("Refreshing Pi-Hole authentication")
	p.Login(p.password)
}
