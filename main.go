package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/gocolly/colly/v2"
)

type Scraper struct {
	collector   *colly.Collector
	visited     map[string]struct{}
	visitedLock sync.Mutex
	outputFile  string
	writer      *csv.Writer
	userAgent   string
	headless    bool
	baseDomain  string
	workerPool  chan struct{}
}

func NewScraper(startURL, outputFile, userAgent string, headless bool, depth, concurrency int) *Scraper {
	parsedURL, err := url.Parse(startURL)
	if err != nil {
		log.Fatal("URL invalide :", err)
	}
	baseDomain := parsedURL.Host

	collector := colly.NewCollector(
		colly.MaxDepth(depth),
		colly.AllowedDomains(baseDomain),
	)
	collector.Limit(&colly.LimitRule{Parallelism: concurrency, RandomDelay: 1 * time.Second})

	s := &Scraper{
		collector:  collector,
		visited:    make(map[string]struct{}),
		outputFile: outputFile,
		userAgent:  userAgent,
		headless:   headless,
		baseDomain: baseDomain,
		workerPool: make(chan struct{}, concurrency),
	}

	if outputFile != "" {
		file, err := os.Create(outputFile)
		if err != nil {
			log.Fatal("Erreur création fichier :", err)
		}
		writer := csv.NewWriter(file)
		writer.Write([]string{"URL", "Status"})
		s.writer = writer
		defer writer.Flush()
	}

	s.collector.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", s.userAgent)
	})

	s.collector.OnResponse(func(r *colly.Response) {
		fmt.Printf("Status %d -> %s\n", r.StatusCode, r.Request.URL)
		if s.writer != nil {
			s.writer.Write([]string{r.Request.URL.String(), fmt.Sprintf("%d", r.StatusCode)})
		}
	})

	s.collector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		absoluteURL := e.Request.AbsoluteURL(link)
		normalizedURL, err := normalizeURL(absoluteURL)
		if err != nil || !strings.Contains(normalizedURL, s.baseDomain) {
			return
		}

		s.visitedLock.Lock()
		if _, exists := s.visited[normalizedURL]; exists {
			s.visitedLock.Unlock()
			return
		}
		s.visited[normalizedURL] = struct{}{}
		s.visitedLock.Unlock()

		go s.scrapeURL(normalizedURL)
	})

	return s
}

func (s *Scraper) scrapeURL(url string) {
	s.workerPool <- struct{}{}
	defer func() { <-s.workerPool }()

	s.collector.Visit(url)
}

func (s *Scraper) Start(url string) {
	fmt.Println("Exploration de :", url)
	if err := s.collector.Visit(url); err != nil {
		log.Fatal(err)
	}
	s.collector.Wait()
	fmt.Println("Scraping terminé !")
}

func normalizeURL(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	parsed.Fragment = ""
	parsed.RawQuery = ""
	return parsed.String(), nil
}

func extractJSLinks(url string) ([]string, error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	var links []string

	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(3*time.Second),
		chromedp.Evaluate(`Array.from(document.querySelectorAll("a")).map(a => a.href)`, &links),
	)

	return links, err
}

func main() {
	startURL := flag.String("url", "", "URL de départ")
	outputFile := flag.String("output", "", "Fichier CSV de sortie")
	userAgent := flag.String("user-agent", "Mozilla/5.0", "User-Agent personnalisé")
	headless := flag.Bool("headless", false, "Activer le mode headless")
	depth := flag.Int("depth", 2, "Profondeur de scraping")
	concurrency := flag.Int("concurrency", 5, "Nombre de requêtes simultanées")
	flag.Parse()

	if *startURL == "" {
		log.Fatal("Usage: go run main.go -url=<URL>")
	}

	scraper := NewScraper(*startURL, *outputFile, *userAgent, *headless, *depth, *concurrency)
	scraper.Start(*startURL)
}
