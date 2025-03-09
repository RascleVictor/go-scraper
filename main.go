package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"os"
	"strings"
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

	// Extraire le domaine
	parsedURL, err := url.Parse(startURL)
	if err != nil {
		log.Fatal("URL invalide :", err)
	}
	baseDomain := parsedURL.Host

	// Arguments CLI
	maxDepth := flag.Int("depth", 2, "Profondeur maximale du scraping")
	asyncMode := flag.Bool("async", true, "Activer le mode asynchrone (true/false)")
	outputFile := flag.String("output", "", "Fichier CSV de sortie (ex: results.csv)")
	proxy := flag.String("proxy", "", "Utiliser un proxy (ex: http://ip:port)")
	flag.Parse()

	// Initialisation du collector
	c := colly.NewCollector(
		colly.MaxDepth(*maxDepth),
		colly.AllowedDomains(baseDomain),
	)
	c.Async = *asyncMode

	// Ajout du multithreading
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 5,
		Delay:       500 * time.Millisecond,
	})

	var visited sync.Map

	var csvFile *os.File
	var writer *csv.Writer

	if *outputFile != "" {
		csvFile, err = os.Create(*outputFile)
		if err != nil {
			log.Fatal("Erreur lors de la création du fichier :", err)
		}
		defer csvFile.Close()

		writer = csv.NewWriter(csvFile)
		defer writer.Flush()
		writer.Write([]string{"URL", "Statut HTTP"})
	}

	// Configuration du proxy si nécessaire
	if *proxy != "" {
		c.SetProxy(*proxy)
	}

	// Liste de User-Agents aléatoires
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, comme Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, comme Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, comme Gecko) Firefox/89.0",
	}

	// Simulation du comportement humain
	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])
		time.Sleep(time.Duration(rand.Intn(3)+1) * time.Second)
	})

	// Extraction des liens
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		absoluteURL := e.Request.AbsoluteURL(link)

		// Vérification que l'URL appartient au domaine de base
		if !strings.Contains(absoluteURL, baseDomain) {
			return
		}

		// Vérification si l'URL a déjà été visitée
		if _, exists := visited.Load(absoluteURL); exists {
			return
		}
		visited.Store(absoluteURL, true)

		// Ajout au CSV si nécessaire
		if writer != nil {
			writer.Write([]string{absoluteURL, "Pending"})
		}

		fmt.Println("Lien trouvé :", absoluteURL)
		c.Visit(absoluteURL)
	})

	// Capture des statuts HTTP
	c.OnResponse(func(r *colly.Response) {
		fmt.Printf("Statut [%d] - %s\n", r.StatusCode, r.Request.URL.String())

		if writer != nil {
			writer.Write([]string{r.Request.URL.String(), fmt.Sprintf("%d", r.StatusCode)})
			writer.Flush()
		}
	})

	// Gestion des erreurs
	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Erreur [%d] - %s : %v\n", r.StatusCode, r.Request.URL.String(), err)
	})

	// Lancement du scraping
	fmt.Println("Exploration de :", startURL)
	err = c.Visit(startURL)
	if err != nil {
		log.Fatal(err)
	}

	c.Wait()
	fmt.Println("Scraping terminé !")
}
