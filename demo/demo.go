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

package main

import (
	"time"

	"github.com/ajanata/faapi"
	log "github.com/sirupsen/logrus"
)

// Demo program for faapi.
func main() {
	log.SetLevel(log.DebugLevel)

	c, err := faapi.New(faapi.Config{
		Proxy:     "socks5://127.0.0.1:18080",
		RateLimit: time.Second,
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/69.0.3497.100 Safari/537.36",
	})
	if err != nil {
		panic(err)
	}
	defer c.Close()

	username, err := c.GetUsername()
	switch err {
	case faapi.ErrNotLoggedIn:
		log.Warn("Could not determine username")
	case nil:
		log.WithField("username", username).Info("Successfully logged in")
	default:
		panic(err)
	}

	subs, journs, err := c.NewUser("dragoneer").GetRecent()
	if err != nil {
		panic(err)
	}
	log.WithFields(log.Fields{
		"submissions": subs,
		"journals":    journs,
	}).Info("user recents")

	subs, err = c.NewSearch("@keywords ych").GetPage(1)
	if err != nil {
		panic(err)
	}
	log.WithField("results", subs).Info("search results")

	bb, err := subs[0].PreviewImage()
	log.WithError(err).WithField("bytes", bb).Info("first result bytes")
}
