// fetch.go — lightweight HTTP fetcher that converts HTML pages to clean text.
// Used by the web_fetch tool for fast reads without launching a browser.
package browser

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// FetchResult contains the result of a lightweight HTTP fetch.
type FetchResult struct {
	URL        string
	Title      string
	StatusCode int
	Text       string // Clean readable text extracted from HTML
	Markdown   string // Best-effort markdown conversion (headings, links, lists)
}

// FetchPage fetches a URL using a plain HTTP request and converts the HTML
// to clean readable text. Faster than browser_navigate — no JS execution,
// no Chrome needed. Best for documentation pages, articles, APIs.
func FetchPage(rawURL string, timeoutSecs int) (*FetchResult, error) {
	if timeoutSecs <= 0 {
		timeoutSecs = 30
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSecs)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	client := &http.Client{
		Timeout: time.Duration(timeoutSecs) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	// Read with a size cap to avoid huge pages blowing memory
	const maxBytes = 2 << 20 // 2 MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")

	// Plain text / JSON / markdown — return as-is
	if !strings.Contains(contentType, "html") {
		text := string(body)
		if len(text) > 8000 {
			text = text[:8000] + "\n[... truncated ...]"
		}
		return &FetchResult{
			URL:        resp.Request.URL.String(),
			StatusCode: resp.StatusCode,
			Text:       text,
			Markdown:   text,
		}, nil
	}

	// Parse HTML with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	title := strings.TrimSpace(doc.Find("title").First().Text())

	// Remove noise elements before extraction
	doc.Find("script, style, noscript, nav, footer, aside, header, .cookie-banner, #cookie-notice, .ads, .advertisement").Remove()

	text := htmlToText(doc)
	markdown := htmlToMarkdown(doc)

	// Trim to LLM-friendly size
	if len(text) > 8000 {
		text = text[:8000] + "\n[... content truncated ...]"
	}
	if len(markdown) > 10000 {
		markdown = markdown[:10000] + "\n[... content truncated ...]"
	}

	return &FetchResult{
		URL:        resp.Request.URL.String(),
		Title:      title,
		StatusCode: resp.StatusCode,
		Text:       text,
		Markdown:   markdown,
	}, nil
}

// htmlToText extracts all text from the document, one line per block element.
func htmlToText(doc *goquery.Document) string {
	var sb strings.Builder
	var walk func(*goquery.Selection)
	walk = func(sel *goquery.Selection) {
		sel.Contents().Each(func(_ int, s *goquery.Selection) {
			node := s.Get(0)
			if node == nil {
				return
			}
			// Text node
			if node.Type == 3 { // html.TextNode
				t := strings.TrimSpace(node.Data)
				if t != "" {
					sb.WriteString(t)
					sb.WriteString(" ")
				}
				return
			}
			tag := strings.ToLower(node.Data)
			// Block elements: flush a newline before recursing
			switch tag {
			case "p", "div", "section", "article", "li", "tr",
				"h1", "h2", "h3", "h4", "h5", "h6",
				"blockquote", "pre", "br", "hr":
				sb.WriteString("\n")
			}
			walk(s)
			switch tag {
			case "p", "div", "section", "article", "li", "tr",
				"h1", "h2", "h3", "h4", "h5", "h6",
				"blockquote", "pre":
				sb.WriteString("\n")
			}
		})
	}
	walk(doc.Selection)

	// Collapse excessive blank lines
	lines := strings.Split(sb.String(), "\n")
	var out []string
	blank := 0
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			blank++
			if blank <= 1 {
				out = append(out, "")
			}
		} else {
			blank = 0
			out = append(out, l)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// htmlToMarkdown does a best-effort HTML → Markdown conversion.
// Handles headings, bold, italic, links, lists, code, blockquote.
func htmlToMarkdown(doc *goquery.Document) string {
	var sb strings.Builder
	var walk func(*goquery.Selection)
	walk = func(sel *goquery.Selection) {
		sel.Contents().Each(func(_ int, s *goquery.Selection) {
			node := s.Get(0)
			if node == nil {
				return
			}
			if node.Type == 3 { // text node
				t := strings.TrimSpace(node.Data)
				if t != "" {
					sb.WriteString(t)
					sb.WriteString(" ")
				}
				return
			}
			tag := strings.ToLower(node.Data)
			switch tag {
			case "h1":
				sb.WriteString("\n# ")
				walk(s)
				sb.WriteString("\n")
			case "h2":
				sb.WriteString("\n## ")
				walk(s)
				sb.WriteString("\n")
			case "h3":
				sb.WriteString("\n### ")
				walk(s)
				sb.WriteString("\n")
			case "h4", "h5", "h6":
				sb.WriteString("\n#### ")
				walk(s)
				sb.WriteString("\n")
			case "p":
				sb.WriteString("\n")
				walk(s)
				sb.WriteString("\n")
			case "br":
				sb.WriteString("\n")
			case "hr":
				sb.WriteString("\n---\n")
			case "strong", "b":
				sb.WriteString("**")
				walk(s)
				sb.WriteString("**")
			case "em", "i":
				sb.WriteString("_")
				walk(s)
				sb.WriteString("_")
			case "code":
				sb.WriteString("`")
				walk(s)
				sb.WriteString("`")
			case "pre":
				sb.WriteString("\n```\n")
				walk(s)
				sb.WriteString("\n```\n")
			case "blockquote":
				sb.WriteString("\n> ")
				walk(s)
				sb.WriteString("\n")
			case "a":
				href, exists := s.Attr("href")
				if exists && href != "" && !strings.HasPrefix(href, "#") && !strings.HasPrefix(href, "javascript") {
					sb.WriteString("[")
					walk(s)
					sb.WriteString(fmt.Sprintf("](%s)", href))
				} else {
					walk(s)
				}
			case "li":
				sb.WriteString("\n- ")
				walk(s)
			case "ul", "ol":
				sb.WriteString("\n")
				walk(s)
				sb.WriteString("\n")
			case "img":
				alt, _ := s.Attr("alt")
				if alt != "" {
					sb.WriteString(fmt.Sprintf("![%s]", alt))
				}
			case "table":
				sb.WriteString("\n")
				walk(s)
				sb.WriteString("\n")
			case "tr":
				walk(s)
				sb.WriteString("\n")
			case "td", "th":
				sb.WriteString("| ")
				walk(s)
				sb.WriteString(" ")
			default:
				walk(s)
			}
		})
	}
	walk(doc.Selection)

	// Collapse excessive blank lines
	lines := strings.Split(sb.String(), "\n")
	var out []string
	blank := 0
	for _, l := range lines {
		trimmed := strings.TrimRight(l, " ")
		if trimmed == "" {
			blank++
			if blank <= 1 {
				out = append(out, "")
			}
		} else {
			blank = 0
			out = append(out, trimmed)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
