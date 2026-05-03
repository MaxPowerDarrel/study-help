---
name: esv-api
description: Use this skill whenever the user wants to fetch, display, search, or play audio of Bible passages from the ESV translation in this Go project — including phrases like "look up a verse", "fetch a passage", "ESV API", "scripture lookup", "Bible search", "verse audio", "study help for John 3:16", or anything that involves api.esv.org. The skill produces idiomatic Go code that calls the ESV REST API (passage text, passage HTML, search, audio) using an ESV_API_KEY env var for auth. Trigger this skill even if the user says "Bible verse" without naming ESV — this project uses the ESV API for all scripture lookup.
---

# ESV API skill

This skill teaches Claude how to write Go code in this `study-help` module that reads from `api.esv.org` (the ESV Bible API). The API has four endpoints; this document covers all of them, the project's Go conventions, and a worked example.

When the user asks for *any* Bible passage, verse, search, or audio feature in this project, write Go that follows the patterns below. Don't introduce a third-party HTTP library — the standard library is enough.

## Overview

`api.esv.org` is a REST API for the ESV (English Standard Version) translation. Four endpoints:

| Endpoint                  | Returns | Use                                       |
|---------------------------|---------|-------------------------------------------|
| `GET /v3/passage/text/`   | JSON    | Plain-text passage (most common)          |
| `GET /v3/passage/html/`   | JSON    | Same passage rendered as HTML             |
| `GET /v3/passage/search/` | JSON    | Full-text search across the Bible         |
| `GET /v3/passage/audio/`  | MP3     | Audio reading of a passage (binary body)  |

Rate limit: **5,000 requests per day per IP and per token** (Crossway's published limit). Plan for 429s in long-running code; surface them clearly.

The base URL is `https://api.esv.org`. All requests require auth.

## Authentication

Every request needs this header:

```
Authorization: Token <ESV_API_KEY>
```

Two things to get right:

1. The literal word is **`Token`**, not `Bearer`. A `Bearer` prefix returns 401.
2. Read the key from `os.Getenv("ESV_API_KEY")`. If it's empty, return a clear error from `NewClient` — don't panic, don't read it lazily on every call.

If the user mentions they don't have a key, point them at https://api.esv.org/account/create-application/ to create one. Otherwise assume it's already in their shell environment.

## Recommended package layout

For this repo, put the client in an internal package so nothing outside the module can import it:

```
study-help/
├── main.go
├── go.mod
└── internal/esv/
    ├── client.go    // Client struct, NewClient, do()
    ├── passage.go   // PassageText, PassageHTML
    ├── search.go    // Search
    └── audio.go     // Audio (returns io.ReadCloser)
```

One file per endpoint family keeps each small. A single `Client` centralizes auth, base URL, and `*http.Client` so endpoint methods stay focused on their request/response shapes.

## Go conventions to follow

These are non-negotiable for code in this repo — they keep the package small, testable, and surprise-free.

- **Client struct.** Holds `httpClient *http.Client`, `apiKey string`, `baseURL string` (default `https://api.esv.org`). Export `BaseURL` as a field (not a method) so tests can swap it for an `httptest.NewServer` URL.
- **Constructor.** `NewClient() (*Client, error)` reads `ESV_API_KEY` once and returns an error if unset. No panics. Default `httpClient` is `&http.Client{Timeout: 30 * time.Second}`.
- **Context first.** Every public method takes `ctx context.Context` as its first parameter and uses `http.NewRequestWithContext`. Callers control timeouts and cancellation.
- **Options structs.** Each endpoint has a sibling `*Options` struct (e.g., `PassageOptions`). Zero values mean "use the ESV API's default" — only emit query params that differ from defaults so URLs stay tidy and easy to debug. Pass `nil` to use all defaults.
- **Typed errors.** Define `APIError` once in `client.go`:

  ```go
  type APIError struct {
      StatusCode int
      Body       string
  }
  func (e *APIError) Error() string {
      return fmt.Sprintf("esv api: status %d: %s", e.StatusCode, e.Body)
  }
  ```

  Return `*APIError` from the shared `do()` helper for any non-2xx response. Callers can then `errors.As(err, &apiErr)` to react to 401/429/etc. Don't wrap in `fmt.Errorf` at every layer.
- **Audio is a stream.** `Audio` returns `io.ReadCloser` directly — don't buffer the whole MP3. The caller `defer rc.Close()`s it.
- **JSON decoding.** Use `json.NewDecoder(resp.Body).Decode(&v)`. Don't `io.ReadAll` then `json.Unmarshal` — wastes memory and obscures errors.
- **Standard library only.** No `resty`, no `go-retryablehttp`, no SDK. `net/http` + `net/url` + `encoding/json` is plenty.

## Endpoints reference

For each endpoint below, only the most common options are shown. For the full parameter list (there are ~25 across the endpoints, including layout knobs like `indent-poetry-lines` and `horizontal-line-length`), read `references/api-params.md`.

### `/v3/passage/text/` — plain text passage

Required: `q` (the reference, e.g. `John 3:16-18`).

Common options (defaults shown — these match the API's defaults, so omit them when the option struct's field equals the default):

| Param                          | Default | Notes                                |
|--------------------------------|---------|--------------------------------------|
| `include-passage-references`   | `true`  | Header line with the reference       |
| `include-verse-numbers`        | `true`  |                                      |
| `include-footnotes`            | `true`  |                                      |
| `include-headings`             | `true`  | Section headings inline              |
| `include-short-copyright`      | `true`  |                                      |
| `indent-poetry`                | `true`  |                                      |

Response shape:

```go
type PassageTextResponse struct {
    Query     string   `json:"query"`
    Canonical string   `json:"canonical"`
    Parsed    [][]int  `json:"parsed"`
    Passages  []string `json:"passages"`
}
```

`Passages` is usually a one-element slice; multiple entries appear if the query matched multiple ranges.

### `/v3/passage/html/` — HTML passage

Same query params as `/v3/passage/text/` plus HTML-specific knobs (`include-css-link`, `wrapping-divs`, etc. — see `references/api-params.md`).

Response shape: same as `PassageTextResponse` but `Passages` contains HTML strings.

### `/v3/passage/search/` — full-text search

Required: `q` (the search query, e.g. `love your enemies`).

Common options:

| Param        | Default | Notes                         |
|--------------|---------|-------------------------------|
| `page-size`  | 20      | Max 100                       |
| `page`       | 1       |                               |

Response shape:

```go
type SearchResponse struct {
    Page          int `json:"page"`
    TotalResults  int `json:"total_results"`
    TotalPages    int `json:"total_pages"`
    Results       []struct {
        Reference string `json:"reference"`
        Content   string `json:"content"`
    } `json:"results"`
}
```

### `/v3/passage/audio/` — MP3 audio

Required: `q`.

Response is a binary `audio/mpeg` body — no JSON wrapper. Stream it back as `io.ReadCloser`. Don't decode it.

## Worked example

Here's the shape Claude should produce when the user says something like *"add a command to fetch a passage"*. Fill in the omitted endpoints (`PassageHTML`, `Search`, `Audio`) using the same patterns.

```go
// internal/esv/client.go
package esv

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "os"
    "time"
)

type Client struct {
    HTTPClient *http.Client
    APIKey     string
    BaseURL    string
}

func NewClient() (*Client, error) {
    key := os.Getenv("ESV_API_KEY")
    if key == "" {
        return nil, fmt.Errorf("esv: ESV_API_KEY not set")
    }
    return &Client{
        HTTPClient: &http.Client{Timeout: 30 * time.Second},
        APIKey:     key,
        BaseURL:    "https://api.esv.org",
    }, nil
}

type APIError struct {
    StatusCode int
    Body       string
}

func (e *APIError) Error() string {
    return fmt.Sprintf("esv api: status %d: %s", e.StatusCode, e.Body)
}

// do issues a GET to path?query and decodes a JSON body into out.
// If out is nil, the response body is returned as a stream instead (used by Audio).
func (c *Client) do(ctx context.Context, path string, q url.Values, out any) (io.ReadCloser, error) {
    u := c.BaseURL + path + "?" + q.Encode()
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Authorization", "Token "+c.APIKey)

    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return nil, err
    }
    if resp.StatusCode/100 != 2 {
        body, _ := io.ReadAll(resp.Body)
        resp.Body.Close()
        return nil, &APIError{StatusCode: resp.StatusCode, Body: string(body)}
    }
    if out == nil {
        return resp.Body, nil // caller closes
    }
    defer resp.Body.Close()
    if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
        return nil, fmt.Errorf("esv: decode: %w", err)
    }
    return nil, nil
}
```

```go
// internal/esv/passage.go
package esv

import (
    "context"
    "net/url"
)

type PassageOptions struct {
    IncludeFootnotes        *bool // pointer so zero-value means "default"
    IncludeHeadings         *bool
    IncludeVerseNumbers     *bool
    IncludePassageReferences *bool
}

type PassageTextResponse struct {
    Query     string   `json:"query"`
    Canonical string   `json:"canonical"`
    Passages  []string `json:"passages"`
}

func (c *Client) PassageText(ctx context.Context, q string, opts *PassageOptions) (*PassageTextResponse, error) {
    v := url.Values{}
    v.Set("q", q)
    applyPassageOptions(v, opts)
    var out PassageTextResponse
    if _, err := c.do(ctx, "/v3/passage/text/", v, &out); err != nil {
        return nil, err
    }
    return &out, nil
}

func applyPassageOptions(v url.Values, o *PassageOptions) {
    if o == nil {
        return
    }
    if o.IncludeFootnotes != nil {
        v.Set("include-footnotes", boolStr(*o.IncludeFootnotes))
    }
    if o.IncludeHeadings != nil {
        v.Set("include-headings", boolStr(*o.IncludeHeadings))
    }
    if o.IncludeVerseNumbers != nil {
        v.Set("include-verse-numbers", boolStr(*o.IncludeVerseNumbers))
    }
    if o.IncludePassageReferences != nil {
        v.Set("include-passage-references", boolStr(*o.IncludePassageReferences))
    }
}

func boolStr(b bool) string {
    if b {
        return "true"
    }
    return "false"
}
```

Caller in `main.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "study-help/internal/esv"
)

func main() {
    if len(os.Args) < 2 {
        log.Fatal("usage: study-help <reference>")
    }
    client, err := esv.NewClient()
    if err != nil {
        log.Fatal(err)
    }
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    res, err := client.PassageText(ctx, os.Args[1], nil)
    if err != nil {
        log.Fatal(err)
    }
    for _, p := range res.Passages {
        fmt.Println(p)
    }
}
```

## Error handling

`do()` returns `*APIError` for any non-2xx. Typical statuses:

- **401** — bad or missing key. Header probably says `Bearer` instead of `Token`, or `ESV_API_KEY` is empty.
- **404** — query couldn't be parsed as a reference (e.g. typo: `Jhn 3:16`). Body is JSON with a `detail` field.
- **429** — rate limit hit (5,000/day). Back off; retry tomorrow or with a different IP/token.

Callers that want to react to specific statuses:

```go
var apiErr *esv.APIError
if errors.As(err, &apiErr) && apiErr.StatusCode == 429 {
    // surface to user, don't retry in a tight loop
}
```

## Testing

Don't hit the real API in tests — it costs quota and is flaky for CI. Spin up an `httptest.NewServer`, set the test handler to return a canned JSON body, and point the client at it:

```go
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    if got := r.Header.Get("Authorization"); got != "Token testkey" {
        t.Fatalf("auth header = %q, want Token testkey", got)
    }
    w.Header().Set("Content-Type", "application/json")
    fmt.Fprint(w, `{"query":"John 3:16","canonical":"John 3:16","passages":["For God so loved..."]}`)
}))
defer srv.Close()

c := &esv.Client{HTTPClient: srv.Client(), APIKey: "testkey", BaseURL: srv.URL}
res, err := c.PassageText(context.Background(), "John 3:16", nil)
```

The two checks worth writing first: that the `Authorization: Token <key>` header is set correctly (a regression here breaks every call), and that an options struct with one field set produces exactly that one query param (defaults shouldn't leak into the URL).

Mock the network at `httptest`, never at the `*http.Client` level — the latter hides URL/header bugs.

## Reference files

- `references/api-params.md` — exhaustive list of query parameters per endpoint. Read this when the user asks for an option that isn't covered above (e.g. `indent-poetry-lines`, `horizontal-line-length`, search's `sort` and `match-not-required-by-default`).
