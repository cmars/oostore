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

type contentDoc struct {
	ContentType string
	Contents    []byte
}

type memStorage map[string]contentDoc

// NewMemStorage returns a new storage implementation that only keeps things in
// memory. Primarily useful for testing. Ephemeral storage for production use
// would probably want to cap memory usage, implement some kind of expiration
// policy, etc.
func NewMemStorage() memStorage {
	return memStorage(make(map[string]contentDoc))
}

// Get implements Storage.
func (s memStorage) Get(id string) ([]byte, string, error) {
	doc, ok := s[id]
	if !ok {
		return nil, "", ErrNotFound
	}
	return doc.Contents, doc.ContentType, nil
}

// Put implements Storage.
func (s memStorage) Put(id string, contents []byte, contentType string) error {
	s[id] = contentDoc{Contents: contents, ContentType: contentType}
	return nil
}

// Delete implements Storage.
func (s memStorage) Delete(id string) error {
	delete(s, id)
	return nil
}
