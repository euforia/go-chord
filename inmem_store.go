package chord

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"sync"

	"gopkg.in/vmihailenco/msgpack.v2"
)

//
// Store manager for vnodes on a given node
//
type InMemKVStore struct {
	mu sync.Mutex
	a  []*InMemKVDatastore
}

func NewInMemKVStore() *InMemKVStore {
	return &InMemKVStore{a: []*InMemKVDatastore{}}
}

func (s *InMemKVStore) New() KVStore {
	s.mu.Lock()
	defer s.mu.Unlock()

	ikv := &InMemKVDatastore{}
	s.a = append(s.a, ikv)

	return ikv
}

func (s *InMemKVStore) Get(vnodeId string) (KVStore, error) {
	for _, a := range s.a {
		if a.id == vnodeId {
			return a, nil
		}
	}
	return nil, fmt.Errorf("not found: %s", vnodeId)
}

//
// A single datastore for a vnode
//
type InMemKVDatastore struct {
	mu sync.Mutex
	id string            // vnode id
	m  map[string][]byte // data
}

func (s *InMemKVDatastore) Open(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.id = id
	s.m = map[string][]byte{}
	return nil
}

func (s *InMemKVDatastore) Snapshot() (io.ReadCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	wr := new(bytes.Buffer)
	err := msgpack.NewEncoder(wr).Encode(s.m)
	if err == nil {
		return &readCloser{wr}, nil
	}
	return nil, err
}

func (s *InMemKVDatastore) Restore(r io.ReadCloser) error {
	defer r.Close()

	var m map[string][]byte
	err := msgpack.NewDecoder(r).Decode(&m)

	if err == nil && len(m) > 0 {

		s.mu.Lock()
		for k, v := range m {
			s.m[k] = v
		}
		s.mu.Unlock()
		log.Printf("loglevel=DBG action=restored keys=%d", len(m))
	}
	return err
}

func (s *InMemKVDatastore) Get(key []byte) ([]byte, error) {
	if val, ok := s.m[string(key)]; ok {
		return val, nil
	}

	return nil, fmt.Errorf("not found: %s", key)
}

func (s *InMemKVDatastore) Set(key []byte, v []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.m[string(key)] = v

	return nil
}

func (s *InMemKVDatastore) Delete(key []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.m, string(key))
	return nil
}

func (s *InMemKVDatastore) Close() error {
	return nil
}

type readCloser struct {
	io.Reader
}

func (rc *readCloser) Close() error {
	return nil
}
