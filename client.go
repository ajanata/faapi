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
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

// Client is a FurAffinity client.
type Client struct {
	http                 http.Client
	config               Config
	journalRegexp        *regexp.Regexp
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

	return &Client{
		http: http.Client{
			Transport: &tr,
		},
		config:               config,
		journalRegexp:        journalRegexp,
		submissionDataRegexp: submissionDataRegexp,
	}, nil
}

func (c *Client) newRequest(method, uri string) (*http.Request, error) {
	log.WithField("uri", uri).Debug("Creating new request")
	req, err := http.NewRequest(method, "https://furaffinity.net"+uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", c.config.UserAgent)
	return req, nil
}

func (c *Client) get(uri string) (*html.Node, error) {
	req, err := c.newRequest(http.MethodGet, uri)
	if err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		bb, _ := ioutil.ReadAll(res.Body)
		log.WithFields(log.Fields{
			"uri":  uri,
			"code": res.StatusCode,
			"body": string(bb),
		}).Error("Unexpected HTTP response code")
		return nil, fmt.Errorf("HTTP response %d not expected", res.StatusCode)
	}

	return html.Parse(res.Body)
}
