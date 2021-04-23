package tinycast

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"html"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"time"

	podgrabm "github.com/akhilrex/podgrab/model"
	podgrabs "github.com/akhilrex/podgrab/service"
	"github.com/patrickmn/go-cache"
	"pipelined.dev/audio/mp3"

	"github.com/beevik/etree"
	"github.com/gin-gonic/gin"
	"github.com/microcosm-cc/bluemonday"
)

// App is the core tinycast web app with handlers and state.
type App struct {
	baseURL url.URL
	cache   *cache.Cache
	apiKey  string
}

func hashAPIKey(in string) string {
	by := sha1.Sum([]byte(in))
	return hex.EncodeToString(by[:])
}

// NewApp returns a new App based on configuration.
func NewApp(baseURL url.URL, apiKey string) *App {
	return &App{
		baseURL: baseURL,
		cache:   cache.New(5*time.Minute, 10*time.Minute),
		apiKey:  hashAPIKey(apiKey),
	}
}

// Convert is a handler which re-encodes an audio stream to MP3 on the fly.
func (a *App) Convert(c *gin.Context) {
	if !a.verifyAPIKey(c) {
		c.Status(http.StatusForbidden)
		return
	}
	var cfg ConversionConfig
	var err error
	if cfg, err = BindConversionConfig(c); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	//parsedUrl, err := url.Parse(cfg.Url)
	//if err != nil {
	//  c.HTML(http.StatusInternalServerError, fmt.Sprintf("Failed to convert: %s", err), nil)
	//}

	c.Writer.Header().Add("Content-Type", "audio/mpeg")
	//c.Writer.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%s", path.Base(parsedUrl.Path)))
	err = transform(c, cfg, c.Writer)
	//if err != nil {
	//  c.HTML(http.StatusInternalServerError, fmt.Sprintf("Failed to convert: %s", err), nil)
	//}
	if err != nil {
		log.Println("Err: ", err)
	}
}

// ConversionConfig holds the desired conversion for an audio file.
type ConversionConfig struct {
	URL         string
	BitRateMode BitRateMode
	BitRate     BitRate
	ChannelMode mp3.ChannelMode
}

func (a *App) verifyAPIKey(c *gin.Context) bool {
	return a.apiKey == c.Query("key")
}

// BindConversionConfig captures and validates a ConversionConfig passed in
// query parameters.
func BindConversionConfig(c *gin.Context) (ConversionConfig, error) {
	var cfg ConversionConfig
	var err error

	if cfg.URL = c.Query("url"); cfg.URL == "" {
		return cfg, fmt.Errorf("url required")
	}
	if cfg.BitRateMode, err = ParseBitRateMode(c.Query("bitRateMode")); err != nil {
		return cfg, err
	}
	if cfg.BitRate, err = ParseBitRate(c.Query("bitRate")); err != nil {
		return cfg, err
	}
	if cfg.ChannelMode, err = ParseChannelMode(c.Query("channelMode")); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// ToQueryValues translates a ConversionCongig to the same query parameters
// expected by BindConversionConfig.
func (cfg ConversionConfig) ToQueryValues() url.Values {
	q := url.Values{}
	q.Add("bitRateMode", string(cfg.BitRateMode))
	q.Add("bitRate", cfg.BitRate.ToString())
	q.Add("channelMode", cfg.ChannelMode.String())
	return q
}

// Feed is a handler which replaces all download entries in a podcast feed with
// a URL handled by the Conversion handler.
func (a *App) Feed(c *gin.Context) {
	if !a.verifyAPIKey(c) {
		c.Status(http.StatusForbidden)
		return
	}
	var cfg ConversionConfig
	var err error
	if cfg, err = BindConversionConfig(c); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	qr := cfg.ToQueryValues()
	qr.Add("key", a.apiKey)
	resp, err := http.Get(cfg.URL)
	if err != nil {
		log.Println(err)
		c.Status(http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	doc := etree.NewDocument()
	_, err = doc.ReadFrom(resp.Body)
	if err != nil {
		log.Println(err)
		c.Status(http.StatusInternalServerError)
		return
	}
	replace(doc, "//enclosure", a.baseURL, qr)
	replace(doc, "//media:content", a.baseURL, qr)
	for _, t := range doc.FindElements("//channel/title") {
		t.SetText(fmt.Sprintf("(Tiny) %s", t.Text()))
	}
	for _, t := range doc.FindElements("//item/title") {
		t.SetText(fmt.Sprintf("(Tiny) %s", t.Text()))
	}
	_, err = doc.WriteTo(c.Writer)
	if err != nil {
		log.Println("Failed to finish writing feed:", err)
		c.Status(http.StatusInternalServerError)
		return
	}
}

func (a *App) search(query string) ([]*podgrabm.CommonSearchResultModel, error) {
	cacheKey := fmt.Sprintf("s-%s", query)
	cacheResult, found := a.cache.Get(cacheKey)
	if found {
		v, ok := cacheResult.([]*podgrabm.CommonSearchResultModel)
		if ok {
			return v, nil
		}
	}
	searcher := new(podgrabs.PodcastIndexService)
	results := searcher.Query(query)

	// Cleanup the HTML in the podcast descriptions.
	for _, r := range results {
		r.Description = html.UnescapeString(sanitizeString(r.Description))
	}
	go func() {
		a.cache.Set(cacheKey, results, cache.DefaultExpiration)
	}()
	return results, nil
}

var bmStrictPolicy = bluemonday.StrictPolicy()

func sanitizeString(input string) string {
	return bmStrictPolicy.Sanitize(input)
}

// Home is a handler which allows users to search for podcasts to subscribe to.
func (a *App) Home(c *gin.Context) {
	query := c.Query("q")
	var results []*podgrabm.CommonSearchResultModel
	var err error
	var p Pagination
	if query != "" {
		results, err = a.search(query)
		if err != nil {
			log.Println("Failed search:", err)
		}

		var currentPage int64
		if currentPage, err = strconv.ParseInt(c.Query("p"), 10, 32); err != nil {
			currentPage = 0
		}
		if currentPage < 0 {
			currentPage = 0
		}
		p = Pagination{
			numItems:     len(results),
			itemsPerPage: 10,
			curPage:      int(currentPage),
		}
		if int(currentPage) > p.NumPages() {
			p.curPage = 0
		}

		lastOffset := len(results)
		if o := p.FirstItem() + p.itemsPerPage; o < lastOffset {
			lastOffset = o
		}
		results = results[p.FirstItem():lastOffset]
	}

	podcastURL := a.baseURL
	podcastURL.Scheme = "podcast"
	podcastURL.Path = "/feed"
	log.Println(podcastURL.String())

	c.HTML(http.StatusOK, "main.tmpl", gin.H{
		"title":           "TinyCast",
		"h1":              "TinyCast",
		"query":           sanitizeString(query),
		"searchResults":   results,
		"pagination":      p,
		"channelModes":    ChannelModes,
		"bitRateModes":    BitRateModes,
		"bitRates":        BitRates,
		"applePodcastUrl": template.URL(podcastURL.String()),
		"apiKey":          a.apiKey,
	})
}
