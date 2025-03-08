package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/gocolly/colly/v2"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <URL>")
		os.Exit(1)
	}

	startURL := os.Args[1]

	maxDepth := flag.Int("depth", 2, "Profondeur maximale du scraping")
	asyncMode := flag.Bool("async", true, "Activer le mode asynchrone (true/false)")
	flag.Parse()

	c := colly.NewCollector(
		colly.MaxDepth(*maxDepth),
	)

	c.Async = *asyncMode

	var visited sync.Map

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		absoluteURL := e.Request.AbsoluteURL(link)

		if _, exists := visited.Load(absoluteURL); exists {
			return
		}

		visited.Store(absoluteURL, true)
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
