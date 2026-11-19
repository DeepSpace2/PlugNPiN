package docker

import (
	"strconv"
	"strings"
)

// Label keys are matched case-insensitively, so plugnpin.ip is treated the same
// as plugNPiN.ip. The constants below are the canonical spellings and are the
// names used in the documentation and in error messages.
const (
	GeneralOptionsCreateOnHealthyLabel = "plugNPiN.options.createOnHealthy"
	IpLabel                            = "plugNPiN.ip"
	UrlLabel                           = "plugNPiN.url"

	adguardHomeOptionsTargetDomainLabel = "plugNPiN.adguardHomeOptions.targetDomain"
	npmOptionsAccessListNameLabel       = "plugNPiN.npmOptions.accessListName"
	npmOptionsAdvancedConfigLabel       = "plugNPiN.npmOptions.advancedConfig"
	npmOptionsBlockExploitsLabel        = "plugNPiN.npmOptions.blockExploits"
	npmOptionsCachingEnabledLabel       = "plugNPiN.npmOptions.cachingEnabled"
	npmOptionsCertificateNameLabel      = "plugNPiN.npmOptions.certificateName"
	npmOptionsHTTP2SupportLabel         = "plugNPiN.npmOptions.http2Support"
	npmOptionsHstsEnabledLabel          = "plugNPiN.npmOptions.hstsEnabled"
	npmOptionsHstsSubdomainsLabel       = "plugNPiN.npmOptions.hstsSubdomains"
	npmOptionsSchemeLabel               = "plugNPiN.npmOptions.scheme"
	npmOptionsSslForcedLabel            = "plugNPiN.npmOptions.forceSsl"
	npmOptionsWebsocketsSupportLabel    = "plugNPiN.npmOptions.websocketsSupport"
	piholeOptionsTargetDomainLabel      = "plugNPiN.piholeOptions.targetDomain"
)

type labelValues map[string]string

// value returns the value of key and whether it was present. An exact match is
// preferred, otherwise the first case-insensitive match is returned. This lets
// a container using plugnpin.ip work like one using plugNPiN.ip, while the
// canonical spelling still wins when a container defines both.
func (l labelValues) value(key string) (string, bool) {
	if value, ok := l[key]; ok {
		return value, true
	}
	for label, value := range l {
		if strings.EqualFold(label, key) {
			return value, true
		}
	}
	return "", false
}

func (l labelValues) valueOrDefault(key, fallback string) string {
	value, ok := l.value(key)
	if !ok {
		return fallback
	}
	return value
}

func (l labelValues) boolOrDefault(key string, fallback bool) bool {
	value, ok := l.value(key)
	if !ok {
		return fallback
	}
	parsed, _ := strconv.ParseBool(value)
	return parsed
}

// hasRequiredLabels reports whether labels contain both labels a container needs
// to be handled by PlugNPiN, regardless of their casing.
func hasRequiredLabels(labels map[string]string) bool {
	values := labelValues(labels)
	_, hasIp := values.value(IpLabel)
	_, hasUrl := values.value(UrlLabel)
	return hasIp && hasUrl
}
