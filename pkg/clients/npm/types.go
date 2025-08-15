package npm

type LoginRequest struct {
	Identity string `json:"identity"`
	Secret   string `json:"secret"`
}

type Token struct {
	Token string `json:"token"`
}

type ProxyHostReply struct {
	ID             int            `json:"id,omitempty"`
	DomainNames    []string       `json:"domain_names"`
	ForwardHost    string         `json:"forward_host"`
	ForwardPort    int            `json:"forward_port"`
	ForwardScheme  string         `json:"forward_scheme"`
	CertificateID  any            `json:"certificate_id"`
	SSLForced      bool           `json:"ssl_forced"`
	HSTSForced     bool           `json:"hsts_forced"`
	HSTSsubdomains bool           `json:"hsts_subdomains"`
	HTTP2Support   bool           `json:"http2_support"`
	BlockExploits  bool           `json:"block_exploits"`
	AccessListID   int            `json:"access_list_id"`
	AdvancedConfig string         `json:"advanced_config"`
	Locations      []Location     `json:"locations"`
	Enabled        int            `json:"enabled"`
	Meta           map[string]any `json:"meta"`
}

type Location struct {
	Path           string `json:"path"`
	ForwardHost    string `json:"forward_host"`
	ForwardPort    int    `json:"forward_port"`
	ForwardScheme  string `json:"forward_scheme"`
	AdvancedConfig string `json:"advanced_config"`
}

type ProxyHost struct {
	DomainNames   []string       `json:"domain_names"`
	ForwardHost   string         `json:"forward_host"`
	ForwardPort   int            `json:"forward_port"`
	ForwardScheme string         `json:"forward_scheme"`
	Locations     []Location     `json:"locations"`
	Meta          map[string]any `json:"meta"`
}
