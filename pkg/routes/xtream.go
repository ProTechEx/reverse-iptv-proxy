package routes

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jamesnetherton/m3u"
	proxyM3U "github.com/pierre-emmanuelJ/iptv-proxy/pkg/m3u"
	xtreamapi "github.com/pierre-emmanuelJ/iptv-proxy/pkg/xtream-proxy"
)

func (p *proxy) xtreamGet(c *gin.Context) {
	rawURL := fmt.Sprintf("%s/get.php?username=%s&password=%s", p.XtreamBaseURL, p.XtreamUser, p.XtreamPassword)

	q := c.Request.URL.Query()

	for k, v := range q {
		if k == "username" || k == "password" {
			continue
		}

		rawURL = fmt.Sprintf("%s&%s=%s", rawURL, k, strings.Join(v, ","))
	}

	m3uURL, err := url.Parse(rawURL)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	playlist, err := m3u.Parse(m3uURL.String())
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	newM3U, err := proxyM3U.ReplaceURL(&playlist, p.User, p.Password, p.HostConfig)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	result, err := proxyM3U.Marshall(newM3U)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.Header("Content-Disposition", "attachment; filename=\"iptv.m3u\"")
	c.Data(http.StatusOK, "application/octet-stream", []byte(result))
}

func (p *proxy) xtreamPlayerAPIGET(c *gin.Context) {
	p.xtreamPlayerAPI(c, c.Request.URL.Query())
}

func (p *proxy) xtreamPlayerAPIPOST(c *gin.Context) {
	contents, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	q, err := url.ParseQuery(string(contents))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	p.xtreamPlayerAPI(c, q)
}

func (p *proxy) xtreamPlayerAPI(c *gin.Context, q url.Values) {
	var action string
	if len(q["action"]) > 0 {
		action = q["action"][0]
	}

	client, err := xtreamapi.New(p.XtreamUser, p.XtreamPassword, p.XtreamBaseURL)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	var respBody interface{}

	switch action {
	case xtreamapi.GetLiveCategories:
		respBody, err = client.GetLiveCategories()
	case xtreamapi.GetLiveStreams:
		respBody, err = client.GetLiveStreams("")
	case xtreamapi.GetVodCategories:
		respBody, err = client.GetVideoOnDemandCategories()
	case xtreamapi.GetVodStreams:
		respBody, err = client.GetVideoOnDemandStreams("")
	case xtreamapi.GetVodInfo:
		if len(q["vod_id"]) < 1 {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf(`bad body url query parameters: missing "vod_id"`))
			return
		}
		respBody, err = client.GetVideoOnDemandInfo(q["vod_id"][0])
	case xtreamapi.GetSeriesCategories:
		respBody, err = client.GetSeriesCategories()
	case xtreamapi.GetSeries:
		respBody, err = client.GetSeries("")
	case xtreamapi.GetSerieInfo:
		if len(q["series_id"]) < 1 {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf(`bad body url query parameters: missing "series_id"`))
			return
		}
		respBody, err = client.GetSeriesInfo(q["series_id"][0])
	default:
		respBody, err = client.Login(p.User, p.Password, "http://"+p.HostConfig.Hostname, int(p.HostConfig.Port))
	}

	log.Printf("[iptv-proxy] %v | %s |Action\t%s\n", time.Now().Format("2006/01/02 - 15:04:05"), c.ClientIP(), action)

	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, respBody)
}

func (p *proxy) xtreamXMLTV(c *gin.Context) {
	client, err := xtreamapi.New(p.XtreamUser, p.XtreamPassword, p.XtreamBaseURL)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	resp, err := client.GetXMLTV()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.Data(http.StatusOK, "application/xml", resp)
}

func (p *proxy) xtreamStream(c *gin.Context) {
	id := c.Param("id")
	rpURL, err := url.Parse(fmt.Sprintf("%s/%s/%s/%s", p.XtreamBaseURL, p.XtreamUser, p.XtreamPassword, id))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	stream(c, rpURL)
}

func (p *proxy) xtreamStreamLive(c *gin.Context) {
	id := c.Param("id")
	rpURL, err := url.Parse(fmt.Sprintf("%s/live/%s/%s/%s", p.XtreamBaseURL, p.XtreamUser, p.XtreamPassword, id))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	stream(c, rpURL)
}

func (p *proxy) xtreamStreamMovie(c *gin.Context) {
	id := c.Param("id")
	rpURL, err := url.Parse(fmt.Sprintf("%s/movie/%s/%s/%s", p.XtreamBaseURL, p.XtreamUser, p.XtreamPassword, id))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	stream(c, rpURL)
}

func (p *proxy) xtreamStreamSeries(c *gin.Context) {
	id := c.Param("id")
	rpURL, err := url.Parse(fmt.Sprintf("%s/series/%s/%s/%s", p.XtreamBaseURL, p.XtreamUser, p.XtreamPassword, id))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	stream(c, rpURL)
}

func (p *proxy) hlsrStream(c *gin.Context) {
	req, err := url.Parse(fmt.Sprintf("%s%s", p.XtreamBaseURL, c.Request.URL.String()))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	stream(c, req)
}
