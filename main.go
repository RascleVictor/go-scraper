package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <URL> [-depth=<depth>] [-async=<true/false>] [-output=<file.csv>] [-proxy=<proxy-url>]")
		os.Exit(1)
	}

	startURL := os.Args[1]

	maxDepth := flag.Int("depth", 2, "Profondeur maximale du scraping")
	asyncMode := flag.Bool("async", true, "Activer le mode asynchrone (true/false)")
	outputFile := flag.String("output", "", "Fichier CSV de sortie (ex: results.csv)")
	proxy := flag.String("proxy", "", "Utiliser un proxy (ex: http://ip:port)")
	flag.Parse()

	c := colly.NewCollector(
		colly.MaxDepth(*maxDepth),
	)

	c.Async = *asyncMode

	var visited sync.Map

	var csvFile *os.File
	var writer *csv.Writer

	if *outputFile != "" {
		var err error
		csvFile, err = os.Create(*outputFile)
		if err != nil {
			log.Fatal("Erreur lors de la création du fichier :", err)
		}
		defer csvFile.Close()

		writer = csv.NewWriter(csvFile)
		defer writer.Flush()
		writer.Write([]string{"URL trouvée"})
	}

	if *proxy != "" {
		c.SetProxy(*proxy)
	}

	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Firefox/89.0",
	}

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])
		time.Sleep(time.Duration(rand.Intn(3)+1) * time.Second)
	})

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		absoluteURL := e.Request.AbsoluteURL(link)

		if _, exists := visited.Load(absoluteURL); exists {
			return
		}
		visited.Store(absoluteURL, true)

		if writer != nil {
			writer.Write([]string{absoluteURL})
		}

		fmt.Println("Lien trouvé :", absoluteURL)
		c.Visit(absoluteURL)
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Println("Erreur :", err)
	})

	fmt.Println("Exploration de :", startURL)
	err := c.Visit(startURL)
	if err != nil {
		log.Fatal(err)
	}

	c.Wait()
	fmt.Println("Scraping terminé !")
}
