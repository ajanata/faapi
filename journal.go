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

	"golang.org/x/net/html"
)

// Journal is a journal entry.
type Journal struct {
	c       *Client
	ID      int64
	Title   string
	User    string
	content *string
}

func (j *Journal) String() string {
	return fmt.Sprintf("%s (%s)", j.Title, j.ID)
}

func (j *Journal) URL() string {
	return fmt.Sprintf("https://www.furaffinity.net/journal/%d/", j.ID)
}

func (j *Journal) Content() (string, error) {
	if j.content != nil {
		return *j.content, nil
	}

	root, err := j.c.get(j.URL())
	if err != nil {
		return "", err
	}

	jch := &journalContentHandler{}
	jdh := &journalDateHandler{}
	rp := &subtreeProcessor{
		tagHandlers: []tagHandler{
			jch,
			jdh,
		},
	}
	rp.processNode(root)

	s := jdh.text + "\n\n" + jch.text
	j.content = &s
	return s, nil
}

type journalContentHandler struct {
	text string
}

func (*journalContentHandler) matches(n *html.Node) bool {
	return checkNodeTagNameAndClass(n, "div", "journal-body")
}

func (dh *journalContentHandler) process(n *html.Node) bool {
	s := strings.ReplaceAll(getText(n), "  ", " ")
	s = strings.ReplaceAll(s, " ", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.Trim(s, " \t \r\n")
	dh.text = s
	return true
}

type journalDateHandler struct {
	text string
}

func (*journalDateHandler) matches(n *html.Node) bool {
	return checkNodeTagNameAndClass(n, "span", "popup_date")
}

func (dh *journalDateHandler) process(n *html.Node) bool {
	dh.text = n.FirstChild.Data
	return true
}
