package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const graphRoot = "https://graph.microsoft.com"

var ErrBlankSearch = errors.New("search query must not be blank")

type Request struct {
	Method string
	URL    string
	Body   any
}

type Client struct {
	token      string
	httpClient *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		token: token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func ValidatePageSize(limit int) error {
	if limit < 1 || limit > 50 {
		return errors.New("expected an integer from 1 to 50")
	}
	return nil
}

func ValidateChatID(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." {
		return errors.New("expected a chat ID from list-chats")
	}
	return nil
}

func ListChatsRequest(limit int, nextLink string) (Request, error) {
	return collectionRequest(graphRoot+"/v1.0/me/chats", limit, nextLink, nil)
}

func ChatDetailsRequest(chatID string) (Request, error) {
	if err := ValidateChatID(chatID); err != nil {
		return Request{}, err
	}
	return Request{Method: http.MethodGet, URL: graphRoot + "/v1.0/me/chats/" + escapePathSegment(chatID)}, nil
}

func ListMessagesRequest(chatID string, limit int, nextLink string) (Request, error) {
	if err := ValidateChatID(chatID); err != nil {
		return Request{}, err
	}
	base := graphRoot + "/v1.0/me/chats/" + escapePathSegment(chatID) + "/messages"
	return collectionRequest(base, limit, nextLink, map[string]string{"$orderby": "createdDateTime desc"})
}

func SearchRequest(query string, offset, limit int) (Request, error) {
	if strings.TrimSpace(query) == "" {
		return Request{}, ErrBlankSearch
	}
	if offset < 0 {
		return Request{}, errors.New("expected an integer >= 0 for --offset")
	}
	if err := ValidatePageSize(limit); err != nil {
		return Request{}, err
	}
	return Request{
		Method: http.MethodPost,
		URL:    graphRoot + "/beta/search/query",
		Body: map[string]any{
			"requests": []any{map[string]any{
				"entityTypes": []string{"chatMessage"},
				"query":       map[string]string{"queryString": query},
				"from":        offset,
				"size":        limit,
			}},
		},
	}, nil
}

func collectionRequest(base string, limit int, nextLink string, extra map[string]string) (Request, error) {
	if nextLink != "" {
		if err := ValidateNextLink(nextLink); err != nil {
			return Request{}, err
		}
		return Request{Method: http.MethodGet, URL: nextLink}, nil
	}
	if err := ValidatePageSize(limit); err != nil {
		return Request{}, err
	}
	u, err := url.Parse(base)
	if err != nil {
		return Request{}, fmt.Errorf("build Microsoft Graph URL: %w", err)
	}
	query := u.Query()
	query.Set("$top", strconv.Itoa(limit))
	for key, value := range extra {
		query.Set(key, value)
	}
	u.RawQuery = query.Encode()
	return Request{Method: http.MethodGet, URL: u.String()}, nil
}

func ValidateNextLink(link string) error {
	u, err := url.Parse(link)
	if err != nil || u.Scheme != "https" || u.Host != "graph.microsoft.com" ||
		!strings.HasPrefix(u.Path, "/v1.0/") || u.Fragment != "" || u.User != nil {
		return errors.New("--next-link must be an HTTPS Microsoft Graph v1.0 continuation URL")
	}
	return nil
}

func escapePathSegment(value string) string {
	const hex = "0123456789ABCDEF"
	var encoded strings.Builder
	for _, b := range []byte(value) {
		if b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' ||
			b >= '0' && b <= '9' || strings.ContainsRune("-_.~", rune(b)) {
			encoded.WriteByte(b)
			continue
		}
		encoded.WriteByte('%')
		encoded.WriteByte(hex[b>>4])
		encoded.WriteByte(hex[b&0x0f])
	}
	return encoded.String()
}

func (c *Client) Do(ctx context.Context, request Request) (map[string]any, error) {
	var body io.Reader
	if request.Body != nil {
		encoded, err := json.Marshal(request.Body)
		if err != nil {
			return nil, fmt.Errorf("encode Microsoft Graph request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, request.Method, request.URL, body)
	if err != nil {
		return nil, errors.New("invalid Microsoft Graph request")
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if request.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := c.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, context.Canceled
		}
		var netErr net.Error
		if errors.As(err, &netErr) || errors.Is(err, context.DeadlineExceeded) {
			return nil, errors.New("HTTPS request to Microsoft failed; check connectivity and proxy settings")
		}
		return nil, errors.New("HTTPS request to Microsoft failed; check connectivity and proxy settings")
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		hint := ""
		switch response.StatusCode {
		case http.StatusUnauthorized:
			hint = " Replace or remove ACCESS_TOKEN in .env if set, or run login again."
		case http.StatusForbidden:
			hint = " Check delegated Chat.Read consent and tenant policy."
		case http.StatusTooManyRequests:
			hint = " Wait before retrying."
			if retryAfter := response.Header.Get("Retry-After"); isDecimal(retryAfter) {
				hint = " Wait at least " + retryAfter + " seconds before retrying."
			}
		}
		return nil, fmt.Errorf("Microsoft Graph returned HTTP %d.%s", response.StatusCode, hint)
	}
	decoder := json.NewDecoder(response.Body)
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, errors.New("Microsoft Graph returned invalid JSON")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, errors.New("Microsoft Graph returned invalid JSON")
	}
	result, ok := value.(map[string]any)
	if !ok {
		return nil, errors.New("Microsoft Graph returned an unexpected JSON response")
	}
	return result, nil
}

func isDecimal(value string) bool {
	if value == "" {
		return false
	}
	_, err := strconv.Atoi(value)
	return err == nil && value[0] >= '0' && value[0] <= '9'
}
