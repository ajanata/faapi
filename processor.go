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
	"strings"

	"golang.org/x/net/html"
)

type subtreeProcessor struct {
	tagHandlers []tagHandler
}

type tagHandler interface {
	matches(n *html.Node) (matches bool)
	process(n *html.Node) (recurseChildren bool)
}

func (rp *subtreeProcessor) processNode(n *html.Node) {
	for _, h := range rp.tagHandlers {
		if h.matches(n) {
			if !h.process(n) {
				return
			}
			break
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		rp.processNode(c)
	}
}

func findAttribute(attrs []html.Attribute, name string) string {
	for _, a := range attrs {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}

func checkNodeTagNameAndID(n *html.Node, name, id string) bool {
	return n.Type == html.ElementNode && n.Data == name && findAttribute(n.Attr, "id") == id
}

func checkNodeTagNameAndClass(n *html.Node, name, class string) bool {
	c := findAttribute(n.Attr, "class")
	hasClass := strings.Contains(c, class)
	return n.Type == html.ElementNode && n.Data == name && hasClass
}

func findChild(n *html.Node, tag string, i int) *html.Node {
	c := 0
	for t := n.FirstChild; t != nil; t = t.NextSibling {
		if t.Data == tag {
			if c == i {
				return t
			}
			c++
		}
	}
	return nil
}

func getText(n *html.Node) string {
	s := ""
	for t := n.FirstChild; t != nil; t = t.NextSibling {
		if t.Type == html.TextNode {
			s = s + strings.Trim(t.Data, " \t \r\n") + "\n"
		}
		if t.FirstChild != nil {
			s = s + getText(t) + "\n"
		}
	}
	return strings.Trim(s, " \t \r\n")
}
