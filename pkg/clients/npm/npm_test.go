//go:build unit

package npm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// setupTestServer creates a new test server and a client pointing to it.
func setupTestServer(handler http.Handler) (*Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	// We must use the NewClient constructor as the fields are unexported.
	client := NewClient(server.URL, "test-user", "test-password")
	// And then we replace the standard http client with the test server's client.
	client.Client = *server.Client()
	return client, server
}

func TestLogin(t *testing.T) {
	t.Run("successful login", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/tokens", r.URL.Path)
			var req LoginRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			assert.NoError(t, err)
			assert.Equal(t, "test-user", req.Identity)
			assert.Equal(t, "test-password", req.Secret)

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(LoginResponse{Token: "test-jwt-token", Expires: time.Now().Add(24 * time.Hour).Format(time.RFC3339Nano)})
		})

		client, server := setupTestServer(handler)
		defer server.Close()

		err := client.Login()
		assert.NoError(t, err)
		assert.Equal(t, "test-jwt-token", client.token)
	})
}

func TestAddProxyHost(t *testing.T) {
	const testToken = "test-jwt-token"

	t.Run("successful add when host does not exist", func(t *testing.T) {
		// This handler needs to handle two requests in sequence
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/nginx/proxy-hosts", r.URL.Path)
			authHeader := r.Header.Get("authorization")
			assert.Equal(t, "Bearer "+testToken, authHeader)

			if r.Method == http.MethodGet {
				// 1. The function first gets existing hosts. We return an empty list.
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`[]`))
				return
			}

			if r.Method == http.MethodPost {
				// 2. The function then creates the new host.
				var receivedHost ProxyHost
				err := json.NewDecoder(r.Body).Decode(&receivedHost)
				assert.NoError(t, err)
				assert.Equal(t, []string{"new-host.com"}, receivedHost.DomainNames)

				w.WriteHeader(http.StatusCreated)
				return
			}
		})

		client, server := setupTestServer(handler)
		client.token = testToken // Pre-authorize client
		client.tokenExpireTime = time.Now().Add(24 * time.Hour)
		client.headers["authorization"] = "Bearer " + testToken
		defer server.Close()

		hostToAdd := ProxyHost{
			DomainNames: []string{"new-host.com"},
		}
		err := client.AddProxyHost(hostToAdd)
		assert.NoError(t, err)
	})

	t.Run("no action when host already exists", func(t *testing.T) {
		// This handler only needs to handle the GET request
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/nginx/proxy-hosts", r.URL.Path)
			assert.Equal(t, http.MethodGet, r.Method)

			existingHosts := []ProxyHostReply{
				{ID: 123, DomainNames: []string{"existing-host.com"}},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(existingHosts)
		})

		client, server := setupTestServer(handler)
		client.token = testToken // Pre-authorize client
		client.tokenExpireTime = time.Now().Add(24 * time.Hour)
		client.headers["authorization"] = "Bearer " + testToken
		defer server.Close()

		// Try to add the same host that already exists.
		hostToAdd := ProxyHost{
			DomainNames: []string{"existing-host.com"},
		}
		err := client.AddProxyHost(hostToAdd)
		// Expect no error, and no POST call would have been made.
		assert.NoError(t, err)
	})
}

func TestDeleteProxyHost(t *testing.T) {
	const testToken = "test-jwt-token"

	t.Run("successful delete when host exists", func(t *testing.T) {
		deleteCalled := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("authorization")
			assert.Equal(t, "Bearer "+testToken, authHeader)

			if r.Method == http.MethodGet {
				assert.Equal(t, "/api/nginx/proxy-hosts", r.URL.Path)
				existingHosts := []ProxyHostReply{
					{ID: 123, DomainNames: []string{"existing-host.com"}},
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(existingHosts)
				return
			}

			if r.Method == http.MethodDelete {
				assert.Equal(t, "/api/nginx/proxy-hosts/123", r.URL.Path)
				w.WriteHeader(http.StatusOK)
				deleteCalled = true
				return
			}
		})

		client, server := setupTestServer(handler)
		client.token = testToken // Pre-authorize client
		client.tokenExpireTime = time.Now().Add(24 * time.Hour)
		client.headers["authorization"] = "Bearer " + testToken
		defer server.Close()

		err := client.DeleteProxyHost("existing-host.com")
		assert.NoError(t, err)
		assert.True(t, deleteCalled, "The DELETE endpoint was not called")
	})

	t.Run("no action when host does not exist", func(t *testing.T) {
		deleteCalled := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/nginx/proxy-hosts", r.URL.Path)
			assert.Equal(t, http.MethodGet, r.Method)

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))

			if r.Method == http.MethodDelete {
				deleteCalled = true
			}
		})

		client, server := setupTestServer(handler)
		client.token = testToken // Pre-authorize client
		client.tokenExpireTime = time.Now().Add(24 * time.Hour)
		defer server.Close()

		err := client.DeleteProxyHost("non-existing-host.com")
		assert.NoError(t, err)
		assert.False(t, deleteCalled, "The DELETE endpoint was called unexpectedly")
	})
}
