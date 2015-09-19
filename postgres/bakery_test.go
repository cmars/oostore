package postgres_test

import (
	gc "gopkg.in/check.v1"
	"gopkg.in/macaroon-bakery.v1/bakery"

	"github.com/cmars/oostore/postgres"
)

var _ = gc.Suite(&bakerySuite{})

type bakerySuite struct {
	postgresSuite
	storage bakery.Storage
}

func (s *bakerySuite) SetUpTest(c *gc.C) {
	s.postgresSuite.SetUpTest(c)
	var err error
	s.storage, err = postgres.NewBakeryStorage(s.db)
	c.Assert(err, gc.IsNil)
}

func (s *bakerySuite) TearDownTest(c *gc.C) {
	s.postgresSuite.TearDownTest(c)
}

func (s *bakerySuite) TestCRUD(c *gc.C) {
	// Put some items in.
	c.Assert(s.storage.Put("baz", "quux"), gc.IsNil)
	c.Assert(s.storage.Put("foo", "bar"), gc.IsNil)
	// Should be able to get an item back out.
	item, err := s.storage.Get("foo")
	c.Assert(err, gc.IsNil)
	c.Assert(item, gc.Equals, "bar")
	// Delete a location.
	c.Assert(s.storage.Del("foo"), gc.IsNil)
	// Get locations that don't exist, should give "not found" error.
	for _, loc := range []string{"foo", "never-seen-it"} {
		_, err = s.storage.Get(loc)
		comment := gc.Commentf("location %q", loc)
		c.Assert(err, gc.NotNil, comment)
		c.Assert(err, gc.Equals, bakery.ErrNotFound, comment)
	}
	// Delete locations that don't exist.
	for _, loc := range []string{"foo", "never-seen-it"} {
		comment := gc.Commentf("location %q", loc)
		c.Assert(s.storage.Del("foo"), gc.IsNil, comment)
	}
}

func (s *bakerySuite) TestPrimaryKey(c *gc.C) {
	// Put some records in, with some duplicates. Exercises rollbacks.
	c.Assert(s.storage.Put("foo", "bar"), gc.IsNil)
	c.Assert(s.storage.Put("foo", "bar"), gc.NotNil)
	c.Assert(s.storage.Put("foo", "bar"), gc.NotNil)
	_, err := s.storage.Get("nope")
	c.Assert(err, gc.NotNil)
	c.Assert(s.storage.Put("baz", "quux"), gc.IsNil)
	c.Assert(s.storage.Put("baz", "quux"), gc.NotNil)
	c.Assert(s.storage.Put("a", "b"), gc.IsNil)
	c.Assert(s.storage.Put("empty", ""), gc.IsNil)
	for i, testCase := range []struct {
		location, item string
	}{{"foo", "bar"}, {"baz", "quux"}, {"a", "b"}, {"empty", ""}} {
		comment := gc.Commentf("test#%d expect contents %#v", i, testCase)
		item, err := s.storage.Get(testCase.location)
		c.Assert(err, gc.IsNil, comment)
		c.Assert(item, gc.Equals, testCase.item, comment)
	}
	for i, loc := range []string{"foo", "baz", "a", "empty"} {
		comment := gc.Commentf("test#%d expect unique %s", i, loc)
		var count int
		row := s.db.QueryRow("SELECT COUNT(1) FROM bakery WHERE location = $1", loc)
		c.Assert(row.Scan(&count), gc.IsNil, comment)
		c.Assert(count, gc.Equals, 1, comment)
	}
}
