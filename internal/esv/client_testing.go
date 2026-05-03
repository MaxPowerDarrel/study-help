package esv

import "net/http"

// HTTPClientForTest exposes the underlying *http.Client so tests can
// install a custom transport (e.g. to point at an httptest stub).
//
// Not intended for production use.
func HTTPClientForTest(c *Client) *http.Client {
	return c.http
}
