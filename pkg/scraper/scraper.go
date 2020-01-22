package scraper

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/gocolly/colly"

	// "github.com/gocolly/colly/debug"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

var hassomealpha = regexp.MustCompile(`[A-Za-z]+`).MatchString
var space = regexp.MustCompile(`\s+`)

type Page struct {
	URL     string
	Summary string
	Title   string
	Links   map[string]string
	Time    int64
}

type SeedSite struct {
	Site      string
	LastCheck int64
}

type ScraperConfig struct {
	SaveFile   string
	Maxage     int64
	Verbose    bool
	Debug      bool
	UserAgent  string
	MaxDepth   int
	URLFilters []string
	Seeds      []string
}

type ScraperData struct {
	Pages     map[string]*Page
	Watchlist map[string]*SeedSite
}

type Scraper struct {
	sync.RWMutex
	ScraperConfig
	data       *ScraperData
	pageAction func(*Page)
	urlFilters []*regexp.Regexp
}

func NewScraper(config ScraperConfig) (*Scraper, error) {
	c := config
	if c.Maxage == 0 {
		c.Maxage = 3600
	}

	if c.UserAgent == "" {
		c.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/79.0.3945.88 Safari/537.36"
	}

	if c.MaxDepth == 0 {
		c.MaxDepth = 7
	}

	data := &ScraperData{
		Watchlist: make(map[string]*SeedSite),
		Pages:     make(map[string]*Page),
	}

	i := Scraper{
		ScraperConfig: c,
		data:          data,
		pageAction:    func(*Page) {},
		urlFilters:    []*regexp.Regexp{},
	}
	return &i, nil
}

func (i *Scraper) PageAction(paf func(*Page)) {
	i.pageAction = paf
}

func (i *Scraper) reconsileFilters() {
	var f []*regexp.Regexp
	for _, s := range i.URLFilters {
		f = append(f, regexp.MustCompile(s))
	}
	i.Lock()
	i.urlFilters = f
	i.Unlock()
}

func (i *Scraper) reconsileSeeds() {
	i.Lock()
	for _, s := range i.Seeds {
		if _, ok := i.data.Watchlist[s]; !ok {
			i.v("Adding seed %s\n", s)
			i.data.Watchlist[s] = &SeedSite{
				Site:      s,
				LastCheck: 0,
			}
		}
	}
	i.Unlock()
}

func (i *Scraper) setupcollector() *colly.Collector {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// colly.Debugger(&debug.LogDebugger{}),
	c := colly.NewCollector(
		colly.UserAgent(i.UserAgent),
		colly.URLFilters(i.urlFilters...),
		colly.MaxDepth(i.MaxDepth),
	)
	c.WithTransport(tr)

	c.OnRequest(func(r *colly.Request) {
		url := r.URL.String()
		r.Ctx.Put("url", url)
		i.v("Visiting %s\n", url)
		i.Lock()
		i.data.Pages[url] = &Page{
			URL:   url,
			Links: make(map[string]string),
			Time:  time.Now().Unix(),
		}
		i.Unlock()
	})

	c.OnHTML("title", func(e *colly.HTMLElement) {
		url := e.Response.Ctx.Get("url")
		title := strings.TrimSpace(space.ReplaceAllString(e.Text, " "))
		i.v("Title: %s %s\n", title, url)
		i.Lock()
		i.data.Pages[url].Title = title
		i.Unlock()
	})

	c.OnHTML("meta", func(e *colly.HTMLElement) {
		url := e.Response.Ctx.Get("url")
		property := strings.TrimSpace(space.ReplaceAllString(e.Attr("property"), " "))
		content := strings.TrimSpace(space.ReplaceAllString(e.Attr("content"), " "))
		switch property {
		case "og:description", "og:title":
			i.d(" Meta %s\n", content)
			i.Lock()
			i.data.Pages[url].Summary = i.data.Pages[url].Summary + content + " "
			i.Unlock()
		}
	})

	c.OnHTML("h1", func(e *colly.HTMLElement) {
		url := e.Response.Ctx.Get("url")
		text := strings.TrimSpace(space.ReplaceAllString(e.Text, " "))
		i.d(" Body H1 %s\n", text)
		i.Lock()
		i.data.Pages[url].Summary = i.data.Pages[url].Summary + text + " "
		i.Unlock()
	})
	c.OnHTML("h2", func(e *colly.HTMLElement) {
		url := e.Response.Ctx.Get("url")
		text := strings.TrimSpace(space.ReplaceAllString(e.Text, " "))
		i.d(" Body H2 %s\n", text)
		i.Lock()
		i.data.Pages[url].Summary = i.data.Pages[url].Summary + text + " "
		i.Unlock()
	})
	c.OnHTML("p", func(e *colly.HTMLElement) {
		url := e.Response.Ctx.Get("url")
		text := strings.TrimSpace(space.ReplaceAllString(e.Text, " "))
		i.d(" Body P %s\n", text)
		i.Lock()
		i.data.Pages[url].Summary = i.data.Pages[url].Summary + text + " "
		i.Unlock()
	})
	c.OnHTML("pre", func(e *colly.HTMLElement) {
		url := e.Response.Ctx.Get("url")
		text := strings.TrimSpace(space.ReplaceAllString(e.Text, " "))
		i.d(" Body pre %s\n", text)
		i.Lock()
		i.data.Pages[url].Summary = i.data.Pages[url].Summary + text + " "
		i.Unlock()
	})
	c.OnHTML("code", func(e *colly.HTMLElement) {
		url := e.Response.Ctx.Get("url")
		text := strings.TrimSpace(space.ReplaceAllString(e.Text, " "))
		i.d(" Body code %s\n", text)
		i.Lock()
		i.data.Pages[url].Summary = i.data.Pages[url].Summary + text + " "
		i.Unlock()
	})
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		url := e.Response.Ctx.Get("url")
		link := e.Request.AbsoluteURL(e.Attr("href"))
		text := strings.TrimSpace(space.ReplaceAllString(e.Text, " "))
		if len(link) > 4 && strings.Contains(link, "http") && hassomealpha(text) && len(text) > 3 {
			i.d(" Link: %s %s\n", text, link)
			i.Lock()
			i.data.Pages[url].Links[text] = link
			i.data.Pages[url].Summary = i.data.Pages[url].Summary + text + ": " + link + " "
			i.Unlock()
			// For now assuming we should probably chop off the questionmark part
			linksplit := strings.Split(link, "?")
			if len(linksplit) >= 1 {
				e.Request.Visit(linksplit[0])
			}
		}
	})
	c.OnScraped(func(r *colly.Response) {
		url := r.Ctx.Get("url")
		i.RLock()
		i.v("Scraped: %s\n", url)
		i.pageAction(i.data.Pages[url])
		i.RUnlock()
	})
	return c
}

func (i *Scraper) UpdateConfig(c ScraperConfig) {
	i.Lock()
	i.ScraperConfig = c
	i.Unlock()
}

func (i *Scraper) Scrape() {
	// initialize a fresh scraper to avoid repeats in this session
	i.reconsileFilters()
	i.reconsileSeeds()
	c := i.setupcollector()
	// Go over watch list
	for _, s := range i.data.Watchlist {
		t := time.Now().Unix()
		if s.LastCheck+i.Maxage < t {
			i.v("Max age expired: %s - %d < %d\n", s.Site, s.LastCheck+i.Maxage, t)
			c.Visit(s.Site)
			s.LastCheck = time.Now().Unix()
		} else {
			i.v("Max age not expired: %s - %d !< %d\n", s.Site, s.LastCheck+i.Maxage, t)
		}
	}
}

func (i *Scraper) ScrapeSaveLoop() {
	i.Load()
	i.Scrape()
	i.Save()
	i.v("Finished first scrape\n")
	timer := time.NewTicker(time.Minute)
	for {
		select {
		case <-timer.C:
			i.Scrape()
			i.Save()
			i.v("Finished a scrape\n")
		}
	}
	i.data.Pages = make(map[string]*Page)
}

func (i *Scraper) ToJSON() ([]byte, error) {
	i.RLock()
	defer i.RUnlock()
	return json.Marshal(i.data)
}

func (i *Scraper) Save() error {
	if i.SaveFile == "" {
		return nil
	}
	b, err := i.ToJSON()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(i.SaveFile, b, 0644)
}

func (i *Scraper) Load() error {
	i.v("Loading previous JSON")
	if i.SaveFile == "" {
		return nil
	}
	var data ScraperData
	file, err := ioutil.ReadFile(i.SaveFile)
	if err != nil {
		return err
	}
	err = json.Unmarshal(file, &data)
	if err != nil {
		return err
	}
	i.Lock()
	i.data = &data
	i.v("Done Loading.. Indexing")
	for page := range i.data.Pages {
		fmt.Println(page)
		i.pageAction(i.data.Pages[page])
	}
	i.Unlock()
	i.v("Done Indexing.")
	return nil
}

func (i *Scraper) v(format string, params ...interface{}) {
	if i.Verbose {
		log.Printf(format, params...)
	}
}

func (i *Scraper) d(format string, params ...interface{}) {
	if i.Debug {
		log.Printf(format, params...)
	}
}
