//go:build unit

package pihole

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// setupTestServer creates a new test server and a client pointing to it.
func setupTestServer(handler http.Handler) (*Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	client := NewClient(server.URL)
	client.Client = *server.Client() // Replace the default client with the test server's client
	return client, server
}

func TestLogin(t *testing.T) {
	t.Run("successful login", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check request
			assert.Equal(t, "/api/auth", r.URL.Path)
			assert.Equal(t, "POST", r.Method)
			var payload map[string]string
			json.NewDecoder(r.Body).Decode(&payload)
			assert.Equal(t, "test-password", payload["password"])

			// Send response
			w.WriteHeader(http.StatusOK)
			// The actual API response is more complex, so we mock the whole thing
			fmt.Fprint(w, `{"session": {"sid": "test-sid", "message": "Login successful"}}`)
		})
		client, server := setupTestServer(handler)
		defer server.Close()

		err := client.Login("test-password")

		assert.NoError(t, err)
		assert.Equal(t, "test-sid", client.sid)
	})

	t.Run("api error on login", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"session": {"sid": "", "message": "Invalid password"}}`)
		})
		client, server := setupTestServer(handler)
		defer server.Close()

		err := client.Login("wrong-password")

		assert.Error(t, err)
		assert.Equal(t, "Invalid password", err.Error())
		assert.Empty(t, client.sid)
	})
}

func TestAddDnsRecord(t *testing.T) {
	t.Run("successful add", func(t *testing.T) {
		// This handler needs to handle two requests in sequence
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/config", r.URL.Path)
			assert.Equal(t, "test-sid", r.Header.Get("X-FTL-SID"))

			if r.Method == http.MethodGet {
				// 1. First, the function gets the existing records.
				w.WriteHeader(http.StatusOK)
				// Respond with one existing record.
				fmt.Fprint(w, `{"config": {"dns": {"hosts": ["1.1.1.1 one.com"]}}}`)
				return
			}

			if r.Method == http.MethodPatch {
				// 2. Second, the function patches the new list.
				var payload updateDnsRecordsPayload
				err := json.NewDecoder(r.Body).Decode(&payload)
				assert.NoError(t, err)

				// Assert that the new payload contains both the old and the new record.
				expectedRecords := []string{"1.1.1.1 one.com", "1.2.3.4 test.com"}
				assert.ElementsMatch(t, expectedRecords, payload.Config.DNS.Hosts)

				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `{"success": true}`)
				return
			}

			// Fail test if an unexpected request is made
			t.Fatalf("Received unexpected request: %s %s", r.Method, r.URL.Path)
		})

		client, server := setupTestServer(handler)
		defer server.Close()

		// Manually set the session ID that the login step would have provided
		client.sid = "test-sid"

		err := client.AddDnsRecord("test.com", "1.2.3.4")
		assert.NoError(t, err)
	})
}

func TestDeleteDnsRecord(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		patchCalled := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/config", r.URL.Path)
			assert.Equal(t, "test-sid", r.Header.Get("X-FTL-SID"))

			if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusOK)
				// Respond with two existing records, one of which we will delete.
				fmt.Fprint(w, `{"config": {"dns": {"hosts": ["1.1.1.1 one.com", "2.2.2.2 two.com"]}}}`)
				return
			}

			if r.Method == http.MethodPatch {
				patchCalled = true
				var payload updateDnsRecordsPayload
				err := json.NewDecoder(r.Body).Decode(&payload)
				assert.NoError(t, err)

				// Assert that the new payload contains only the remaining record.
				expectedRecords := []string{"1.1.1.1 one.com"}
				assert.ElementsMatch(t, expectedRecords, payload.Config.DNS.Hosts)

				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `{"success": true}`)
				return
			}
			t.Fatalf("Received unexpected request: %s %s", r.Method, r.URL.Path)
		})

		client, server := setupTestServer(handler)
		defer server.Close()
		client.sid = "test-sid"

		err := client.DeleteDnsRecord("two.com")
		assert.NoError(t, err)
		assert.True(t, patchCalled, "The PATCH endpoint was not called")
	})

	t.Run("no action if record does not exist", func(t *testing.T) {
		patchCalled := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/config", r.URL.Path)
			assert.Equal(t, http.MethodGet, r.Method)

			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"config": {"dns": {"hosts": ["1.1.1.1 one.com"]}}}`)

			if r.Method == http.MethodPatch {
				patchCalled = true
			}
		})

		client, server := setupTestServer(handler)
		defer server.Close()
		client.sid = "test-sid"

		err := client.DeleteDnsRecord("non-existent.com")
		assert.NoError(t, err)
		assert.False(t, patchCalled, "The PATCH endpoint was called unexpectedly")
	})
}

func TestRawDnsRecordToRecord(t *testing.T) {
	testCases := []struct {
		name               string
		input              string
		expectedDomainName DomainName
		expectedIP         IP
		expectErr          bool
	}{
		{
			name:               "happy path",
			input:              "1.2.3.4 test.com",
			expectedDomainName: "test.com",
			expectedIP:         "1.2.3.4",
			expectErr:          false,
		},
		{
			name:               "malformed record",
			input:              "baddata",
			expectedDomainName: "",
			expectedIP:         "",
			expectErr:          true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			domainName, ip, err := rawDnsRecordToRecord(tc.input)
			assert.Equal(t, tc.expectedDomainName, domainName)
			assert.Equal(t, tc.expectedIP, ip)
			assert.Equal(t, tc.expectErr, err != nil)
		})
	}
}

func TestDnsRecordToRaw(t *testing.T) {
	dom := DomainName("test.com")
	ip := IP("1.2.3.4")
	expected := "1.2.3.4 test.com"
	actual := dnsRecordToRaw(dom, ip)
	assert.Equal(t, expected, actual)
}

func TestAddCNameRecord(t *testing.T) {
	t.Run("successful add", func(t *testing.T) {
		// This handler needs to handle two requests in sequence
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/config", r.URL.Path)
			assert.Equal(t, "test-sid", r.Header.Get("X-FTL-SID"))

			if r.Method == http.MethodGet {
				// 1. First, the function gets the existing cname records.
				w.WriteHeader(http.StatusOK)
				// Respond with one existing cname record.
				fmt.Fprint(w, `{"config": {"dns": {"cnameRecords": ["one.com,one.two.com"]}}}`)
				return
			}

			if r.Method == http.MethodPatch {
				// 2. Second, the function patches the new list.
				var payload updateCNameRecordsPayload
				err := json.NewDecoder(r.Body).Decode(&payload)
				assert.NoError(t, err)

				// Assert that the new payload contains both the old and the new record.
				expectedCNames := []string{"one.com,one.two.com", "test.com,test.two.com"}
				assert.ElementsMatch(t, expectedCNames, payload.Config.DNS.CnameRecords)

				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `{"success": true}`)
				return
			}

			// Fail test if an unexpected request is made
			t.Fatalf("Received unexpected request: %s %s", r.Method, r.URL.Path)
		})

		client, server := setupTestServer(handler)
		defer server.Close()

		// Manually set the session ID that the login step would have provided
		client.sid = "test-sid"

		err := client.AddCNameRecord("test.com", "test.two.com")
		assert.NoError(t, err)
	})
}

func TestDeleteCNameRecord(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		patchCalled := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/config", r.URL.Path)
			assert.Equal(t, "test-sid", r.Header.Get("X-FTL-SID"))

			if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusOK)
				// Respond with two existing cname records, one of which we will delete.
				fmt.Fprint(w, `{"config": {"dns": {"cnameRecords": ["one.com,one.two.com", "two.com,two.two.com"]}}}`)
				return
			}

			if r.Method == http.MethodPatch {
				patchCalled = true
				var payload updateCNameRecordsPayload
				err := json.NewDecoder(r.Body).Decode(&payload)
				assert.NoError(t, err)

				// Assert that the new payload contains only the remaining record.
				expectedCNames := []string{"one.com,one.two.com"}
				assert.ElementsMatch(t, expectedCNames, payload.Config.DNS.CnameRecords)

				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `{"success": true}`)
				return
			}
			t.Fatalf("Received unexpected request: %s %s", r.Method, r.URL.Path)
		})

		client, server := setupTestServer(handler)
		defer server.Close()
		client.sid = "test-sid"

		err := client.DeleteCNameRecord("two.com", "two.two.com")
		assert.NoError(t, err)
		assert.True(t, patchCalled, "The PATCH endpoint was not called")
	})

	t.Run("no action if cname does not exist", func(t *testing.T) {
		patchCalled := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/config", r.URL.Path)
			assert.Equal(t, http.MethodGet, r.Method)

			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"config": {"dns": {"cname": ["one.com,one.two.com"]}}}`)

			if r.Method == http.MethodPatch {
				patchCalled = true
			}
		})

		client, server := setupTestServer(handler)
		defer server.Close()
		client.sid = "test-sid"

		err := client.DeleteCNameRecord("non-existent.com", "non-existent.two.com")
		assert.NoError(t, err)
		assert.False(t, patchCalled, "The PATCH endpoint was called unexpectedly")
	})
}

func TestRawCNameRecordToRecord(t *testing.T) {
	testCases := []struct {
		name               string
		input              string
		expectedDomainName DomainName
		expectedTarget     Target
		expectErr          bool
	}{
		{
			name:               "happy path",
			input:              "test.com,target.com",
			expectedDomainName: "test.com",
			expectedTarget:     "target.com",
			expectErr:          false,
		},
		{
			name:               "malformed record",
			input:              "baddata",
			expectedDomainName: "",
			expectedTarget:     "",
			expectErr:          true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			domainName, target, err := rawCNameRecordToRecord(tc.input)
			assert.Equal(t, tc.expectedDomainName, domainName)
			assert.Equal(t, tc.expectedTarget, target)
			assert.Equal(t, tc.expectErr, err != nil)
		})
	}
}

func TestCNameRecordToRaw(t *testing.T) {
	dom := DomainName("test.com")
	target := Target("target.com")
	expected := "test.com,target.com"
	actual := cNameRecordToRaw(dom, target)
	assert.Equal(t, expected, actual)
}
