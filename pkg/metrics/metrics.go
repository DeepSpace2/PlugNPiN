package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	ADGUARD_HOME = "adguard-home"
	NPM          = "nginx-proxy-manager"
	PI_HOLE      = "pi-hole"
)

const (
	ADDED   = "added"
	DELETED = "deleted"
)

const (
	ADD_CNAME_RECORD    = "add_cname_record"
	ADD_DNS_RECORD      = "add_dns_record"
	ADD_DNS_REWRITE     = "add_dns_rewrite"
	ADD_PROXY_HOST      = "add_proxy_host"
	DELETE_CNAME_RECORD = "delete_cname_record"
	DELETE_DNS_RECORD   = "delete_dns_record"
	DELETE_DNS_REWRITE  = "delete_dns_rewrite"
	DELETE_PROXY_HOST   = "delete_proxy_host"
	GET_ACCESS_LIST_ID  = "get_access_list_id"
	GET_CERTIFICATE_ID  = "get_certificate_id"
)

var (
	discoveredContainers = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "plugnpin_discovered_containers",
			Help: "Number of containers with plugnpin labels discovered during the last reconciliation scan",
		},
		[]string{"docker_host"},
	)

	managedEntries = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugnpin_managed_entries_total",
			Help: "Total number of DNS entries and proxy hosts created or deleted per service",
		},
		[]string{"service", "action"},
	)

	servicesApiErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugnpin_api_request_errors_total",
			Help: "Total number of API request errors",
		},
		[]string{"service", "action"},
	)

	handledDockerEvents = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugnpin_handled_docker_events_total",
			Help: "Total number of Docker container events processed by the event listener",
		},
		[]string{"docker_host", "event"},
	)

	scanDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "plugnpin_scan_duration_seconds",
			Help: "Time taken for a full reconciliation scan per Docker host",
		},
		[]string{"docker_host"},
	)

	apiRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "plugnpin_api_request_duration_seconds",
			Help: "Duration of HTTP requests to external APIs",
		},
		[]string{"service", "method", "status_group"},
	)
)

func IncrementAdguardHomeEntriesCreated(n int) {
	managedEntries.WithLabelValues(ADGUARD_HOME, ADDED).Add(float64(n))
}

func IncrementAdguardHomeEntriesDeleted(n int) {
	managedEntries.WithLabelValues(ADGUARD_HOME, DELETED).Add(float64(n))
}

func IncrementPiHoleEntriesCreated(n int) {
	managedEntries.WithLabelValues(PI_HOLE, ADDED).Add(float64(n))
}

func IncrementPiHoleEntriesDeleted(n int) {
	managedEntries.WithLabelValues(PI_HOLE, DELETED).Add(float64(n))
}

func IncrementNpmEntriesCreated() {
	managedEntries.WithLabelValues(NPM, ADDED).Add(float64(1))
}

func IncrementNpmEntriesDeleted() {
	managedEntries.WithLabelValues(NPM, DELETED).Add(float64(1))
}

func SetDiscoveredContainers(dockerHost string, n int) {
	discoveredContainers.WithLabelValues(dockerHost).Set(float64(n))
}

func IncrementHandledDockerEvents(dockerHost, event string) {
	handledDockerEvents.WithLabelValues(dockerHost, event).Inc()
}

func ObserveScanDuration(dockerHost string, durationSeconds float64) {
	scanDuration.WithLabelValues(dockerHost).Observe(durationSeconds)
}

func IncrementAdguardHomeApiRequestErrors(action string) {
	incrementApiRequestErrors(ADGUARD_HOME, action)
}

func IncrementPiHoleApiRequestErrors(action string) {
	incrementApiRequestErrors(PI_HOLE, action)
}

func IncrementNpmApiRequestErrors(action string) {
	incrementApiRequestErrors(NPM, action)
}

func incrementApiRequestErrors(service, action string) {
	servicesApiErrors.WithLabelValues(service, action).Inc()
}

type ObserveFunc func(service, method, statusGroup string, durationSeconds float64)

func ObserveApiRequestDuration(service, method, statusGroup string, durationSeconds float64) {
	apiRequestDuration.WithLabelValues(service, method, statusGroup).Observe(durationSeconds)
}
