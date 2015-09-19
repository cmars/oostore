package postgres

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
	"gopkg.in/errgo.v1"

	"github.com/cmars/oostore"
)

const createObjectTable = `CREATE TABLE IF NOT EXISTS object (
    id          TEXT,
	contentType TEXT,
	contents    bytea,
	PRIMARY KEY(id))`

type objectStorage struct {
	db *sql.DB
}

// NewObjectStorage returns a new PostgreSQL object storage instance.
func NewObjectStorage(db *sql.DB) (*objectStorage, error) {
	st := &objectStorage{
		db: db,
	}
	err := st.createIfNotExists()
	if err != nil {
		return nil, errgo.Mask(err, errgo.Any)
	}
	return st, nil
}

// Get implements oostore.Storage.
func (s *objectStorage) Get(id string) ([]byte, string, error) {
	var (
		contents    []byte
		contentType string
	)
	row := s.db.QueryRow(`SELECT contents, contentType FROM object WHERE id = $1`, id)
	err := row.Scan(&contents, &contentType)
	if err == sql.ErrNoRows {
		return nil, "", oostore.ErrNotFound
	} else if err != nil {
		return nil, "", errgo.Mask(err, errgo.Any)
	}
	return contents, contentType, nil
}

// Put implements oostore.Storage.
func (s *objectStorage) Put(id string, contents []byte, contentType string) (_err error) {
	tx, err := s.db.Begin()
	if err != nil {
		return errgo.Mask(err, errgo.Any)
	}
	defer func() {
		_err = completeTransaction(tx, _err)
	}()

	_, err = tx.Exec(`INSERT INTO object (id, contents, contentType) VALUES ($1, $2, $3)`,
		id, contents, contentType)
	return errgo.Mask(err, errgo.Any)
}

// Delete implements oostore.Storage.
func (s *objectStorage) Delete(id string) (_err error) {
	tx, err := s.db.Begin()
	if err != nil {
		return errgo.Mask(err, errgo.Any)
	}
	defer func() {
		_err = completeTransaction(tx, _err)
	}()

	_, err = tx.Exec(`DELETE FROM object WHERE id = $1`, id)
	return errgo.Mask(err, errgo.Any)
}

func (s *objectStorage) createIfNotExists() error {
	_, err := s.db.Exec(createObjectTable)
	return errgo.Mask(err, errgo.Any)
}

func completeTransaction(tx *sql.Tx, errResult error) error {
	if errResult != nil {
		err := tx.Rollback()
		if err != nil {
			log.Printf("warning: failed to rollback: %v", err)
		}
	} else {
		errResult = tx.Commit()
	}
	return errResult
}
