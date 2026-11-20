package graph

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRequests(t *testing.T) {
	tests := []struct {
		name string
		got  func() (Request, error)
		want string
	}{
		{
			name: "chats",
			got:  func() (Request, error) { return ListChatsRequest(20, "") },
			want: "https://graph.microsoft.com/v1.0/me/chats?%24top=20",
		},
		{
			name: "chat details escapes identifier",
			got:  func() (Request, error) { return ChatDetailsRequest("19:abc@thread.v2") },
			want: "https://graph.microsoft.com/v1.0/me/chats/19%3Aabc%40thread.v2",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.got()
			require.NoError(t, err)
			assert.Equal(t, test.want, got.URL)
		})
	}
}

func TestDoUsesBearerTokenAndDoesNotFollowRedirects(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		http.Redirect(w, r, "/redirected", http.StatusFound)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.httpClient = &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	result, err := client.Do(context.Background(), Request{Method: http.MethodGet, URL: server.URL})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 302")
	assert.Nil(t, result)
	assert.Equal(t, 1, requestCount)
}

func TestDoValidatesObjectJSONAndSanitizesHTTPError(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		retryAfter string
		wantErr    string
	}{
		{name: "non object JSON", status: http.StatusOK, body: `[]`, wantErr: "unexpected JSON"},
		{name: "invalid JSON", status: http.StatusOK, body: `{`, wantErr: "invalid JSON"},
		{name: "throttled", status: http.StatusTooManyRequests, body: `{"private":"body"}`, retryAfter: "30", wantErr: "30 seconds"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if test.retryAfter != "" {
					w.Header().Set("Retry-After", test.retryAfter)
				}
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()
			client := NewClient("secret-token")
			client.httpClient = server.Client()
			_, err := client.Do(context.Background(), Request{Method: http.MethodGet, URL: server.URL})
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.wantErr)
			assert.NotContains(t, err.Error(), "private")
			assert.NotContains(t, err.Error(), "secret-token")
		})
	}
}

func TestListMessagesRequest(t *testing.T) {
	request, err := ListMessagesRequest("19:abc@thread.v2", 3, "")
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, request.Method)
	assert.Equal(t, "https://graph.microsoft.com/v1.0/me/chats/19%3Aabc%40thread.v2/messages?%24orderby=createdDateTime+desc&%24top=3", request.URL)
}

func TestNextLinkAndChatIDValidation(t *testing.T) {
	for _, id := range []string{"", " ", ".", ".."} {
		_, err := ChatDetailsRequest(id)
		assert.Error(t, err, "ChatDetailsRequest(%q) should fail", id)
	}
	assert.Error(t, ValidateNextLink(""))
}

func TestRejectsUntrustedContinuationURL(t *testing.T) {
	for _, link := range []string{
		"https://graph.microsoft.com@other.example/v1.0/me/chats",
		"https://graph.microsoft.com.evil.example/v1.0/me/chats",
		"https://graph.microsoft.com/v1.0/me/chats#fragment",
		"http://graph.microsoft.com/v1.0/me/chats",
		"https://graph.microsoft.com/beta/me/chats",
	} {
		assert.Error(t, ValidateNextLink(link), "ValidateNextLink(%q) should fail", link)
	}
}

func TestSearchRequestRejectsInvalidBounds(t *testing.T) {
	for _, test := range []struct {
		query  string
		offset int
		limit  int
	}{
		{query: "  ", limit: 1},
		{query: "hello", offset: -1, limit: 1},
		{query: "hello", limit: 51},
	} {
		_, err := SearchRequest(test.query, test.offset, test.limit)
		assert.Error(t, err, "SearchRequest(%q, %d, %d) should fail", test.query, test.offset, test.limit)
	}
}
