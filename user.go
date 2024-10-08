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
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

type (
	User struct {
		c    *Client
		name string
	}

	faSubmission struct {
		// user profile pages only provide the rating in the JSON
		Rating string `json:"icon_rating"`
		Title  string `json:"title"`
		User   string `json:"username"`
	}

	SubmissionType int
)

const (
	SubmissionTypeGallery SubmissionType = iota
	SubmissionTypeScraps
)

var (
	journalRegexp        = regexp.MustCompile(`^/journal/(\d+)/$`)
	galleryDataRegexp    = regexp.MustCompile(`var descriptions = (.*}});`)
	submissionDataRegexp = regexp.MustCompile(`var submission_data = (.*}});`)
)

func (st SubmissionType) URI() string {
	switch st {
	case SubmissionTypeGallery:
		return "gallery"
	case SubmissionTypeScraps:
		return "scraps"
	default:
		panic("unknown SubmissionType")
	}
}

func (c *Client) NewUser(name string) *User {
	return &User{
		c:    c,
		name: name,
	}
}

// GetRecent retrieves the user's most recent submissions and journal.
// It obtains the data from the user's profile page, so the number of results is limited.
func (u *User) GetRecent() ([]*Submission, []*Journal, error) {
	log.WithField("user", u).Debug("Retrieving recent submissions and journals")
	var subs []*Submission
	var journs []*Journal

	root, err := u.c.get("/user/" + u.name)
	if err != nil {
		return subs, journs, err
	}

	submissions := &submissionSectionHandler{
		c:         u.c,
		sectionID: "gallery-latest-submissions",
	}
	journals := &journalHandler{
		c: u.c,
	}
	scripts := &scriptHandler{
		regexp: submissionDataRegexp,
	}

	rp := &subtreeProcessor{
		tagHandlers: []tagHandler{
			submissions,
			journals,
			scripts,
		},
	}
	rp.processNode(root)

	subs = u.attachSubmissionData(submissions.subs, scripts.data)
	journs = u.attachJournalData(journals.js)

	return subs, journs, nil
}

// GetJournals retrieves the specified page of the user's journal. Page numbering starts at 1.
func (u *User) GetJournals(page uint) ([]*Journal, error) {
	if page == 0 {
		page = 1
	}
	log.WithField("user", u).WithField("page", page).Debug("Retrieving journals")

	var journs []*Journal
	root, err := u.c.get(fmt.Sprintf("/journals/%s/%d/", u.name, page))
	if err != nil {
		return journs, err
	}

	journals := &journalHandler{
		c: u.c,
	}
	rp := &subtreeProcessor{
		tagHandlers: []tagHandler{
			journals,
		},
	}
	rp.processNode(root)
	journs = u.attachJournalData(journals.js)

	return journs, nil
}

// GetSubmissions retrieves the specified page of the user's gallery. Page numbering starts at 1.
// NOTE: Rating information is currently not provided on the submissions.
func (u *User) GetSubmissions(page uint) ([]*Submission, error) {
	return u.GetGallery(SubmissionTypeGallery, page)
}

// GetGallery retrieves the specified page of the user's gallery of the specified type. Page numbering starts at 1.
// NOTE: Rating information is currently not provided on the submissions.
func (u *User) GetGallery(st SubmissionType, page uint) ([]*Submission, error) {
	if page == 0 {
		page = 1
	}
	log.WithField("user", u).WithField("page", page).Debugf("Retrieving submissions %s", st.URI())

	var subs []*Submission
	root, err := u.c.get(fmt.Sprintf("/%s/%s/%d/", st.URI(), u.name, page))
	if err != nil {
		return subs, err
	}

	submissions := &submissionSectionHandler{
		c:         u.c,
		sectionID: "gallery-gallery",
	}
	scripts := &scriptHandler{
		regexp: galleryDataRegexp,
	}
	rp := &subtreeProcessor{
		tagHandlers: []tagHandler{
			submissions,
			scripts,
		},
	}
	rp.processNode(root)

	subs = u.attachSubmissionData(submissions.subs, scripts.data)
	return subs, nil
}

func (u *User) attachSubmissionData(subs []*Submission, data map[int64]faSubmission) []*Submission {
	for i := range subs {
		id := subs[i].ID
		if data[id].Rating != "" {
			subs[i].Rating = Rating(strings.Replace(data[id].Rating, "r-", "", 1))
		}
		subs[i].Title = data[id].Title
		subs[i].User = data[id].User
	}

	return subs
}

func (u *User) attachJournalData(js []*Journal) []*Journal {
	for i := range js {
		js[i].c = u.c
		js[i].User = u.name
	}
	return js
}

type scriptHandler struct {
	data   map[int64]faSubmission
	regexp *regexp.Regexp
}

func (s *scriptHandler) matches(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Data == "script" && n.FirstChild != nil &&
		s.regexp.MatchString(n.FirstChild.Data)
}

func (s *scriptHandler) process(n *html.Node) bool {
	raw := s.regexp.FindStringSubmatch(n.FirstChild.Data)[1]
	data := make(map[int64]faSubmission)
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		log.WithError(err).Error("Unable to unmarshal submission JSON data")
	}
	s.data = data
	return false
}

// submissionSectionHandler finds and extracts the recent submissionHandler section
type submissionSectionHandler struct {
	c         *Client
	sectionID string
	subs      []*Submission
}

func (sh *submissionSectionHandler) matches(n *html.Node) bool {
	return checkNodeTagNameAndID(n, "section", sh.sectionID)
}

func (sh *submissionSectionHandler) process(n *html.Node) bool {
	s := &submissionHandler{
		c: sh.c,
	}
	p := subtreeProcessor{
		tagHandlers: []tagHandler{
			s,
		},
	}
	p.processNode(n)

	sh.subs = s.subs
	return false
}

// submissionHandler finds and extracts each submission
type submissionHandler struct {
	c    *Client
	subs []*Submission
}

func (*submissionHandler) matches(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Data == "figure"
}

func (s *submissionHandler) process(n *html.Node) bool {
	si := &submissionImageHandler{}
	p := subtreeProcessor{
		tagHandlers: []tagHandler{
			si,
		},
	}
	p.processNode(n)
	s.subs = append(s.subs, &Submission{
		c:  s.c,
		ID: parseSubmissionID(findAttribute(n.Attr, "id")),
		// gallery pages only provide the rating as a class attribute
		Rating:     Rating(strings.Replace(strings.Split(findAttribute(n.Attr, "class"), " ")[0], "r-", "", 1)),
		PreviewURL: si.url,
	})
	return false
}

type submissionImageHandler struct {
	url string
}

func (*submissionImageHandler) matches(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Data == "img"
}

func (si *submissionImageHandler) process(n *html.Node) bool {
	si.url = "https:" + findAttribute(n.Attr, "src")
	return false
}

// journalHandler finds and retrieves journal links
type journalHandler struct {
	c  *Client
	js []*Journal
}

func (j *journalHandler) matches(n *html.Node) bool {
	if n.Type == html.ElementNode && n.Data == "a" {
		href := findAttribute(n.Attr, "href")
		if journalRegexp.MatchString(href) {
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

func (j *journalHandler) process(n *html.Node) bool {
	href := findAttribute(n.Attr, "href")
	id := journalRegexp.FindStringSubmatch(href)[1]
	j.js = append(j.js, &Journal{
		ID:    parseSubmissionID(id),
		Title: n.FirstChild.Data,
	})
	return false
}

func (j *journalHandler) String() string {
	return fmt.Sprintf("%+v", j.js)
}
