package npm

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/deepspace2/plugnpin/pkg/clients"
	"github.com/deepspace2/plugnpin/pkg/logging"
)

var log = logging.GetLogger()

type Client struct {
	http.Client
	baseURL         string
	headers         map[string]string
	identity        string
	secret          string
	token           string
	tokenExpireTime time.Time
}

func NewClient(baseURL, identity, secret string) *Client {
	return &Client{
		Client:  http.Client{},
		baseURL: fmt.Sprintf("%v/api", baseURL),
		headers: map[string]string{
			"content-type": "application/json",
		},
		identity: identity,
		secret:   secret,
	}
}

func parseTokenExpireTime(timeStr string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, timeStr)
}

func (n *Client) hasTokenExpired() bool {
	now := time.Now().UTC()
	return now.Compare(n.tokenExpireTime) >= 0
}

func (n *Client) Login() error {
	loginPayload := LoginRequest{
		Identity: n.identity,
		Secret:   n.secret,
	}
	payloadBytes, err := json.Marshal(loginPayload)
	if err != nil {
		return err
	}
	payloadString := string(payloadBytes)
	loginResponseString, statusCode, err := clients.Post(&n.Client, n.baseURL+"/tokens", n.headers, &payloadString)
	if err != nil {
		return err
	}

	var resp LoginResponse
	err = json.Unmarshal([]byte(loginResponseString), &resp)
	if statusCode >= 400 || err != nil || resp.Token == "" {
		var loginError ErrorResponse
		json.Unmarshal([]byte(loginResponseString), &loginError)
		return errors.New(loginError.Error.Message)
	}

	tokenExpireTime, err := parseTokenExpireTime(resp.Expires)
	if err != nil {
		return fmt.Errorf("failed to parse token expiry time '%v': %v", resp.Expires, err)
	}
	n.tokenExpireTime = tokenExpireTime

	n.token = resp.Token
	n.headers["authorization"] = "Bearer " + n.token
	return nil
}

func (n *Client) GetIP() string {
	url, _ := url.Parse(n.baseURL)
	return url.Hostname()
}

func (n *Client) getProxyHosts() (map[string]int, error) {
	if n.hasTokenExpired() {
		if err := n.refreshToken(); err != nil {
			return nil, err
		}
	}

	proxyHostsString, statusCode, err := clients.Get(&n.Client, n.baseURL+"/nginx/proxy-hosts", n.headers)
	if err != nil || statusCode >= 400 {
		return nil, err
	}

	var proxyHosts []ProxyHostReply
	existingProxyHostsMap := map[string]int{}
	json.Unmarshal([]byte(proxyHostsString), &proxyHosts)

	for _, host := range proxyHosts {
		for _, domainName := range host.DomainNames {
			existingProxyHostsMap[domainName] = host.ID
		}
	}

	return existingProxyHostsMap, nil
}

func (n *Client) refreshToken() error {
	log.Info("Refreshing Nginx Proxy Manager token")
	return n.Login()
}

func (n *Client) getCertificates() (Certificates, error) {
	if n.hasTokenExpired() {
		if err := n.refreshToken(); err != nil {
			return nil, err
		}
	}

	resp, statusCode, err := clients.Get(&n.Client, n.baseURL+"/nginx/certificates", n.headers)
	if err != nil || statusCode >= 400 {
		return nil, err
	}

	var certificates Certificates
	json.Unmarshal([]byte(resp), &certificates)
	return certificates, nil
}

func (n *Client) GetCertificateIDByName(name string) *int {
	certificates, err := n.getCertificates()
	if err != nil {
		log.Error("Failed to get certificates", "error", err)
		return nil
	}
	for _, certificate := range certificates {
		if certificate.NiceName == name {
			return &certificate.ID
		}
	}

	return nil
}

func (n *Client) AddProxyHost(host ProxyHost) error {
	existingProxyHosts, err := n.getProxyHosts()
	if err != nil {
		return err
	}
	for _, domainName := range host.DomainNames {
		if _, exists := existingProxyHosts[domainName]; exists {
			return nil
		}
	}

	payloadBytes, err := json.Marshal(host)
	if err != nil {
		return err
	}

	payloadString := string(payloadBytes)
	resp, statusCode, err := clients.Post(&n.Client, n.baseURL+"/nginx/proxy-hosts", n.headers, &payloadString)
	if err != nil {
		return err
	}

	if statusCode >= 400 {
		var errorResponse ErrorResponse
		json.Unmarshal([]byte(resp), &errorResponse)
		return errors.New(errorResponse.Error.Message)
	}
	return nil
}

func (n *Client) DeleteProxyHost(domain string) error {
	existingProxyHosts, err := n.getProxyHosts()
	if err != nil {
		return err
	}
	hostID, exists := existingProxyHosts[domain]
	if !exists {
		return nil
	}

	url := fmt.Sprintf("%v/nginx/proxy-hosts/%v", n.baseURL, hostID)
	resp, statusCode, err := clients.Delete(&n.Client, url, n.headers)
	if err != nil {
		return err
	}

	if statusCode >= 400 {
		var errorResponse ErrorResponse
		json.Unmarshal([]byte(resp), &errorResponse)
		return errors.New(errorResponse.Error.Message)
	}
	return nil
}
