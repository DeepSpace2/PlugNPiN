package npm

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/deepspace2/plugnpin/pkg/clients"
)

type Client struct {
	http.Client
	baseURL  string
	identity string
	secret   string
	token    string
}

var headers map[string]string = map[string]string{
	"content-type": "application/json",
}

func NewClient(baseURL, identity, secret string) *Client {
	return &Client{
		Client:   http.Client{},
		baseURL:  fmt.Sprintf("%v/api", baseURL),
		identity: identity,
		secret:   secret,
	}
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
	loginResponseString, statusCode, err := clients.Post(&n.Client, n.baseURL+"/tokens", headers, &payloadString)
	if err != nil {
		return err
	}

	var resp Token
	err = json.Unmarshal([]byte(loginResponseString), &resp)
	if statusCode >= 400 || err != nil || resp.Token == "" {
		var loginError ErrorResponse
		json.Unmarshal([]byte(loginResponseString), &loginError)
		return errors.New(loginError.Error.Message)
	}
	n.token = resp.Token
	headers["authorization"] = "Bearer " + n.token
	return nil
}

func (n *Client) getProxyHosts() (map[string]string, error) {
	proxyHostsString, statusCode, err := clients.Get(&n.Client, n.baseURL+"/nginx/proxy-hosts", headers)
	if statusCode == 401 {
		n.refreshToken()
		return n.getProxyHosts()
	} else if statusCode >= 400 {
		return nil, err
	}
	var proxyHosts []ProxyHost
	existingProxyHostsMap := map[string]string{}
	json.Unmarshal([]byte(proxyHostsString), &proxyHosts)

	for _, host := range proxyHosts {
		for _, domainName := range host.DomainNames {
			existingProxyHostsMap[domainName] = host.ForwardHost
		}
	}

	return existingProxyHostsMap, nil
}

func (n *Client) refreshToken() error {
	log.Println("Refreshing Nginx Proxy Manager token")
	return n.Login()
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
	resp, statusCode, err := clients.Post(&n.Client, n.baseURL+"/nginx/proxy-hosts", headers, &payloadString)
	if err != nil {
		return err
	}
	if statusCode == 401 {
		err := n.refreshToken()
		if err != nil {
			return err
		}
		_, _, err = clients.Post(&n.Client, n.baseURL+"/nginx/proxy-hosts", headers, &payloadString)
		if err != nil {
			return err
		}
	} else if statusCode >= 400 {
		var errorResponse ErrorResponse
		json.Unmarshal([]byte(resp), &errorResponse)
		return errors.New(errorResponse.Error.Message)
	}
	return nil
}

func (n *Client) DeleteProxyHost(domain string) error {
	log.Printf("STUB: deleting proxy host for %s", domain)
	return nil
}