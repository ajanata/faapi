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
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

type User struct {
	c    *Client
	name string
}

type Submission struct {
	c     *Client
	id    string
	title string
}

type faSubmission struct {
	Rating   string `json:"icon_rating"`
	Title    string `json:"title"`
	Username string `json:"username"`
}

type Journal struct {
	c     *Client
	id    string
	title string
}

func (c *Client) NewUser(name string) *User {
	return &User{
		c:    c,
		name: name,
	}
}

func (s Submission) String() string {
	return fmt.Sprintf("%s (%s)", s.title, s.id)
}

func (j Journal) String() string {
	return fmt.Sprintf("%s (%s)", j.title, j.id)
}

func (c *Client) submissionsFromData(ids []string, data map[string]faSubmission) []*Submission {
	subs := make([]*Submission, len(ids))
	for i, id := range ids {
		subs[i] = c.newSubmission(id, data[id].Title)
	}
	return subs
}

func (c *Client) journalsFromData(ids, titles []string) []*Journal {
	journs := make([]*Journal, len(ids))
	for i, id := range ids {
		journs[i] = c.newJournal(id, titles[i])
	}
	return journs
}

func (c *Client) newSubmission(id, title string) *Submission {
	return &Submission{
		c:     c,
		id:    strings.Replace(id, "sid-", "", 1),
		title: title,
	}
}

func (c *Client) newJournal(id, title string) *Journal {
	return &Journal{
		c:     c,
		id:    id,
		title: title,
	}
}

// GetRecent retrieves the user's most recent submissions and journals.
func (u *User) GetRecent() ([]*Submission, []*Journal, error) {
	log.WithField("user", u).Debug("Retrieving recent submissions and journals")
	var subs []*Submission
	var journs []*Journal

	root, err := u.c.get("/user/" + u.name)
	if err != nil {
		return subs, journs, err
	}

	submissions := &submissionSectionHandler{}
	journals := &journalHandler{
		client: u.c,
	}
	scripts := &scriptHandler{
		client: u.c,
	}

	rp := &subtreeProcessor{
		tagHandlers: []tagHandler{
			submissions,
			journals,
			scripts,
		},
	}
	rp.processNode(root)

	subs = u.c.submissionsFromData(submissions.ids, scripts.data)
	journs = u.c.journalsFromData(journals.ids, journals.titles)

	return subs, journs, nil
}

type scriptHandler struct {
	client *Client
	data   map[string]faSubmission
}

func (s *scriptHandler) Matches(n *html.Node) (matches bool) {
	return n.Type == html.ElementNode && n.Data == "script" && n.FirstChild != nil &&
		s.client.submissionDataRegexp.MatchString(n.FirstChild.Data)
}

func (s *scriptHandler) Process(n *html.Node) (recurseChildren bool) {
	raw := s.client.submissionDataRegexp.FindStringSubmatch(n.FirstChild.Data)[1]
	data := make(map[string]faSubmission)
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		log.WithError(err).Error("Unable to unmarshal submission JSON data")
	}
	s.data = data
	return false
}

// submissionSectionHandler finds and extracts the recent submissionHandler section
type submissionSectionHandler struct {
	ids    []string
	titles []string
}

func (*submissionSectionHandler) Matches(n *html.Node) bool {
	return checkNodeTagNameAndID(n, "section", "gallery-latest-submissions")
}

func (sh *submissionSectionHandler) Process(n *html.Node) bool {
	s := &submissionHandler{}
	p := subtreeProcessor{
		tagHandlers: []tagHandler{
			s,
		},
	}
	p.processNode(n)

	sh.ids = s.ids
	sh.titles = s.titles
	return false
}

// submissionHandler finds and extracts each submission
type submissionHandler struct {
	ids    []string
	titles []string
}

func (*submissionHandler) Matches(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Data == "figure"
}

func (s *submissionHandler) Process(n *html.Node) bool {
	s.ids = append(s.ids, strings.Replace(findAttribute(n.Attr, "id"), "sid-", "", 1))
	s.titles = append(s.titles, "")
	return false
}

// journalHandler finds and retrieves journal links
type journalHandler struct {
	client *Client
	ids    []string
	titles []string
}

func (j *journalHandler) Matches(n *html.Node) bool {
	if n.Type == html.ElementNode && n.Data == "a" {
		href := findAttribute(n.Attr, "href")
		if j.client.journalRegexp.MatchString(href) {
			linkText := n.FirstChild
			// Exclude other links that lead to the journal that don't include its title.
			if linkText != nil && linkText.Type == html.TextNode {
				return !strings.HasPrefix(linkText.Data, "Comments ") && linkText.Data != "Read more..."
			}
		}
		return false
	}
	return false
}

func (j *journalHandler) Process(n *html.Node) bool {
	href := findAttribute(n.Attr, "href")
	id := j.client.journalRegexp.FindStringSubmatch(href)[1]
	j.ids = append(j.ids, id)
	j.titles = append(j.titles, n.FirstChild.Data)
	return false
}

func (j *journalHandler) String() string {
	return fmt.Sprintf("%+v", j.ids)
}
