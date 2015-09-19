package postgres

import (
	"database/sql"

	"gopkg.in/errgo.v1"
	"gopkg.in/macaroon-bakery.v1/bakery"
)

const createBakeryTable = `
CREATE TABLE IF NOT EXISTS bakery (
	location TEXT,
	item TEXT,
	PRIMARY KEY(location)
)`

type bakeryStorage struct {
	db *sql.DB
}

// NewBakeryStorage returns a new PostgreSQL bakery storage instance.
func NewBakeryStorage(db *sql.DB) (*bakeryStorage, error) {
	st := &bakeryStorage{
		db: db,
	}
	err := st.createIfNotExists()
	if err != nil {
		return nil, errgo.Mask(err, errgo.Any)
	}
	return st, nil
}

// Get implements bakery.Storage.
func (s *bakeryStorage) Get(location string) (string, error) {
	var item string
	row := s.db.QueryRow(`SELECT item FROM bakery WHERE location = $1`, location)
	err := row.Scan(&item)
	if err == sql.ErrNoRows {
		return "", bakery.ErrNotFound
	} else if err != nil {
		return "", errgo.Mask(err, errgo.Any)
	}
	return item, nil
}

// Put implements bakery.Storage.
func (s *bakeryStorage) Put(location, item string) (_err error) {
	tx, err := s.db.Begin()
	if err != nil {
		return errgo.Mask(err, errgo.Any)
	}
	defer func() {
		_err = completeTransaction(tx, _err)
	}()

	_, err = tx.Exec(`INSERT INTO bakery (location, item) VALUES ($1, $2)`, location, item)
	return errgo.Mask(err, errgo.Any)
}

// Del implements bakery.Storage.
func (s *bakeryStorage) Del(location string) (_err error) {
	tx, err := s.db.Begin()
	if err != nil {
		return errgo.Mask(err, errgo.Any)
	}
	defer func() {
		_err = completeTransaction(tx, _err)
	}()

	_, err = tx.Exec(`DELETE FROM bakery WHERE location = $1`, location)
	return errgo.Mask(err, errgo.Any)
}

func (s *bakeryStorage) createIfNotExists() error {
	_, err := s.db.Exec(createBakeryTable)
	return errgo.Mask(err, errgo.Any)
}
