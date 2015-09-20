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

	result, err := tx.Exec(`DELETE FROM object WHERE id = $1`, id)
	if err != nil {
		return errgo.Mask(err, errgo.Any)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return errgo.Mask(err, errgo.Any)
	}
	switch n {
	case 0:
		return oostore.ErrNotFound
	case 1:
		return nil
	default:
		return errgo.Newf("deleted %d rows, expected 1", n)
	}
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
