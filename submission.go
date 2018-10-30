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
	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Submission is an artwork submission.
type Submission struct {
	c            *Client
	ID           int64
	PreviewURL   string
	Rating       Rating
	Title        string
	User         string
	previewImage *[]byte
}

// Rating is the decency rating of a submission.
type Rating string

// Rating values
const (
	RatingGeneral Rating = "general"
	RatingMature  Rating = "mature"
	RatingAdult   Rating = "adult"
)

const (
	previewURLFormat = "https://t.facdn.net/%s@400-%s.%s"
)

var (
	previewSizeRegexp = regexp.MustCompile(`^https://t.facdn.net/(\d+)@(\d+)-(\d+)\.([a-zA-Z]+)$`)
)

func (s Submission) String() string {
	return fmt.Sprintf("%s %s by %s (%s, %d)", s.PreviewURL, s.Title, s.User, s.Rating, s.ID)
}

func (s *Submission) PreviewImage() ([]byte, error) {
	if s.previewImage != nil {
		return *s.previewImage, nil
	}
	logger := log.WithField("submission", s)

	// try to get the largest preview available
	parts := previewSizeRegexp.FindStringSubmatch(s.PreviewURL)
	if len(parts) == 5 {
		// don't bother for preview URLs already at the large size
		if parts[2] != "400" {
			url := fmt.Sprintf(previewURLFormat, parts[1], parts[3], parts[4])
			bb, err := s.getImage(url)
			if err != nil {
				logger.WithError(err).Warn("Unable to retrieve large-size preview; falling back to provided size")
			} else {
				s.previewImage = &bb
				return bb, nil
			}
		}
	} else {
		logger.Warn("Regexp failed to parse preview URL")
	}

	bb, err := s.getImage(s.PreviewURL)
	if err != nil {
		return nil, err
	}
	s.previewImage = &bb
	return bb, nil
}

func (s *Submission) getImage(url string) ([]byte, error) {
	req, err := s.c.newRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	res, err := s.c.doRaw(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	bb, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return bb, nil
}

func parseSubmissionID(str string) int64 {
	id, err := strconv.ParseInt(strings.Replace(str, "sid-", "", 1), 10, 64)
	// if this ever happens, everything will be completely broken, so returning 0 is... fine?
	if err != nil {
		log.WithError(err).Error("Unable to parse submission ID")
	}
	return id
}
