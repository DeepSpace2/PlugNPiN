//go:build unit

package adguardhome

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// setupTestServer creates a new test server and a client pointing to it.
func setupTestServer(username, password string, handler http.Handler) (*Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	client := NewClient(server.URL, username, password)
	client.Client = *server.Client()
	return client, server
}

func TestAddDnsRewrite(t *testing.T) {
	t.Run("successful add", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
			assert.Equal(t, expectedAuth, auth)
			assert.Equal(t, "application/json", r.Header.Get("accept"))
			assert.Equal(t, "application/json", r.Header.Get("content-type"))

			if r.URL.Path == "/control/rewrite/list" && r.Method == http.MethodGet {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `[{"domain": "one.com", "answer": "1.1.1.1"}]`)
				return
			}

			if r.URL.Path == "/control/rewrite/add" && r.Method == http.MethodPost {
				var payload DnsRewrite
				err := json.NewDecoder(r.Body).Decode(&payload)
				assert.NoError(t, err)

				assert.Equal(t, "test.com", payload.Domain)
				assert.Equal(t, "1.2.3.4", payload.Answer)
				assert.True(t, payload.Enabled)

				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `{}`)
				return
			}

			t.Fatalf("Received unexpected request: %s %s", r.Method, r.URL.Path)
		})

		client, server := setupTestServer("testuser", "testpass", handler)
		defer server.Close()

		err := client.AddDnsRewrite("test.com", "1.2.3.4")
		assert.NoError(t, err)
	})
}

func TestDeleteDnsRewrite(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		var deleteCalled bool
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
			assert.Equal(t, expectedAuth, auth)

			if r.URL.Path == "/control/rewrite/delete" && r.Method == http.MethodPost {
				deleteCalled = true
				var payload DnsRewrite
				err := json.NewDecoder(r.Body).Decode(&payload)
				assert.NoError(t, err)

				assert.Equal(t, "test.com", payload.Domain)
				assert.Equal(t, "1.2.3.4", payload.Answer)
				assert.True(t, payload.Enabled)

				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `{}`)
				return
			}

			if r.URL.Path == "/control/rewrite/list" && r.Method == http.MethodGet {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `[]`)
				return
			}

			t.Fatalf("Received unexpected request: %s %s", r.Method, r.URL.Path)
		})

		client, server := setupTestServer("testuser", "testpass", handler)
		defer server.Close()

		err := client.DeleteDnsRewrite("test.com", "1.2.3.4")
		assert.NoError(t, err)
		assert.True(t, deleteCalled, "Delete API endpoint was not called")

		existingDnsRewrites, err := client.GetDnsRewrites()
		assert.NoError(t, err)
		assert.Equal(t, 0, len(existingDnsRewrites))
	})
}

func TestWrongCredentials(t *testing.T) {
	t.Run("wrong credentials", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/control/rewrite/delete" && r.Method == http.MethodPost {
				auth := r.Header.Get("Authorization")
				expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
				if auth != expectedAuth {
					w.WriteHeader(http.StatusUnauthorized)
					fmt.Fprint(w, `{}`)
				}
				return
			}

			if r.URL.Path == "/control/rewrite/list" && r.Method == http.MethodGet {
				auth := r.Header.Get("Authorization")
				expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
				if auth != expectedAuth {
					w.WriteHeader(http.StatusUnauthorized)
					fmt.Fprint(w, `{}`)
				}
				return
			}

			t.Fatalf("Received unexpected request: %s %s", r.Method, r.URL.Path)
		})

		client, server := setupTestServer("testuser", "wrongpass", handler)
		defer server.Close()

		err := client.DeleteDnsRewrite("test.com", "1.2.3.4")
		assert.Error(t, err)
	})
}
