# Go Sitemap Crawler

A production-oriented sitemap crawler written in Go that simulates how search engines crawl real-world websites using XML sitemaps.

This project is intentionally designed to handle the messy realities of enterprise websites, including malformed XML, sitemap indexes, gzip compression, and CDN throttling.

---

## What is a Sitemap?

A sitemap is a machine-readable file, usually XML, that lists the URLs a website wants search engines to crawl. Large sites often split URLs across multiple sitemaps and reference them through a sitemap index.

Search engines rely on sitemaps to:
- Discover pages efficiently
- Crawl large or complex sites
- Detect updated or newly published content

---

## What This Project Does

This crawler:
- Downloads a sitemap or sitemap index
- Recursively follows child sitemaps
- Extracts page URLs
- Requests each page
- Logs crawl behavior and HTTP responses


## Why This Project Is Useful

Real-world sitemaps are rarely clean:
- XML is often invalid or malformed
- Sitemaps are split across many files
- Files are gzip-compressed
- CDNs throttle or block crawler traffic
- Servers sometimes return HTML instead of XML

This project demonstrates how a **real crawler must behave defensively** and still extract value from imperfect inputs.

---

## Features

- Supports standard XML sitemaps (`<urlset>`)
- Supports sitemap index files (`<sitemapindex>`)
- Recursively crawls child sitemaps
- Handles `.xml.gz` compressed sitemaps
- Sanitizes malformed XML entities
- Reuses HTTP connections for stability
- Rate-limits requests to avoid blocking
- Detailed logging for debugging and analysis
- Graceful failure handling

---

## What This Project Does Not Do (Yet)

These are intentional omissions to keep the core focused:

- robots.txt enforcement
- Concurrent crawling
- HTML parsing (title, meta tags, canonicals)
- Data persistence (CSV, database)
- Crawl depth beyond sitemap URLs

Each of these can be added incrementally.

---

## Usage

Run the crawler with a sitemap URL:

```bash
go run main.go <sitemap_url>
```

### Example

```bash
go run main.go https://developers.google.com/sitemap.xml
```

---

## Sample Output

```text
START SITEMAP: https://developers.google.com/sitemap.xml
Parsed sitemap index with 40 children
Following child sitemap: https://developers.google.com/sitemap_1_of_40.xml
SCRAPED: https://developers.google.com/ Status: 200
SCRAPED: https://developers.google.com/search Status: 200
```

Logs show:

* Request lifecycle
* Sitemap traversal
* Failed or skipped URLs
* HTTP status codes

---

## Supported Sitemap Examples

These are known to work:

* [https://developers.google.com/sitemap.xml](https://developers.google.com/sitemap.xml)
* [https://www.cloudflare.com/sitemap.xml](https://www.cloudflare.com/sitemap.xml)
* [https://www.example.com/sitemap.xml](https://www.example.com/sitemap.xml)

Some sites may throttle or block requests. This is expected behavior.

---

## Project Structure

```
.
├── main.go    # Crawler implementation
├── go.mod     # Go module definition
└── README.md  # Project documentation
```

---

## How This Adds Real Value

In practical terms, this crawler can be used for:

* Technical SEO audits
* Sitemap validation
* Website availability checks
* Crawl readiness assessments
* Identifying blocked or unreachable URLs
* Simulating search engine crawl behavior

In consulting or enterprise contexts, this forms the foundation of:

* SEO diagnostics tooling
* Website quality monitoring
* Automated crawl reporting pipelines

---

