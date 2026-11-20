package graph

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
			if err != nil {
				t.Fatal(err)
			}
			if got.URL != test.want {
				t.Fatalf("URL = %q, want %q", got.URL, test.want)
			}
		})
	}
}

func TestDoUsesBearerTokenAndDoesNotFollowRedirects(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
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
	if err == nil || !strings.Contains(err.Error(), "HTTP 302") {
		t.Fatalf("Do error = %v, want HTTP 302", err)
	}
	if result != nil || requestCount != 1 {
		t.Fatalf("result = %#v, requestCount = %d", result, requestCount)
	}
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
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("Do error = %v, want substring %q", err, test.wantErr)
			}
			if strings.Contains(err.Error(), "private") || strings.Contains(err.Error(), "secret-token") {
				t.Fatalf("error leaked response data or token: %v", err)
			}
		})
	}
}

func TestListMessagesRequest(t *testing.T) {
	request, err := ListMessagesRequest("19:abc@thread.v2", 3, "")
	if err != nil {
		t.Fatal(err)
	}
	if request.Method != http.MethodGet || request.URL != "https://graph.microsoft.com/v1.0/me/chats/19%3Aabc%40thread.v2/messages?%24orderby=createdDateTime+desc&%24top=3" {
		t.Fatalf("unexpected request: %#v", request)
	}
}

func TestNextLinkAndChatIDValidation(t *testing.T) {
	for _, id := range []string{"", " ", ".", ".."} {
		if _, err := ChatDetailsRequest(id); err == nil {
			t.Errorf("ChatDetailsRequest(%q) unexpectedly succeeded", id)
		}
	}
	if err := ValidateNextLink(""); err == nil {
		t.Fatal("empty continuation link unexpectedly succeeded")
	}
}

func TestRejectsUntrustedContinuationURL(t *testing.T) {
	for _, link := range []string{
		"https://graph.microsoft.com@other.example/v1.0/me/chats",
		"https://graph.microsoft.com.evil.example/v1.0/me/chats",
		"https://graph.microsoft.com/v1.0/me/chats#fragment",
		"http://graph.microsoft.com/v1.0/me/chats",
		"https://graph.microsoft.com/beta/me/chats",
	} {
		if err := ValidateNextLink(link); err == nil {
			t.Errorf("ValidateNextLink(%q) unexpectedly succeeded", link)
		}
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
		if _, err := SearchRequest(test.query, test.offset, test.limit); err == nil {
			t.Errorf("SearchRequest(%q, %d, %d) unexpectedly succeeded", test.query, test.offset, test.limit)
		}
	}
}
