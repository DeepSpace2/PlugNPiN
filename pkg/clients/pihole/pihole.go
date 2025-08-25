package pihole

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/deepspace2/plugnpin/pkg/clients"
)

type Client struct {
	http.Client
	baseURL string
	sid     string
}

var headers map[string]string = map[string]string{
	"accept":       "application/json",
	"content-type": "application/json",
}

func NewClient(baseURL string) *Client {
	return &Client{
		http.Client{},
		fmt.Sprintf("%v/api", baseURL),
		"",
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
		return fmt.Errorf(resp.Session.Message)
	}

	p.sid = resp.Session.Sid
	return nil
}

func rawDNSHostRawEntryToEntry(rawDNSHostEntry string) (DomainName, IP, error) {
	splitRawDNSHostEntry := strings.Split(rawDNSHostEntry, " ")
	if len(splitRawDNSHostEntry) == 2 {
		ip := IP(splitRawDNSHostEntry[0])
		domain := DomainName(splitRawDNSHostEntry[1])
		return domain, ip, nil
	} else {
		return "", "", fmt.Errorf("got bad raw dns host entry from pihole: %v", rawDNSHostEntry)
	}
}

func dnsHostEntryToRawEntry(domain DomainName, ip IP) string {
	return fmt.Sprintf("%v %v", ip, domain)
}

func (p *Client) getDNSHosts() (DNSHostEntries, error) {
	if p.sid == "" {
		// TODO:
		log.Fatal("no SID")
	}
	headers["X-FTL-SID"] = p.sid
	configResponseString, _, err := clients.Get(&p.Client, p.baseURL+"/config", headers)
	if err != nil {
		return nil, err
	}
	var resp configResponse
	json.Unmarshal([]byte(configResponseString), &resp)

	dnsHostEntries := DNSHostEntries{}
	for _, rawDNSHostEntry := range resp.Config.DNS.Hosts {
		domain, ip, err := rawDNSHostRawEntryToEntry(rawDNSHostEntry)
		if err != nil {
			return nil, err
		}
		dnsHostEntries[domain] = ip
	}
	return dnsHostEntries, nil
}

func (p *Client) AddDNSHostEntry(domain, ip string) error {
	existingEntries, err := p.getDNSHosts()
	if err != nil {
		return err
	}
	d := DomainName(domain)
	_, exists := existingEntries[d]

	if exists {
		return nil
	}

	existingEntries[d] = IP(ip)

	rawEntriesSlice := []string{}
	for domain, ip := range existingEntries {
		rawEntriesSlice = append(rawEntriesSlice, dnsHostEntryToRawEntry(domain, ip))
	}

	payload := updateDNSHostsEntriesPayload{}
	payload.Config.DNS.Hosts = rawEntriesSlice

	payloadString, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if p.sid == "" {
		// TODO:
		log.Fatal("no SID")
	}
	headers["X-FTL-SID"] = p.sid
	resp, statusCode, err := clients.Patch(&p.Client, p.baseURL+"/config", headers, string(payloadString))
	if err != nil {
		return err
	}
	if statusCode >= 400 {
		return errors.New(resp)
	}

	return nil
}

func (p *Client) DeleteDNSHostEntry(domain, ip string) error {
	log.Printf("STUB: deleting DNS host entry for %s (%s)", domain, ip)
	return nil
}