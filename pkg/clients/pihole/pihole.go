package pihole

import (
	"encoding/json"
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
	loginResponseString, statusCode := clients.Post(&p.Client, p.baseURL+"/auth", headers, &loginPayload)

	var resp loginResponse
	json.Unmarshal([]byte(loginResponseString), &resp)

	if statusCode >= 400 || resp.Session.Sid == "" {
		return fmt.Errorf("ERROR loging in to Pi-Hole: %v", loginResponseString)
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

func (p *Client) getDNSHosts() DNSHostEntries {
	if p.sid == "" {
		// TODO:
		log.Fatal("no SID")
	}
	headers["X-FTL-SID"] = p.sid
	configResponseString, _ := clients.Get(&p.Client, p.baseURL+"/config", headers)

	var resp configResponse
	json.Unmarshal([]byte(configResponseString), &resp)

	dnsHostEntries := DNSHostEntries{}
	for _, rawDNSHostEntry := range resp.Config.DNS.Hosts {
		domain, ip, err := rawDNSHostRawEntryToEntry(rawDNSHostEntry)
		if err != nil {
			log.Fatal(err)
		}
		dnsHostEntries[domain] = ip
	}
	return dnsHostEntries
}

func (p *Client) AddDNSHostEntry(domain, ip string) {
	existingEntries := p.getDNSHosts()
	d := DomainName(domain)
	_, exists := existingEntries[d]

	if exists {
		return
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
		log.Fatal(err)
	}

	if p.sid == "" {
		// TODO:
		log.Fatal("no SID")
	}
	headers["X-FTL-SID"] = p.sid
	resp, _ := clients.Patch(&p.Client, p.baseURL+"/config", headers, string(payloadString))

	var r configResponse
	json.Unmarshal([]byte(resp), &r)
}
