/*
 *
 * Copyright (c) 2018, Andy Janata
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without modification, are permitted
 * provided that the following conditions are met:
 *
 * * Redistributions of source code must retain the above copyright notice, this list of conditions
 *   and the following disclaimer.
 * * Redistributions in binary form must reproduce the above copyright notice, this list of
 *   conditions and the following disclaimer in the documentation and/or other materials provided
 *   with the distribution.
 * * Neither the name of the copyright holder nor the names of its contributors may be used to
 *   endorse or promote products derived from this software without specific prior written
 *   permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR
 * IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND
 * FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR
 * CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
 * DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
 * DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY,
 * WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY
 * WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 *
 */

package faapi

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

var (
	ErrNotLoggedIn = errors.New("not logged in")
)

// Client is a FurAffinity client.
type Client struct {
	http                 http.Client
	config               Config
	journalRegexp        *regexp.Regexp
	rateLimiter          *time.Ticker
	submissionDataRegexp *regexp.Regexp
}

// New creates a new Client with the given configuration.
func New(config Config) (*Client, error) {
	journalRegexp, err := regexp.Compile(`^/journal/(\d+)/$`)
	if err != nil {
		return nil, err
	}

	submissionDataRegexp, err := regexp.Compile(`var submission_data = (.*}});`)
	if err != nil {
		return nil, err
	}

	tr := http.Transport{}
	if config.Proxy != "" {
		purl, err := url.Parse(config.Proxy)
		if err != nil {
			return nil, err
		}
		tr.Proxy = http.ProxyURL(purl)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	curl, err := url.Parse("https://www.furaffinity.net/")
	if err != nil {
		return nil, err
	}
	cookies := make([]*http.Cookie, len(config.Cookies))
	for i, cookie := range config.Cookies {
		cookies[i] = &http.Cookie{
			Name:  cookie.Name,
			Value: cookie.Value,
		}
	}
	jar.SetCookies(curl, cookies)

	return &Client{
		http: http.Client{
			Jar:       jar,
			Timeout:   15 * time.Second,
			Transport: &tr,
		},
		config:               config,
		journalRegexp:        journalRegexp,
		rateLimiter:          time.NewTicker(config.RateLimit),
		submissionDataRegexp: submissionDataRegexp,
	}, nil
}

func (c *Client) Close() {
	c.rateLimiter.Stop()
}

func (c *Client) newRequest(method, uri string, body io.Reader) (*http.Request, error) {
	log.WithField("uri", uri).Debug("Creating new request")
	if !strings.HasPrefix(uri, "https://") {
		uri = "https://www.furaffinity.net" + uri
	}
	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", c.config.UserAgent)
	return req, nil
}

func (c *Client) doRaw(req *http.Request) (*http.Response, error) {
	log.WithFields(log.Fields{
		"url":    req.URL,
		"method": req.Method,
	}).Debug("Making request")

	// wait for rate limiting
	<-c.rateLimiter.C

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		bb, _ := ioutil.ReadAll(res.Body)
		log.WithFields(log.Fields{
			"url":  req.URL,
			"code": res.StatusCode,
			"body": string(bb),
		}).Error("Unexpected HTTP response code")
		return nil, fmt.Errorf("HTTP response %d not expected", res.StatusCode)
	}

	return res, nil
}

func (c *Client) do(req *http.Request) (*html.Node, error) {
	res, err := c.doRaw(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if cType := res.Header.Get("Content-Type"); !strings.HasPrefix(cType, "text/html") {
		bb, _ := ioutil.ReadAll(res.Body)
		log.WithFields(log.Fields{
			"url":          req.URL,
			"content-type": cType,
			"body":         string(bb),
		}).Error("Unexpected content-type")
		return nil, fmt.Errorf("response content-type %s not expected", cType)
	}

	return html.Parse(res.Body)
}

func (c *Client) get(uri string) (*html.Node, error) {
	req, err := c.newRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	return c.do(req)
}

func (c *Client) post(uri string, values url.Values) (*html.Node, error) {
	log.WithField("values", values).Debug("POST parameters")
	req, err := c.newRequest(http.MethodPost, uri, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	return c.do(req)
}

// GetUsername makes a request to FA to verify that the provided cookies result in being logged in
// by finding our username. Returns ErrNotLoggedIn if username could not be found.
func (c *Client) GetUsername() (string, error) {
	root, err := c.get("/search")
	if err != nil {
		return "", err
	}

	h := &myUsernameHandler{}
	p := subtreeProcessor{
		tagHandlers: []tagHandler{
			h,
		},
	}
	p.processNode(root)

	if h.username == "" {
		return "", ErrNotLoggedIn
	}
	return h.username, nil
}

type myUsernameHandler struct {
	username string
}

func (*myUsernameHandler) matches(n *html.Node) bool {
	return checkNodeTagNameAndID(n, "a", "my-username") && n.FirstChild != nil
}

func (h *myUsernameHandler) process(n *html.Node) bool {
	h.username = n.FirstChild.Data
	return false
}
