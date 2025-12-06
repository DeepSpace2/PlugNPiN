package npm

type LoginRequest struct {
	Identity string `json:"identity"`
	Secret   string `json:"secret"`
}

type ErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type Token struct {
	Token string `json:"token"`
}

type ProxyHostReply struct {
	AccessListID          int        `json:"access_list_id"`
	AdvancedConfig        string     `json:"advanced_config"`
	AllowWebsocketUpgrade bool       `json:"allow_websocket_upgrade"`
	BlockExploits         bool       `json:"block_exploits"`
	CachingEnabled        bool       `json:"caching_enabled"`
	CertificateID         int        `json:"certificate_id"`
	CreatedOn             string     `json:"created_on"`
	DomainNames           []string   `json:"domain_names"`
	Enabled               bool       `json:"enabled"`
	ForwardHost           string     `json:"forward_host"`
	ForwardPort           int        `json:"forward_port"`
	ForwardScheme         string     `json:"forward_scheme"`
	HTTP2Support          bool       `json:"http2_support"`
	HstsEnabled           bool       `json:"hsts_enabled"`
	HstsSubdomains        bool       `json:"hsts_subdomains"`
	ID                    int        `json:"id"`
	Locations             []Location `json:"locations"`
	Meta                  Meta       `json:"meta"`
	ModifiedOn            string     `json:"modified_on"`
	OwnerUserID           int        `json:"owner_user_id"`
	SslForced             bool       `json:"ssl_forced"`
}

type Meta struct {
	DNSChallenge     bool   `json:"dns_challenge"`
	LetsencryptAgree bool   `json:"letsencrypt_agree"`
	LetsencryptEmail string `json:"letsencrypt_email"`
	NginxErr         any    `json:"nginx_err"`
	NginxOnline      bool   `json:"nginx_online"`
}

type Location struct {
	Path           string `json:"path"`
	ForwardHost    string `json:"forward_host"`
	ForwardPort    int    `json:"forward_port"`
	ForwardScheme  string `json:"forward_scheme"`
	AdvancedConfig string `json:"advanced_config"`
}

type ProxyHost struct {
	AdvancedConfig        string     `json:"advanced_config"`
	AllowWebsocketUpgrade bool       `json:"allow_websocket_upgrade"`
	BlockExploits         bool       `json:"block_exploits"`
	CachingEnabled        bool       `json:"caching_enabled"`
	CertificateID         int        `json:"certificate_id"`
	DomainNames           []string   `json:"domain_names"`
	ForwardHost           string     `json:"forward_host"`
	ForwardPort           int        `json:"forward_port"`
	ForwardScheme         string     `json:"forward_scheme"`
	HTTP2Support          bool       `json:"http2_support"`
	HstsEnabled           bool       `json:"hsts_enabled"`
	HstsSubdomains        bool       `json:"hsts_subdomains"`
	Locations             []Location `json:"locations"`
	Meta                  Meta       `json:"meta"`
	SslForced             bool       `json:"ssl_forced"`
}

type Certificates []struct {
	ID          int      `json:"id"`
	CreatedOn   string   `json:"created_on"`
	ModifiedOn  string   `json:"modified_on"`
	OwnerUserID int      `json:"owner_user_id"`
	Provider    string   `json:"provider"`
	NiceName    string   `json:"nice_name"`
	DomainNames []string `json:"domain_names"`
	ExpiresOn   string   `json:"expires_on"`
	Meta        Meta     `json:"meta"`
}

type NpmProxyHostOptions struct {
	AdvancedConfig        string
	AllowWebsocketUpgrade bool
	BlockExploits         bool
	CachingEnabled        bool
	CertificateName       string
	ForwardScheme         string
	HTTP2Support          bool
	HstsEnabled           bool
	HstsSubdomains        bool
	SslForced             bool
}
