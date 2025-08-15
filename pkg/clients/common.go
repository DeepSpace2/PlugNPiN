package clients

import (
	"io"
	"log"
	"net/http"
	"strings"
)

func setHeaders(req *http.Request, headers map[string]string) {
	for k, v := range headers {
		req.Header.Add(k, v)
	}
}

func Post(client *http.Client, path string, headers map[string]string, data *string) (string, int) {
	req, err := http.NewRequest(
		http.MethodPost,
		path,
		strings.NewReader(*data),
	)
	if err != nil {
		log.Fatal(err)
	}

	setHeaders(req, headers)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return string(body), resp.StatusCode
}

func Get(client *http.Client, path string, headers map[string]string) (string, int) {
	req, err := http.NewRequest(
		http.MethodGet,
		path,
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}

	setHeaders(req, headers)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode >= 400 {
		log.Fatal(resp.StatusCode)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return string(body), resp.StatusCode
}

func Patch(client *http.Client, path string, headers map[string]string, data string) (string, int) {
	req, err := http.NewRequest(
		http.MethodPatch,
		path,
		strings.NewReader(data),
	)
	if err != nil {
		log.Fatal(err)
	}

	setHeaders(req, headers)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode >= 400 {
		log.Fatal(resp.StatusCode)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return string(body), resp.StatusCode
}
