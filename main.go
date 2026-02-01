package main

import (
	"bytes"
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

type CrawlResult struct {
	URL             string
	Status          int
	Title           string
	MetaDescription string
	Canonical       string
}

type urlSet struct {
	URLs []struct {
		Loc string `xml:"loc"`
	} `xml:"url"`
}

type siteMapIndex struct {
	Sitemaps []struct {
		Loc string `xml:"loc"`
	} `xml:"sitemap"`
}

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
	"Mozilla/5.0 (X11; Linux x86_64)",
}

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

func randomUserAgent() string {
	rand.Seed(time.Now().UnixNano())
	return userAgents[rand.Intn(len(userAgents))]
}

func makeRequest(url string) ([]byte, string, int, error) {
	log.Println("REQUEST:", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println("REQUEST ERROR:", err)
		return nil, "", 0, err
	}

	req.Header.Set("User-Agent", randomUserAgent())
	req.Header.Set("Accept", "application/xml,text/xml,text/html,*/*")

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Println("HTTP ERROR:", url, err)
		return nil, "", 0, err
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	log.Println("RESPONSE:", url, "STATUS:", resp.StatusCode, "CT:", ct)

	var reader io.Reader = resp.Body
	if strings.HasSuffix(url, ".gz") {
		log.Println("GZIP DETECTED:", url)
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			log.Println("GZIP ERROR:", err)
			return nil, ct, resp.StatusCode, err
		}
		defer gz.Close()
		reader = gz
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		log.Println("READ ERROR:", err)
		return nil, ct, resp.StatusCode, err
	}

	return body, ct, resp.StatusCode, nil
}

func sanitizeXML(b []byte) []byte {
	b = bytes.ReplaceAll(b, []byte("&"), []byte("&amp;"))
	b = bytes.ReplaceAll(b, []byte("&amp;amp;"), []byte("&amp;"))
	return b
}

func extractURLsFromXML(data []byte) ([]string, error) {
	var urls []string

	var us urlSet
	if xml.Unmarshal(data, &us) == nil && len(us.URLs) > 0 {
		log.Println("SITEMAP TYPE: urlset | URLs:", len(us.URLs))
		for _, u := range us.URLs {
			urls = append(urls, u.Loc)
		}
		return urls, nil
	}

	var si siteMapIndex
	if xml.Unmarshal(data, &si) == nil && len(si.Sitemaps) > 0 {
		log.Println("SITEMAP TYPE: index | CHILD SITEMAPS:", len(si.Sitemaps))
		for _, sm := range si.Sitemaps {
			log.Println("FOLLOW CHILD SITEMAP:", sm.Loc)
			time.Sleep(1 * time.Second)

			childData, _, _, err := makeRequest(sm.Loc)
			if err != nil {
				log.Println("CHILD FETCH FAILED:", sm.Loc)
				continue
			}

			childData = sanitizeXML(childData)
			childURLs, err := extractURLsFromXML(childData)
			if err != nil {
				log.Println("CHILD PARSE FAILED:", sm.Loc)
				continue
			}

			urls = append(urls, childURLs...)
		}
		return urls, nil
	}

	return nil, fmt.Errorf("unsupported sitemap format")
}

func parseHTML(htmlBytes []byte) (string, string, string) {
	doc, err := html.Parse(bytes.NewReader(htmlBytes))
	if err != nil {
		log.Println("HTML PARSE ERROR:", err)
		return "", "", ""
	}

	var title, metaDesc, canonical string

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				if n.FirstChild != nil {
					title = strings.TrimSpace(n.FirstChild.Data)
				}
			case "meta":
				var name, content string
				for _, a := range n.Attr {
					if a.Key == "name" && a.Val == "description" {
						name = a.Val
					}
					if a.Key == "content" {
						content = a.Val
					}
				}
				if name == "description" {
					metaDesc = content
				}
			case "link":
				for _, a := range n.Attr {
					if a.Key == "rel" && a.Val == "canonical" {
						for _, b := range n.Attr {
							if b.Key == "href" {
								canonical = b.Val
							}
						}
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(doc)
	return title, metaDesc, canonical
}

func worker(id int, jobs <-chan string, results chan<- CrawlResult, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Println("WORKER STARTED:", id)

	for url := range jobs {
		log.Println("WORKER", id, "FETCHING:", url)

		body, ct, status, err := makeRequest(url)
		if err != nil {
			log.Println("WORKER", id, "FAILED:", url)
			continue
		}

		if !strings.Contains(ct, "text/html") {
			results <- CrawlResult{URL: url, Status: status}
			continue
		}

		title, desc, canonical := parseHTML(body)

		results <- CrawlResult{
			URL:             url,
			Status:          status,
			Title:           title,
			MetaDescription: desc,
			Canonical:       canonical,
		}

		time.Sleep(300 * time.Millisecond)
	}

	log.Println("WORKER STOPPED:", id)
}

func crawlSiteMap(sitemapURL string) error {
	log.Println("START CRAWL:", sitemapURL)

	data, _, _, err := makeRequest(sitemapURL)
	if err != nil {
		return err
	}

	data = sanitizeXML(data)
	urls, err := extractURLsFromXML(data)
	if err != nil {
		return err
	}

	log.Println("TOTAL URLS DISCOVERED:", len(urls))

	jobs := make(chan string, 100)
	results := make(chan CrawlResult, 100)

	var wg sync.WaitGroup
	workerCount := 5

	log.Println("SPAWNING WORKERS:", workerCount)

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go worker(i, jobs, results, &wg)
	}

	go func() {
		for _, url := range urls {
			jobs <- url
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	for res := range results {
		log.Printf(
			"RESULT | URL=%s STATUS=%d TITLE=%q DESC=%q CANONICAL=%q",
			res.URL,
			res.Status,
			res.Title,
			res.MetaDescription,
			res.Canonical,
		)
	}

	log.Println("CRAWL COMPLETE")
	return nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if len(os.Args) < 2 {
		log.Println("usage: go run main.go <sitemap_url>")
		return
	}

	if err := crawlSiteMap(os.Args[1]); err != nil {
		log.Println("FATAL ERROR:", err)
	}
}
