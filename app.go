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

	_ "net/http/pprof"

	podgrabm "github.com/akhilrex/podgrab/model"
	podgrabs "github.com/akhilrex/podgrab/service"
	"github.com/patrickmn/go-cache"
	"pipelined.dev/audio/mp3"

	"github.com/beevik/etree"
	"github.com/gin-gonic/gin"
	"github.com/microcosm-cc/bluemonday"
)

type App struct {
	BaseUrl url.URL
	cache   *cache.Cache
	apiKey  string
}

func hashApiKey(in string) string {
	by := sha1.Sum([]byte(in))
	return hex.EncodeToString(by[:])
}

func NewApp(baseUrl url.URL, apiKey string) *App {
	return &App{
		BaseUrl: baseUrl,
		cache:   cache.New(5*time.Minute, 10*time.Minute),
		apiKey:  hashApiKey(apiKey),
	}
}

func (a *App) Get(c *gin.Context) {
	if !a.verifyApiKey(c) {
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

type ConversionConfig struct {
	Url         string
	BitRateMode BitRateMode
	BitRate     BitRate
	ChannelMode mp3.ChannelMode
}

func (a *App) verifyApiKey(c *gin.Context) bool {
	return a.apiKey == c.Query("key")
}

func BindConversionConfig(c *gin.Context) (ConversionConfig, error) {
	var cfg ConversionConfig
	var err error

	if cfg.Url = c.Query("url"); cfg.Url == "" {
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

func (cfg ConversionConfig) ToQueryValues() url.Values {
	q := url.Values{}
	q.Add("bitRateMode", string(cfg.BitRateMode))
	q.Add("bitRate", cfg.BitRate.ToString())
	q.Add("channelMode", cfg.ChannelMode.String())
	return q
}

func (a *App) Feed(c *gin.Context) {
	if !a.verifyApiKey(c) {
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
	resp, err := http.Get(cfg.Url)
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()

	doc := etree.NewDocument()
	_, err = doc.ReadFrom(resp.Body)
	replace(doc, "//enclosure", a.BaseUrl, qr)
	replace(doc, "//media:content", a.BaseUrl, qr)
	for _, t := range doc.FindElements("//channel/title") {
		t.SetText(fmt.Sprintf("(Tiny) %s", t.Text()))
	}
	for _, t := range doc.FindElements("//item/title") {
		t.SetText(fmt.Sprintf("(Tiny) %s", t.Text()))
	}
	doc.WriteTo(c.Writer)
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
	var searcher podgrabs.SearchService
	searcher = new(podgrabs.PodcastIndexService)
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
			currentPage = 0
			p.curPage = 0
		}

		lastOffset := len(results)
		if o := p.FirstItem() + p.itemsPerPage; o < lastOffset {
			lastOffset = o
		}
		results = results[p.FirstItem():lastOffset]
	}

	podcastUrl := a.BaseUrl
	podcastUrl.Scheme = "podcast"
	podcastUrl.Path = "/feed"
	log.Println(podcastUrl.String())

	c.HTML(http.StatusOK, "main.tmpl", gin.H{
		"title":           "TinyCast",
		"h1":              "TinyCast",
		"query":           sanitizeString(query),
		"searchResults":   results,
		"pagination":      p,
		"channelModes":    ChannelModes,
		"bitRateModes":    BitRateModes,
		"bitRates":        BitRates,
		"applePodcastUrl": template.URL(podcastUrl.String()),
		"apiKey":          a.apiKey,
	})
}
