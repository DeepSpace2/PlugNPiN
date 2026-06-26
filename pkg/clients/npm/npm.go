package npm

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/deepspace2/plugnpin/pkg/clients/common"
	"github.com/deepspace2/plugnpin/pkg/logging"
	"github.com/deepspace2/plugnpin/pkg/metrics"
)

var log = logging.GetLogger("npm")

type Client struct {
	http.Client
	baseURL         string
	headers         map[string]string
	identity        string
	secret          string
	token           string
	tokenExpireTime time.Time
	mu              sync.Mutex
}

func NewClient(baseURL, identity, secret string) *Client {
	return &Client{
		Client: http.Client{
			Transport: common.NewInstrumentedRoundTripper(metrics.NPM, metrics.ObserveApiRequestDuration),
		},
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
	loginResponseString, statusCode, err := common.Post(&n.Client, n.baseURL+"/tokens", n.headers, &payloadString)
	if err != nil {
		return err
	}

	var resp LoginResponse
	err = json.Unmarshal([]byte(loginResponseString), &resp)
	if err != nil {
		return err
	}

	if statusCode >= 400 || resp.Token == "" {
		var loginError ErrorResponse
		err = json.Unmarshal([]byte(loginResponseString), &loginError)
		if err != nil {
			return err
		}
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

func (n *Client) GetProxyHosts() (map[string]int, error) {
	proxyHostsString, statusCode, err := n.makeRequest(http.MethodGet, n.baseURL+"/nginx/proxy-hosts", nil)
	if err != nil || statusCode >= 400 {
		return nil, err
	}

	var proxyHosts []ProxyHostReply
	existingProxyHostsMap := map[string]int{}
	err = json.Unmarshal([]byte(proxyHostsString), &proxyHosts)
	if err != nil {
		return nil, err
	}

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

func (n *Client) makeRequest(method, url string, payload *string) (string, int, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.hasTokenExpired() {
		if err := n.refreshToken(); err != nil {
			return "", 0, fmt.Errorf("pre-emptive token refresh failed: %v", err)
		}
	}

	doRequest := func() (string, int, error) {
		switch method {
		case http.MethodGet:
			return common.Get(&n.Client, url, n.headers)
		case http.MethodPost:
			return common.Post(&n.Client, url, n.headers, payload)
		case http.MethodDelete:
			return common.Delete(&n.Client, url, n.headers)
		default:
			return "", 0, fmt.Errorf("unsupported http method: %s", method)
		}
	}

	resp, statusCode, err := doRequest()

	var errorResponse ErrorResponse
	_ = json.Unmarshal([]byte(resp), &errorResponse)
	isTokenExpiredError := strings.Contains(errorResponse.Error.Message, "Token has expired")

	if statusCode == http.StatusUnauthorized || (statusCode >= 400 && isTokenExpiredError) {
		log.Info("Received auth-related error, attempting reactive token refresh and retry.")
		if refreshErr := n.refreshToken(); refreshErr != nil {
			return resp, statusCode, fmt.Errorf("request failed with auth error, and subsequent token refresh also failed: %v", refreshErr)
		}
		resp, statusCode, err = doRequest()
	}

	return resp, statusCode, err
}

func (n *Client) getCertificates() (Certificates, error) {
	resp, statusCode, err := n.makeRequest(http.MethodGet, n.baseURL+"/nginx/certificates", nil)
	if err != nil || statusCode >= 400 {
		return nil, err
	}

	var certificates Certificates
	err = json.Unmarshal([]byte(resp), &certificates)
	if err != nil {
		return nil, err
	}
	return certificates, nil
}

func (n *Client) GetCertificateIDByName(name string) (int, error) {
	certificates, err := n.getCertificates()
	if err != nil {
		return 0, err
	}
	for _, certificate := range certificates {
		if certificate.NiceName == name {
			return certificate.ID, nil
		}
	}

	return 0, fmt.Errorf("certificate with name %q does not exist", name)
}

func (n *Client) getAccessLists() (AccessLists, error) {
	resp, statusCode, err := n.makeRequest(http.MethodGet, n.baseURL+"/nginx/access-lists", nil)
	if err != nil || statusCode >= 400 {
		return nil, err
	}

	var accessLists AccessLists
	err = json.Unmarshal([]byte(resp), &accessLists)
	if err != nil {
		return nil, err
	}
	return accessLists, nil
}

func (n *Client) GetAccessListIDByName(name string) (int, error) {
	accessLists, err := n.getAccessLists()
	if err != nil {
		return 0, err
	}
	for _, accessList := range accessLists {
		if accessList.Name == name {
			return accessList.ID, nil
		}
	}

	return 0, fmt.Errorf("access list with name %q does not exist", name)
}

func (n *Client) AddProxyHost(host ProxyHost) (bool, error) {
	existingProxyHosts, err := n.GetProxyHosts()
	if err != nil {
		return false, err
	}
	for _, domainName := range host.DomainNames {
		if _, exists := existingProxyHosts[domainName]; exists {
			return false, nil
		}
	}

	payloadBytes, err := json.Marshal(host)
	if err != nil {
		return false, err
	}

	payloadString := string(payloadBytes)
	resp, statusCode, err := n.makeRequest(http.MethodPost, n.baseURL+"/nginx/proxy-hosts", &payloadString)
	if err != nil {
		return false, err
	}

	if statusCode >= 400 {
		var errorResponse ErrorResponse
		err = json.Unmarshal([]byte(resp), &errorResponse)
		if err != nil {
			return false, err
		}
		return false, errors.New(errorResponse.Error.Message)
	}
	return true, nil
}

func (n *Client) DeleteProxyHosts(domains []string) (bool, error) {
	existingProxyHosts, err := n.GetProxyHosts()
	if err != nil {
		return false, err
	}

	var hostID int
	found := false
	for _, domain := range domains {
		if id, exists := existingProxyHosts[domain]; exists {
			hostID = id
			found = true
			break
		}
	}

	if !found {
		return false, nil
	}

	url := fmt.Sprintf("%v/nginx/proxy-hosts/%v", n.baseURL, hostID)
	resp, statusCode, err := n.makeRequest(http.MethodDelete, url, nil)
	if err != nil {
		return false, err
	}

	if statusCode >= 400 {
		var errorResponse ErrorResponse
		err = json.Unmarshal([]byte(resp), &errorResponse)
		if err != nil {
			return false, err
		}
		return false, errors.New(errorResponse.Error.Message)
	}
	return true, nil
}
