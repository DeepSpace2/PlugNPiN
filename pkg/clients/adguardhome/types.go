package adguardhome

type DnsRewrite struct {
	Answer  string `json:"answer"`
	Domain  string `json:"domain"`
	Enabled bool   `json:"enabled"`
}

type AdguardHomeOptions struct {
	TargetDomain string
}

type (
	DnsRewrites map[DomainName]IP
	DomainName  string
	IP          string
	Target      string
)
