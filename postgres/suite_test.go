package postgres_test

import (
	"database/sql"
	"testing"

	"github.com/cmars/pgtest"
	gc "gopkg.in/check.v1"
)

func Test(t *testing.T) { gc.TestingT(t) }

type postgresSuite struct {
	pgtest.PGSuite

	db *sql.DB
}

func (s *postgresSuite) SetUpTest(c *gc.C) {
	s.PGSuite.SetUpTest(c)
	var err error
	s.db, err = sql.Open("postgres", s.URL)
	c.Assert(err, gc.IsNil)
}
