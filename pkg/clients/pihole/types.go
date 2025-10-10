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
			Upstreams           []string `json:"upstreams"`
			CNAMEdeepInspect    bool     `json:"CNAMEdeepInspect"`
			BlockESNI           bool     `json:"blockESNI"`
			EDNS0ECS            bool     `json:"EDNS0ECS"`
			IgnoreLocalhost     bool     `json:"ignoreLocalhost"`
			ShowDNSSEC          bool     `json:"showDNSSEC"`
			AnalyzeOnlyAandAAAA bool     `json:"analyzeOnlyAandAAAA"`
			PiholePTR           string   `json:"piholePTR"`
			ReplyWhenBusy       string   `json:"replyWhenBusy"`
			BlockTTL            int      `json:"blockTTL"`
			Hosts               []string `json:"hosts"`
			DomainNeeded        bool     `json:"domainNeeded"`
			ExpandHosts         bool     `json:"expandHosts"`
			Domain              string   `json:"domain"`
			BogusPriv           bool     `json:"bogusPriv"`
			Dnssec              bool     `json:"dnssec"`
			Interface           string   `json:"interface"`
			HostRecord          string   `json:"hostRecord"`
			ListeningMode       string   `json:"listeningMode"`
			QueryLogging        bool     `json:"queryLogging"`
			CnameRecords        []any    `json:"cnameRecords"`
			Port                int      `json:"port"`
			RevServers          []any    `json:"revServers"`
			Cache               struct {
				Size               int `json:"size"`
				Optimizer          int `json:"optimizer"`
				UpstreamBlockedTTL int `json:"upstreamBlockedTTL"`
			} `json:"cache"`
			Blocking struct {
				Active bool   `json:"active"`
				Mode   string `json:"mode"`
				Edns   string `json:"edns"`
			} `json:"blocking"`
			SpecialDomains struct {
				MozillaCanary      bool `json:"mozillaCanary"`
				ICloudPrivateRelay bool `json:"iCloudPrivateRelay"`
				DesignatedResolver bool `json:"designatedResolver"`
			} `json:"specialDomains"`
			Reply struct {
				Host struct {
					Force4 bool   `json:"force4"`
					IPv4   string `json:"IPv4"`
					Force6 bool   `json:"force6"`
					IPv6   string `json:"IPv6"`
				} `json:"host"`
				Blocking struct {
					Force4 bool   `json:"force4"`
					IPv4   string `json:"IPv4"`
					Force6 bool   `json:"force6"`
					IPv6   string `json:"IPv6"`
				} `json:"blocking"`
			} `json:"reply"`
			RateLimit struct {
				Count    int `json:"count"`
				Interval int `json:"interval"`
			} `json:"rateLimit"`
		} `json:"dns"`
		Dhcp struct {
			Active               bool   `json:"active"`
			Start                string `json:"start"`
			End                  string `json:"end"`
			Router               string `json:"router"`
			Netmask              string `json:"netmask"`
			LeaseTime            string `json:"leaseTime"`
			Ipv6                 bool   `json:"ipv6"`
			RapidCommit          bool   `json:"rapidCommit"`
			MultiDNS             bool   `json:"multiDNS"`
			Logging              bool   `json:"logging"`
			IgnoreUnknownClients bool   `json:"ignoreUnknownClients"`
			Hosts                []any  `json:"hosts"`
		} `json:"dhcp"`
		Ntp struct {
			Ipv4 struct {
				Active  bool   `json:"active"`
				Address string `json:"address"`
			} `json:"ipv4"`
			Ipv6 struct {
				Active  bool   `json:"active"`
				Address string `json:"address"`
			} `json:"ipv6"`
			Sync struct {
				Active   bool   `json:"active"`
				Server   string `json:"server"`
				Interval int    `json:"interval"`
				Count    int    `json:"count"`
				Rtc      struct {
					Set    bool   `json:"set"`
					Device string `json:"device"`
					Utc    bool   `json:"utc"`
				} `json:"rtc"`
			} `json:"sync"`
		} `json:"ntp"`
		Resolver struct {
			ResolveIPv4  bool   `json:"resolveIPv4"`
			ResolveIPv6  bool   `json:"resolveIPv6"`
			NetworkNames bool   `json:"networkNames"`
			RefreshNames string `json:"refreshNames"`
		} `json:"resolver"`
		Database struct {
			DBimport   bool `json:"DBimport"`
			MaxDBdays  int  `json:"maxDBdays"`
			DBinterval int  `json:"DBinterval"`
			UseWAL     bool `json:"useWAL"`
			Network    struct {
				ParseARPcache bool `json:"parseARPcache"`
				Expire        int  `json:"expire"`
			} `json:"network"`
		} `json:"database"`
		Webserver struct {
			Domain   string   `json:"domain"`
			Acl      string   `json:"acl"`
			Port     string   `json:"port"`
			Threads  int      `json:"threads"`
			Headers  []string `json:"headers"`
			ServeAll bool     `json:"serve_all"`
			Session  struct {
				Timeout int  `json:"timeout"`
				Restore bool `json:"restore"`
			} `json:"session"`
			TLS struct {
				Cert string `json:"cert"`
			} `json:"tls"`
			Paths struct {
				Webroot string `json:"webroot"`
				Webhome string `json:"webhome"`
				Prefix  string `json:"prefix"`
			} `json:"paths"`
			Interface struct {
				Boxed bool   `json:"boxed"`
				Theme string `json:"theme"`
			} `json:"interface"`
			API struct {
				MaxSessions            int    `json:"max_sessions"`
				PrettyJSON             bool   `json:"prettyJSON"`
				Pwhash                 string `json:"pwhash"`
				Password               string `json:"password"`
				TotpSecret             string `json:"totp_secret"`
				AppPwhash              string `json:"app_pwhash"`
				AppSudo                bool   `json:"app_sudo"`
				CliPw                  bool   `json:"cli_pw"`
				ExcludeClients         []any  `json:"excludeClients"`
				ExcludeDomains         []any  `json:"excludeDomains"`
				MaxHistory             int    `json:"maxHistory"`
				MaxClients             int    `json:"maxClients"`
				ClientHistoryGlobalMax bool   `json:"client_history_global_max"`
				AllowDestructive       bool   `json:"allow_destructive"`
				Temp                   struct {
					Limit int    `json:"limit"`
					Unit  string `json:"unit"`
				} `json:"temp"`
			} `json:"api"`
		} `json:"webserver"`
		Files struct {
			Pid        string `json:"pid"`
			Database   string `json:"database"`
			Gravity    string `json:"gravity"`
			GravityTmp string `json:"gravity_tmp"`
			Macvendor  string `json:"macvendor"`
			Pcap       string `json:"pcap"`
			Log        struct {
				Ftl       string `json:"ftl"`
				Dnsmasq   string `json:"dnsmasq"`
				Webserver string `json:"webserver"`
			} `json:"log"`
		} `json:"files"`
		Misc struct {
			Privacylevel int   `json:"privacylevel"`
			DelayStartup int   `json:"delay_startup"`
			Nice         int   `json:"nice"`
			Addr2Line    bool  `json:"addr2line"`
			EtcDnsmasqD  bool  `json:"etc_dnsmasq_d"`
			DnsmasqLines []any `json:"dnsmasq_lines"`
			ExtraLogging bool  `json:"extraLogging"`
			ReadOnly     bool  `json:"readOnly"`
			Check        struct {
				Load  bool `json:"load"`
				Shmem int  `json:"shmem"`
				Disk  int  `json:"disk"`
			} `json:"check"`
		} `json:"misc"`
		Debug struct {
			Database     bool `json:"database"`
			Networking   bool `json:"networking"`
			Locks        bool `json:"locks"`
			Queries      bool `json:"queries"`
			Flags        bool `json:"flags"`
			Shmem        bool `json:"shmem"`
			Gc           bool `json:"gc"`
			Arp          bool `json:"arp"`
			Regex        bool `json:"regex"`
			API          bool `json:"api"`
			TLS          bool `json:"tls"`
			Overtime     bool `json:"overtime"`
			Status       bool `json:"status"`
			Caps         bool `json:"caps"`
			Dnssec       bool `json:"dnssec"`
			Vectors      bool `json:"vectors"`
			Resolver     bool `json:"resolver"`
			Edns0        bool `json:"edns0"`
			Clients      bool `json:"clients"`
			Aliasclients bool `json:"aliasclients"`
			Events       bool `json:"events"`
			Helper       bool `json:"helper"`
			Config       bool `json:"config"`
			Inotify      bool `json:"inotify"`
			Webserver    bool `json:"webserver"`
			Extra        bool `json:"extra"`
			Reserved     bool `json:"reserved"`
			Ntp          bool `json:"ntp"`
			Netlink      bool `json:"netlink"`
			All          bool `json:"all"`
		} `json:"debug"`
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
