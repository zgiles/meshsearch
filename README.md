# meshsearch

An offline self-contained search engine for mesh/distributed networks

## About
`meshsearch` is a golang program that brings together `bleve` full-text indexing,
and `gocolly` for page scraping, along with a nice simple main search page that
provides a full, or a base for an offline search engine.

## How it works
* We start with a set of seed sites to branch off of, along with a maxdepth.
* The search engine will crawl and "summarize" the pages it encounters, following the links
it learns of ( up to maxdepth ), summarizing and linking those pages as well.
* Upon a page being scraped, an external function is called, provided by the full-text indexer.
* The indexer indexes the data structure containing the summary in a way that can be retrieved and displayed well.
* After a scraping cycle, the scraped content and "last visited time" are/can be exported via a json file for later use.
  - The json file can be loaded at start and fed to the indexer to provide a fast-start without having to re-scrape the world.
  - Future work will be to have many search heads that share scrape results.
* Search page is made available via the in-built web server and API server.
* Searches are via the API, returning JSON to the front end.

## How to build
Type `make`

or

`go build`

The docker container can be built like so:
```
docker build --network homenet -t <yourname/meshsearch:latest .
```
( Naturally replace `<yourname>` etc with your preferred tag, if any; homenet is optional, but would probably be necessary/useful to download the go modules for building )

Resulting container image is a small image with the binary, config, static components, and alpine linux.

It works with `scratch` also, but alpine helps with debugging for now.

## Base your search on this one
You may want to replace the logo or template page. This can easily be done with an overlay.

Simply make a new repo with a `Dockerfile` that overlays this one and replaces the config, and static assets. Like so:
```
FROM zgiles/meshsearch:latest
WORKDIR /
ADD config.json /config.json
ADD logo.svg /static/logo.svg
CMD ["/meshsearch"]
```
Then build the new container.

## Todo / Future
* Distributed gossip cluster to hand-off discovered pages
* Template and bindata-embed index page instead of static page ( less JS on frontend )
* All JS on frontend optional
* Add seeds from discovered pages
* Add seeds from frontend
* Derive top-sites "what's cool" list from search results
* Save Bleve index instead of []Pages?
* Better CLI flags and Env Vars
* Think about moving the main package to `cmd/meshsearch` and reduce CLI surface area

## LICENSE:
Copyright (c) 2020 Zachary Giles
Licensed under the MIT/Expat License
