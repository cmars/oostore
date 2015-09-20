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

package postgres_test

import (
	gc "gopkg.in/check.v1"

	"github.com/cmars/oostore"
	"github.com/cmars/oostore/postgres"
)

var _ = gc.Suite(&objectSuite{})

type objectSuite struct {
	postgresSuite
	storage oostore.Storage
}

func (s *objectSuite) SetUpTest(c *gc.C) {
	s.postgresSuite.SetUpTest(c)
	var err error
	s.storage, err = postgres.NewObjectStorage(s.db)
	c.Assert(err, gc.IsNil)
}

func (s *objectSuite) TearDownTest(c *gc.C) {
	s.postgresSuite.TearDownTest(c)
}

func (s *objectSuite) TestCRUD(c *gc.C) {
	// Put some records in.
	c.Assert(s.storage.Put("baz", []byte("quux"), "quux-ish"), gc.IsNil)
	c.Assert(s.storage.Put("foo", []byte("bar"), "bar-ish"), gc.IsNil)
	// Should be able to get it back out.
	content, contentType, err := s.storage.Get("foo")
	c.Assert(err, gc.IsNil)
	c.Assert(content, gc.DeepEquals, []byte("bar"))
	c.Assert(contentType, gc.Equals, "bar-ish")
	// Delete the record.
	c.Assert(s.storage.Delete("foo"), gc.IsNil)
	// Get records that don't exist, should give "not found" error.
	for _, id := range []string{"foo", "never-seen-it"} {
		comment := gc.Commentf("id %q", id)
		_, _, err = s.storage.Get(id)
		c.Check(err, gc.NotNil, comment)
		c.Check(err, gc.Equals, oostore.ErrNotFound, comment)
	}
	// Delete records that don't exist.
	for _, id := range []string{"foo", "never-seen-it"} {
		comment := gc.Commentf("id %q", id)
		c.Check(s.storage.Delete(id), gc.Equals, oostore.ErrNotFound, comment)
	}
}

func (s *objectSuite) TestPrimaryKey(c *gc.C) {
	// Put some records in, with some duplicates. Exercises rollbacks.
	c.Assert(s.storage.Put("foo", []byte("bar"), "bar-ish"), gc.IsNil)
	c.Assert(s.storage.Put("foo", []byte("bar"), "bar-ish"), gc.NotNil)
	c.Assert(s.storage.Put("foo", []byte("bar"), "bar-ish"), gc.NotNil)
	_, _, err := s.storage.Get("nope")
	c.Assert(err, gc.NotNil)
	c.Assert(s.storage.Put("baz", []byte("quux"), "quux-ish"), gc.IsNil)
	c.Assert(s.storage.Put("baz", []byte("quux"), "quux-ish"), gc.NotNil)
	c.Assert(s.storage.Put("a", []byte("b"), ""), gc.IsNil)
	c.Assert(s.storage.Put("empty", []byte(""), "nothing-ness"), gc.IsNil)
	for i, testCase := range []struct {
		id, contents, contentType string
	}{{"foo", "bar", "bar-ish"}, {"baz", "quux", "quux-ish"}, {"a", "b", ""}, {"empty", "", "nothing-ness"}} {
		comment := gc.Commentf("test#%d expect contents %#v", i, testCase)
		content, contentType, err := s.storage.Get(testCase.id)
		c.Assert(err, gc.IsNil, comment)
		c.Assert(content, gc.DeepEquals, []byte(testCase.contents), comment)
		c.Assert(contentType, gc.Equals, testCase.contentType, comment)
	}
	for i, id := range []string{"foo", "baz", "a", "empty"} {
		comment := gc.Commentf("test#%d expect unique %s", i, id)
		var count int
		row := s.db.QueryRow("SELECT COUNT(1) FROM object WHERE id = $1", id)
		c.Assert(row.Scan(&count), gc.IsNil, comment)
		c.Assert(count, gc.Equals, 1, comment)
	}
}
