package docker

import "strings"

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

// getLabel returns the value of key from labels. An exact match is preferred,
// otherwise the first case-insensitive match is returned. This lets a container
// using plugnpin.ip work like one using plugNPiN.ip, while the canonical
// spelling still wins when a container defines both.
func getLabel(labels map[string]string, key string) (string, bool) {
	if value, ok := labels[key]; ok {
		return value, true
	}
	for label, value := range labels {
		if strings.EqualFold(label, key) {
			return value, true
		}
	}
	return "", false
}

// hasRequiredLabels reports whether labels contain both labels a container needs
// to be handled by PlugNPiN, regardless of their casing.
func hasRequiredLabels(labels map[string]string) bool {
	_, hasIp := getLabel(labels, IpLabel)
	_, hasUrl := getLabel(labels, UrlLabel)
	return hasIp && hasUrl
}
