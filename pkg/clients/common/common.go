package common

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/deepspace2/plugnpin/pkg/metrics"
)

type instrumentedRoundTripper struct {
	service string
	wrapped http.RoundTripper
	observe metrics.ObserveFunc
}

func NewInstrumentedRoundTripper(service string, observe metrics.ObserveFunc) http.RoundTripper {
	if observe == nil {
		observe = func(string, string, string, float64) {}
	}
	return &instrumentedRoundTripper{
		service: service,
		wrapped: http.DefaultTransport,
		observe: observe,
	}
}

func (rt *instrumentedRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := rt.wrapped.RoundTrip(r)
	duration := time.Since(start).Seconds()

	var statusGroup string
	if resp == nil {
		statusGroup = "transport_error"
	} else {
		statusGroup = fmt.Sprintf("%dxx", resp.StatusCode/100)
	}

	rt.observe(rt.service, r.Method, statusGroup, duration)

	return resp, err
}

func setHeaders(req *http.Request, headers map[string]string) {
	for k, v := range headers {
		req.Header.Add(k, v)
	}
}

func Post(client *http.Client, path string, headers map[string]string, data *string) (string, int, error) {
	req, err := http.NewRequest(
		http.MethodPost,
		path,
		strings.NewReader(*data),
	)
	if err != nil {
		return "", 0, err
	}

	setHeaders(req, headers)

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	return string(body), resp.StatusCode, nil
}

func Get(client *http.Client, path string, headers map[string]string) (string, int, error) {
	req, err := http.NewRequest(
		http.MethodGet,
		path,
		nil,
	)
	if err != nil {
		return "", 0, err
	}

	setHeaders(req, headers)

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}
	return string(body), resp.StatusCode, nil
}

func Patch(client *http.Client, path string, headers map[string]string, data string) (string, int, error) {
	req, err := http.NewRequest(
		http.MethodPatch,
		path,
		strings.NewReader(data),
	)
	if err != nil {
		return "", 0, nil
	}

	setHeaders(req, headers)

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}
	return string(body), resp.StatusCode, nil
}

func Delete(client *http.Client, path string, headers map[string]string) (string, int, error) {
	req, err := http.NewRequest(
		http.MethodDelete,
		path,
		nil,
	)
	if err != nil {
		return "", 0, err
	}

	setHeaders(req, headers)

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}
	return string(body), resp.StatusCode, nil
}
