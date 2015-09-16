/*
 * Copyright 2015 Casey Marshall
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package oostore_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"
	"time"

	gc "gopkg.in/check.v1"
	"gopkg.in/macaroon.v1"

	"github.com/cmars/oostore"
)

func Test(t *testing.T) { gc.TestingT(t) }

type serviceSuite struct {
	store   oostore.Storage
	service *oostore.Service
	server  *httptest.Server
}

var _ = gc.Suite(&serviceSuite{})

func (s *serviceSuite) SetUpTest(c *gc.C) {
	var err error
	s.store = oostore.NewMemStorage()
	s.service, err = oostore.NewService(oostore.ServiceConfig{
		ObjectStore: s.store,
	})
	c.Assert(err, gc.IsNil)
	s.server = httptest.NewServer(s.service)
}

func (s *serviceSuite) TearDownTest(c *gc.C) {
	if s.server != nil {
		s.server.Close()
	}
}

func (s *serviceSuite) TestObjectNoAuth(c *gc.C) {
	cl := &http.Client{}
	resp, err := cl.Post(s.server.URL+"/nope", "application/json", bytes.NewBuffer(nil))
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusForbidden)
}

func (s *serviceSuite) TestObjectBadAuth(c *gc.C) {
	cl := &http.Client{}
	m, err := macaroon.New(nil, "", "here")
	c.Assert(err, gc.IsNil)
	var mjson bytes.Buffer
	err = json.NewEncoder(&mjson).Encode(macaroon.Slice{m})
	c.Assert(err, gc.IsNil)
	resp, err := cl.Post(s.server.URL+"/nope", "application/json", bytes.NewBuffer(mjson.Bytes()))
	c.Assert(err, gc.IsNil)
	c.Assert(resp.StatusCode, gc.Equals, http.StatusForbidden)
}

func (s *serviceSuite) TestPost(c *gc.C) {
	cl := &http.Client{}
	resp, err := cl.Post(s.server.URL, "something/something", bytes.NewBufferString("hunter2"))
	c.Assert(err, gc.IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)

	var ms macaroon.Slice
	err = json.NewDecoder(resp.Body).Decode(&ms)
	c.Assert(err, gc.IsNil)
	c.Assert(ms, gc.HasLen, 1)
}

func (s *serviceSuite) TestPostRetrieve(c *gc.C) {
	cl := &http.Client{}
	resp, err := cl.Post(s.server.URL, "something/something", bytes.NewBufferString("hunter2"))
	c.Assert(err, gc.IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)

	loc := resp.Header.Get("Location")
	c.Assert(loc, gc.Not(gc.Equals), "", gc.Commentf("empty location"))

	var mjson bytes.Buffer
	_, err = io.Copy(&mjson, resp.Body)
	c.Assert(err, gc.IsNil)

	var ms macaroon.Slice
	err = json.NewDecoder(bytes.NewBuffer(mjson.Bytes())).Decode(&ms)
	c.Assert(err, gc.IsNil)
	c.Assert(ms, gc.HasLen, 1)
	c.Assert(ms[0].Caveats(), gc.HasLen, 1)
	c.Assert(ms[0].Caveats()[0].Id, gc.Equals, fmt.Sprintf("object %s", path.Base(loc)))

	resp, err = cl.Post(s.server.URL+loc, "application/json", bytes.NewBuffer(mjson.Bytes()))
	c.Assert(err, gc.IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)
	c.Assert(resp.Header.Get("Content-Type"), gc.Equals, "something/something")

	var contents bytes.Buffer
	_, err = io.Copy(&contents, resp.Body)
	c.Assert(err, gc.IsNil)
	c.Assert(string(contents.Bytes()), gc.Equals, "hunter2")
}

func (s *serviceSuite) TestPostDeleteGone(c *gc.C) {
	cl := &http.Client{}
	resp, err := cl.Post(s.server.URL, "something/something", bytes.NewBufferString("hunter2"))
	c.Assert(err, gc.IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)

	loc := resp.Header.Get("Location")
	c.Assert(loc, gc.Not(gc.Equals), "", gc.Commentf("empty location"))

	var mjson bytes.Buffer
	_, err = io.Copy(&mjson, resp.Body)
	c.Assert(err, gc.IsNil)

	req, err := http.NewRequest("DELETE", s.server.URL+loc, bytes.NewBuffer(mjson.Bytes()))
	c.Assert(err, gc.IsNil)
	resp, err = cl.Do(req)
	c.Assert(err, gc.IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, gc.Equals, http.StatusNoContent)

	resp, err = cl.Post(s.server.URL+loc, "application/json", bytes.NewBuffer(mjson.Bytes()))
	c.Assert(err, gc.IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, gc.Equals, http.StatusNotFound)
}

func (s *serviceSuite) TestTimeBefore(c *gc.C) {
	cl := &http.Client{}
	resp, err := cl.Post(s.server.URL, "something/something", bytes.NewBufferString("hunter2"))
	c.Assert(err, gc.IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)

	loc := resp.Header.Get("Location")
	c.Assert(loc, gc.Not(gc.Equals), "", gc.Commentf("empty location"))

	var mjson bytes.Buffer
	_, err = io.Copy(&mjson, resp.Body)
	c.Assert(err, gc.IsNil)

	for _, testCase := range []struct {
		t          time.Time
		statusCode int
	}{{
		time.Now().UTC().Add(time.Hour),
		http.StatusOK,
	}, {
		time.Now().UTC().Add(-time.Hour),
		http.StatusForbidden,
	}} {

		var ms macaroon.Slice
		err = json.NewDecoder(bytes.NewBuffer(mjson.Bytes())).Decode(&ms)
		c.Assert(err, gc.IsNil)
		c.Assert(ms, gc.HasLen, 1)
		err = ms[0].AddFirstPartyCaveat(fmt.Sprintf("time-before %s", testCase.t.Format(time.RFC3339)))
		c.Assert(err, gc.IsNil)

		var mjsonCav bytes.Buffer
		err = json.NewEncoder(&mjsonCav).Encode(ms)
		c.Assert(err, gc.IsNil)

		resp, err = cl.Post(s.server.URL+loc, "application/json", bytes.NewBuffer(mjsonCav.Bytes()))
		c.Assert(err, gc.IsNil)
		defer resp.Body.Close()
		c.Assert(resp.StatusCode, gc.Equals, testCase.statusCode,
			gc.Commentf("time-before %v at %v", testCase.t, time.Now().UTC()))
	}
}

func (s *serviceSuite) TestClientIPAddr(c *gc.C) {
	cl := &http.Client{}
	resp, err := cl.Post(s.server.URL, "something/something", bytes.NewBufferString("hunter2"))
	c.Assert(err, gc.IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK)

	loc := resp.Header.Get("Location")
	c.Assert(loc, gc.Not(gc.Equals), "", gc.Commentf("empty location"))

	var mjson bytes.Buffer
	_, err = io.Copy(&mjson, resp.Body)
	c.Assert(err, gc.IsNil)

	for _, testCase := range []struct {
		clientIP   string
		statusCode int
	}{{
		"127.0.0.1",
		http.StatusOK,
	}, {
		"1.2.3.4",
		http.StatusForbidden,
	}, {
		"bad-address",
		http.StatusForbidden,
	}} {

		var ms macaroon.Slice
		err = json.NewDecoder(bytes.NewBuffer(mjson.Bytes())).Decode(&ms)
		c.Assert(err, gc.IsNil)
		c.Assert(ms, gc.HasLen, 1)
		err = ms[0].AddFirstPartyCaveat("client-ip-addr " + testCase.clientIP)
		c.Assert(err, gc.IsNil)

		var mjsonCav bytes.Buffer
		err = json.NewEncoder(&mjsonCav).Encode(ms)
		c.Assert(err, gc.IsNil)

		resp, err = cl.Post(s.server.URL+loc, "application/json", bytes.NewBuffer(mjsonCav.Bytes()))
		c.Assert(err, gc.IsNil)
		defer resp.Body.Close()
		c.Assert(resp.StatusCode, gc.Equals, testCase.statusCode,
			gc.Commentf("client-ip-addr %s", testCase.clientIP))
	}
}
