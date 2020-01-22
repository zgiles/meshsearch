package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/blevesearch/bleve"
	"github.com/zgiles/meshsearch/pkg/scraper"
)

var version string

type Config struct {
	ListenAddr string
	SaveFile   string
	Maxage     int64
	Verbose    bool
	Debug      bool
	UserAgent  string
	MaxDepth   int
	URLFilters []string
	Seeds      []string
}

func dumphandler(i *scraper.Scraper) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := i.ToJSON()
		if err != nil {
			log.Println("failed to serialize response:", err)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.Write(b)
	}
}

func searchhandler(i bleve.Index) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		keys, ok := r.URL.Query()["q"]
		if !ok || len(keys[0]) < 1 {
			http.Error(w, "Missing query", 401)
			return
		}
		fmt.Printf("Search %s\n", keys[0])
		query := bleve.NewQueryStringQuery(keys[0])
		sr := bleve.NewSearchRequestOptions(query, 10, 0, false)
		sr.Fields = []string{"URL", "Title", "Time"}
		sr.Highlight = bleve.NewHighlight()
		sr.Validate()
		s, err := i.Search(sr)
		if err != nil {
			http.Error(w, err.Error(), 401)
			return
		}
		b, err := json.Marshal(s)
		if err != nil {
			http.Error(w, err.Error(), 401)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.Write(b)
	}
}

func statshandler(i bleve.Index) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		dc, _ := i.DocCount()
		st := i.Stats()
		hostname, _ := os.Hostname()
		b, err := json.Marshal(struct {
			Index    *bleve.IndexStat
			DocCount uint64
			Hostname string
			Version  string
		}{
			Index:    st,
			DocCount: dc,
			Hostname: hostname,
			Version:  version,
		})
		if err != nil {
			http.Error(w, err.Error(), 401)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.Write(b)
	}
}

func loadConfig(filename string) (Config, error) {
	var c Config
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return c, err
	}
	if err = json.Unmarshal(file, &c); err != nil {
		return c, err
	}
	return c, nil
}

func main() {
	c, err := loadConfig("config.json")
	if err != nil {
		log.Fatal(err)
	}

	flag.StringVar(&c.ListenAddr, "listenaddr", c.ListenAddr, "Override ListenAddr")
	flag.Parse()

	i, err := scraper.NewScraper(scraper.ScraperConfig{
		SaveFile:   c.SaveFile,
		Maxage:     c.Maxage,
		Verbose:    c.Verbose,
		Debug:      c.Debug,
		UserAgent:  c.UserAgent,
		MaxDepth:   c.MaxDepth,
		URLFilters: c.URLFilters,
		Seeds:      c.Seeds,
	})
	if err != nil {
		log.Fatal(err)
	}

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGHUP)
	go func() {
		for range sigchan {
			fmt.Println("Reloading config...")
			c, err := loadConfig("config.json")
			if err != nil {
				log.Fatal(err)
				continue
			}
			i.UpdateConfig(scraper.ScraperConfig{
				SaveFile:   c.SaveFile,
				Maxage:     c.Maxage,
				Verbose:    c.Verbose,
				Debug:      c.Debug,
				UserAgent:  c.UserAgent,
				MaxDepth:   c.MaxDepth,
				URLFilters: c.URLFilters,
				Seeds:      c.Seeds,
			})
		}
	}()

	blevemapping := bleve.NewIndexMapping()
	bleveindex, err := bleve.NewMemOnly(blevemapping)
	if err != nil {
		log.Fatal(err)
	}

	i.PageAction(func(p *scraper.Page) {
		log.Println(p.URL, p)
		bleveindex.Index(p.URL, p)
	})

	//i.Scrape()
	go i.ScrapeSaveLoop()

	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/dump", dumphandler(i))
	http.HandleFunc("/search", searchhandler(bleveindex))
	http.HandleFunc("/stats", statshandler(bleveindex))
	log.Fatal(http.ListenAndServe(c.ListenAddr, nil))

}
