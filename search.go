/*
 *
 * Copyright (c) 2018-2019, Andy Janata
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
	"net/url"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

type Search struct {
	c     *Client
	query string
}

// NewSearch creates a new search for the given query.
func (c *Client) NewSearch(query string) *Search {
	return &Search{
		c:     c,
		query: query,
	}
}

// GetPage returns the search results on the given page. The page numbering starts at 1.
func (s *Search) GetPage(page int) ([]*Submission, error) {
	var subs []*Submission
	log.WithFields(log.Fields{
		"query": s.query,
		"page":  page,
	}).Debug("Performing search")

	params := url.Values{}
	params.Set("q", s.query)
	params.Set("page", strconv.Itoa(page))
	params.Set("perpage", "72")
	params.Set("order-by", "date")
	params.Set("order-direction", "desc")
	params.Set("do_search", "Search")
	params.Set("range", "all")
	params.Set("rating-general", "on")
	params.Set("rating-mature", "on")
	params.Set("rating-adult", "on")
	params.Set("type-art", "on")
	params.Set("type-flash", "on")
	params.Set("type-photo", "on")
	params.Set("type-music", "on")
	params.Set("type-story", "on")
	params.Set("type-poetry", "on")
	params.Set("mode", "extended")

	root, err := s.c.post("/search/", params)
	if err != nil {
		return subs, err
	}

	srh := &searchResultsHandler{}
	p := subtreeProcessor{
		tagHandlers: []tagHandler{
			srh,
		},
	}
	p.processNode(root)

	subs = srh.results
	for i := range subs {
		subs[i].c = s.c
	}

	return subs, nil
}

type searchResultsHandler struct {
	results []*Submission
}

func (*searchResultsHandler) matches(n *html.Node) bool {
	return checkNodeTagNameAndID(n, "section", "gallery-search-results")
}

func (srh *searchResultsHandler) process(n *html.Node) bool {
	srsh := &searchResultsSectionHandler{}
	p := subtreeProcessor{
		tagHandlers: []tagHandler{
			srsh,
		},
	}
	p.processNode(n)
	srh.results = srsh.results
	return false
}

type searchResultsSectionHandler struct {
	results []*Submission
}

func (*searchResultsSectionHandler) matches(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Data == "figure"
}

func (srsh *searchResultsSectionHandler) process(n *html.Node) bool {
	classes := strings.Split(findAttribute(n.Attr, "class"), " ")
	var rating string
	for _, class := range classes {
		if strings.HasPrefix(class, "r-") {
			rating = class
			break
		}
	}
	ssh := &searchSubmissionHandler{}
	ssph := &searchSubmissionPreviewHandler{}
	p := subtreeProcessor{
		tagHandlers: []tagHandler{
			ssh,
			ssph,
		},
	}
	p.processNode(n)

	srsh.results = append(srsh.results, &Submission{
		ID:         parseSubmissionID(findAttribute(n.Attr, "id")),
		Rating:     Rating(strings.Replace(rating, "r-", "", 1)),
		PreviewURL: ssph.url,
		Title:      ssh.title,
		User:       ssh.user,
	})
	return false
}

type searchSubmissionHandler struct {
	title string
	user  string
}

func (*searchSubmissionHandler) matches(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Data == "figcaption"
}

func (ssh *searchSubmissionHandler) process(n *html.Node) bool {
	ssch := &searchSubmissionCaptionHandler{}
	p := subtreeProcessor{
		tagHandlers: []tagHandler{
			ssch,
		},
	}
	p.processNode(n)
	ssh.title = ssch.title
	ssh.user = ssch.user
	return false
}

type searchSubmissionPreviewHandler struct {
	url string
}

func (*searchSubmissionPreviewHandler) matches(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Data == "img"
}

func (ssph *searchSubmissionPreviewHandler) process(n *html.Node) bool {
	ssph.url = "https:" + findAttribute(n.Attr, "src")
	return false
}

type searchSubmissionCaptionHandler struct {
	title string
	user  string
}

func (*searchSubmissionCaptionHandler) matches(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Data == "a"
}

func (ssch *searchSubmissionCaptionHandler) process(n *html.Node) bool {
	href := findAttribute(n.Attr, "href")
	val := findAttribute(n.Attr, "title")
	if strings.HasPrefix(href, "/view/") {
		ssch.title = val
	} else if strings.HasPrefix(href, "/user/") {
		ssch.user = val
	}
	return false
}
