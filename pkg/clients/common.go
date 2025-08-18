package clients

import (
	"io"
	"net/http"
	"strings"
)

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
