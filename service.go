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

package oostore

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/julienschmidt/httprouter"
	"gopkg.in/errgo.v1"
	"gopkg.in/macaroon-bakery.v1/bakery"
	"gopkg.in/macaroon-bakery.v1/bakery/checkers"
	"gopkg.in/macaroon-bakery.v1/httpbakery"
	"gopkg.in/macaroon.v1"
)

const idLen = 32

// Service provides an HTTP API for opaque object storage.
type Service struct {
	bakery *bakery.Service
	store  Storage
	router *httprouter.Router
}

// ServiceConfig contains the items needed to create a new Service.
type ServiceConfig struct {
	BakeryStore bakery.Storage
	ObjectStore Storage
	Prefix      string
}

// ErrNotFound indicates that the requested content ID was not found.
var ErrNotFound = fmt.Errorf("contents not found")

// Storage defines the interface that is used to associate content with
// unique ID strings.
type Storage interface {
	// Get returns the content bytes and content-type string for the given ID.
	Get(id string) ([]byte, string, error)

	// Put stores new content for the given ID.
	Put(id string, contents []byte, contentType string) error

	// Delete removes content by ID.
	Delete(id string) error
}

// NewService creates a new opaque object storage service.
func NewService(config ServiceConfig) (*Service, error) {
	bakeryService, err := bakery.NewService(bakery.NewServiceParams{
		Store: config.BakeryStore,
	})
	if err != nil {
		return nil, err
	}
	s := &Service{
		bakery: bakeryService,
		store:  config.ObjectStore,
	}

	prefix := "/"
	if config.Prefix != "" {
		prefix = config.Prefix
	}
	s.router = httprouter.New()
	s.router.POST(prefix, s.create)
	s.router.POST(path.Join(prefix, ":object"), s.fetch)
	s.router.DELETE(path.Join(prefix, ":object"), s.del)
	return s, nil
}

// ServeHTTP implements net/http.Handler.
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// httpErrorf writes an HTTP error response. Errors should be noted with a
// message that is useful yet also security-appropriate for a public HTTP
// response to potentially anonymous, unauthenticated clients. Mask errors to
// capture details that will be logged for server-side troubleshooting.
func httpErrorf(w http.ResponseWriter, statusCode int, err error) {
	http.Error(w, err.Error(), statusCode)
	log.Printf("HTTP %d: %s", statusCode, errgo.Details(err))
}

func newID() (string, error) {
	var fail string
	var buf [idLen]byte
	_, err := rand.Read(buf[:])
	if err != nil {
		return fail, err
	}
	return base64.URLEncoding.EncodeToString(buf[:]), nil
}

// create handles the request to store new content, responding with a macaroon
// that can later be used to fetch or delete it.
func (s *Service) create(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		httpErrorf(w, http.StatusInternalServerError, errgo.Notef(err, "failed to read request body"))
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(contents)
	}

	id, err := newID()
	if err != nil {
		httpErrorf(w, http.StatusInternalServerError, errgo.Notef(err, "failed to create an object ID"))
		return
	}
	w.Header().Set("Location", r.URL.Path+id)

	err = s.store.Put(id, contents, contentType)
	if err != nil {
		httpErrorf(w, http.StatusInternalServerError, errgo.Notef(err, "failed to store content"))
		return
	}

	m, err := s.bakery.NewMacaroon("", nil, nil)
	if err != nil {
		httpErrorf(w, http.StatusInternalServerError, errgo.Notef(err, "failed to create macaroon"))
		return
	}
	err = s.bakery.AddCaveat(m, checkers.Caveat{Condition: fmt.Sprintf("object %s", id)})
	if err != nil {
		httpErrorf(w, http.StatusInternalServerError, errgo.Notef(err, "failed to add caveat"))
		return
	}

	ms := macaroon.Slice{m}
	err = json.NewEncoder(w).Encode(ms)
	if err != nil {
		log.Println("failed to write response: %v", err)
	}
}

type authInfo struct {
	object   string
	declared map[string]string
}

type requestInfo struct {
	request   *http.Request
	params    httprouter.Params
	operation string
}

func (s *Service) checkRequest(info requestInfo) (*authInfo, error) {
	var ms macaroon.Slice
	err := json.NewDecoder(info.request.Body).Decode(&ms)
	if err != nil {
		return nil, errgo.Mask(err, errgo.Any)
	}
	declared := checkers.InferDeclared(ms)
	// TODO: assert any declared caveats here
	err = s.bakery.Check(ms, checkers.New(declared, newCheckers(info)))
	if err != nil {
		return nil, errgo.Mask(err, errgo.Any)
	}

	return &authInfo{
		object:   info.params.ByName("object"),
		declared: declared,
	}, nil
}

// fetch handles the request to fetch the content authorized by the given
// macaroon.
func (s *Service) fetch(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	auth, err := s.checkRequest(requestInfo{request: r, params: p, operation: "fetch"})
	if err != nil {
		httpErrorf(w, http.StatusForbidden, err)
		return
	}

	contents, contentType, err := s.store.Get(auth.object)
	if err != nil {
		httpErrorf(w, http.StatusNotFound, errgo.Newf("not found: %q", auth.object))
		return
	}

	w.Header().Set("Content-Type", contentType)
	_, err = w.Write(contents)
	if err != nil {
		log.Println("failed to write contents in response: %v", err)
		return
	}
}

// del handles the request to delete content.
func (s *Service) del(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	auth, err := s.checkRequest(requestInfo{request: r, params: p, operation: "delete"})
	if err != nil {
		httpErrorf(w, http.StatusForbidden, err)
		return
	}

	err = s.store.Delete(auth.object)
	if err == ErrNotFound {
		httpErrorf(w, http.StatusNotFound, errgo.Newf("not found: %q", auth.object))
		return
	} else if err != nil {
		httpErrorf(w, http.StatusInternalServerError, errgo.Notef(err, "failed to delete %q: %v", auth.object))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func newCheckers(info requestInfo) checkers.Checker {
	return checkers.New(
		checkers.TimeBefore,
		httpbakery.Checkers(info.request),
		operationChecker(info.operation),
		requestObjectChecker(info.request, info.params),
	)
}

func requestObjectChecker(r *http.Request, p httprouter.Params) checkers.Checker {
	return checkers.CheckerFunc{
		Condition_: "object",
		Check_: func(_, cav string) error {
			if cav != p.ByName("object") {
				return errgo.New("request does not match")
			}
			return nil
		},
	}
}

func operationChecker(op string) checkers.Checker {
	return checkers.CheckerFunc{
		Condition_: "operation",
		Check_: func(_, cav string) error {
			allowedOps := strings.Split(cav, ",")
			for _, allowedOp := range allowedOps {
				if strings.ToLower(op) == strings.TrimSpace(strings.ToLower(allowedOp)) {
					return nil
				}
			}
			return fmt.Errorf("operation %q not allowed", op)
		},
	}
}
