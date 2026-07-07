package pihole

type loginResponse struct {
	Session struct {
		Valid    bool   `json:"valid"`
		Totp     bool   `json:"totp"`
		Sid      string `json:"sid"`
		Csrf     string `json:"csrf"`
		Validity int    `json:"validity"`
		Message  string `json:"message"`
	} `json:"session"`
	Took float64 `json:"took"`
}

type configResponse struct {
	Config struct {
		DNS struct {
			Hosts        []string `json:"hosts"`
			CnameRecords []any    `json:"cnameRecords"`
		} `json:"dns"`
	} `json:"config"`
	Took float64 `json:"took"`
}

type ErrorResponse struct {
	Error struct {
		Key     string `json:"key"`
		Message string `json:"message"`
		Hint    string `json:"hint"`
	} `json:"error"`
	Took float64 `json:"took"`
}

type updateDnsRecordsPayload struct {
	Config struct {
		DNS struct {
			Hosts []string `json:"hosts"`
		} `json:"dns"`
	} `json:"config"`
}

type updateCNameRecordsPayload struct {
	Config struct {
		DNS struct {
			CnameRecords []string `json:"cnameRecords"`
		}
	}
}

type PiHoleOptions struct {
	TargetDomain string
}

type (
	CNameRecords map[DomainName]Target
	DnsRecords   map[DomainName]IP
	DomainName   string
	IP           string
	Target       string
)
