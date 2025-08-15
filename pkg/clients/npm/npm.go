package npm

import (
	"encoding/json"
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
		log.Fatal(err)
	}
	payloadString := string(payloadBytes)
	loginResponseString, statusCode := clients.Post(&n.Client, n.baseURL+"/tokens", headers, &payloadString)
	var resp Token
	json.Unmarshal([]byte(loginResponseString), &resp)
	if statusCode >= 400 || resp.Token == "" {
		return fmt.Errorf("ERROR loging in to Nginx Proxy Manager: %v", loginResponseString)
	}
	n.token = resp.Token
	headers["authorization"] = "Bearer " + n.token
	return nil
}

func (n *Client) getProxyHosts() (map[string]string, error) {
	proxyHostsString, statusCode := clients.Get(&n.Client, n.baseURL+"/nginx/proxy-hosts", headers)
	if statusCode >= 400 {
		return nil, fmt.Errorf("ERROR getting proxy hosts from nginx proxy manager")
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

func (n *Client) refreshToken() {
	log.Println("Refreshing nginx proxy manager token")
	if err := n.Login(); err != nil {
		log.Fatal(err)
	}
}

func (n *Client) AddProxyHost(host ProxyHost) {
	existingProxyHosts, err := n.getProxyHosts()
	if err != nil {
		n.refreshToken()
		existingProxyHosts, _ = n.getProxyHosts()
	}

	for _, domainName := range host.DomainNames {
		if _, exists := existingProxyHosts[domainName]; exists {
			return
		}
	}

	payloadBytes, err := json.Marshal(host)
	if err != nil {
		log.Fatal(err)
	}
	payloadString := string(payloadBytes)
	_, statusCode := clients.Post(&n.Client, n.baseURL+"/nginx/proxy-hosts", headers, &payloadString)
	if statusCode == 401 {
		n.refreshToken()
		clients.Post(&n.Client, n.baseURL+"/nginx/proxy-hosts", headers, &payloadString)
	}
}
