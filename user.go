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
	"strings"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

type User struct {
	c    *Client
	name string
}

type Submission struct {
	c  *Client
	id string
}

type Journal struct {
	c  *Client
	id string
}

func (c *Client) NewUser(name string) *User {
	return &User{
		c:    c,
		name: name,
	}
}

func (s Submission) String() string {
	return s.id
}

func (j Journal) String() string {
	return j.id
}

func (c *Client) submissionsFromIDs(ids []string) []*Submission {
	subs := make([]*Submission, len(ids))
	for i, id := range ids {
		subs[i] = c.newSubmission(id)
	}
	return subs
}

func (c *Client) journalsFromIDs(ids map[string]bool) []*Journal {
	journs := make([]*Journal, len(ids))
	i := 0
	for id := range ids {
		journs[i] = c.newJournal(id)
		i++
	}
	return journs
}

func (c *Client) newSubmission(id string) *Submission {
	return &Submission{
		c:  c,
		id: strings.Replace(id, "sid-", "", 1),
	}
}

func (c *Client) newJournal(id string) *Journal {
	return &Journal{
		c:  c,
		id: id,
	}
}

func (u *User) GetRecent() ([]*Submission, []*Journal, error) {
	log.WithField("user", u).Debug("Retrieving recent submissionHandler and journals")
	var subs []*Submission
	var journs []*Journal

	root, err := u.c.get("/user/" + u.name)
	if err != nil {
		return subs, journs, err
	}

	submissions := &submissionSectionHandler{}
	journals := &journalHandler{
		client: u.c,
		ids:    make(map[string]bool),
	}

	rp := &subtreeProcessor{
		tagHandlers: []tagHandler{
			submissions,
			journals,
		},
	}
	rp.processNode(root)

	subs = u.c.submissionsFromIDs(submissions.ids)
	journs = u.c.journalsFromIDs(journals.ids)

	return subs, journs, nil
}

// submissionSectionHandler finds and extracts the recent submissionHandler section
type submissionSectionHandler struct {
	ids []string
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
	return false
}

// submissionHandler finds and extracts each submission
type submissionHandler struct {
	ids []string
}

func (*submissionHandler) Matches(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Data == "figure"
}

func (s *submissionHandler) Process(n *html.Node) bool {
	s.ids = append(s.ids, findAttribute(n.Attr, "id"))
	return false
}

// journalHandler finds and retrieves journal links
type journalHandler struct {
	client *Client
	ids    map[string]bool
}

func (j *journalHandler) Matches(n *html.Node) bool {
	if n.Type == html.ElementNode && n.Data == "a" {
		href := findAttribute(n.Attr, "href")
		return j.client.journalRegexp.MatchString(href)
	}
	return false
}

func (j *journalHandler) Process(n *html.Node) bool {
	href := findAttribute(n.Attr, "href")
	id := j.client.journalRegexp.FindStringSubmatch(href)[1]
	j.ids[id] = true
	return false
}

func (j *journalHandler) String() string {
	ids := make([]string, len(j.ids))
	i := 0
	for id := range j.ids {
		ids[i] = id
	}
	return fmt.Sprintf("%+v", ids)
}
