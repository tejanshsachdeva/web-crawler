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
	"time"
)

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

func makeRequest(url string) ([]byte, string, error) {
	log.Println("REQUEST:", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", err
	}

	req.Header.Set("User-Agent", randomUserAgent())
	req.Header.Set("Accept", "application/xml,text/xml,*/*")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	log.Println("RESPONSE:", resp.Status, "Content-Type:", ct)

	var reader io.Reader = resp.Body
	if strings.HasSuffix(url, ".gz") {
		log.Println("GZIP detected:", url)
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, ct, err
		}
		defer gz.Close()
		reader = gz
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, ct, err
	}

	return body, ct, nil
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
		log.Println("Parsed urlset with", len(us.URLs), "URLs")
		for _, u := range us.URLs {
			urls = append(urls, u.Loc)
		}
		return urls, nil
	}

	var si siteMapIndex
	if xml.Unmarshal(data, &si) == nil && len(si.Sitemaps) > 0 {
		log.Println("Parsed sitemap index with", len(si.Sitemaps), "children")
		for _, sm := range si.Sitemaps {
			log.Println("Following child sitemap:", sm.Loc)
			time.Sleep(2 * time.Second)

			childData, _, err := makeRequest(sm.Loc)
			if err != nil {
				log.Println("Child sitemap failed:", err)
				continue
			}

			childData = sanitizeXML(childData)
			childURLs, err := extractURLsFromXML(childData)
			if err != nil {
				log.Println("Child parse failed:", err)
				continue
			}

			urls = append(urls, childURLs...)
		}
		return urls, nil
	}

	return nil, fmt.Errorf("unsupported sitemap format")
}

func scrapePage(url string) {
	log.Println("SCRAPE PAGE:", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}

	req.Header.Set("User-Agent", randomUserAgent())
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Println("Fetch failed:", err)
		return
	}
	defer resp.Body.Close()

	log.Println("SCRAPED:", url, "Status:", resp.StatusCode)
}

func scrapeSiteMap(sitemapURL string) error {
	log.Println("START SITEMAP:", sitemapURL)

	data, ct, err := makeRequest(sitemapURL)
	if err != nil {
		return err
	}

	if strings.Contains(ct, "text/html") {
		return fmt.Errorf("received HTML instead of XML")
	}

	data = sanitizeXML(data)
	urls, err := extractURLsFromXML(data)
	if err != nil {
		return err
	}

	log.Println("TOTAL URLS FOUND:", len(urls))

	for _, u := range urls {
		scrapePage(u)
		time.Sleep(300 * time.Millisecond)
	}

	return nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if len(os.Args) < 2 {
		log.Println("usage: go run main.go <sitemap_url>")
		return
	}

	if err := scrapeSiteMap(os.Args[1]); err != nil {
		log.Println("ERROR:", err)
	}
}
