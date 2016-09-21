/*
This package is used to provide an implementation of the
Chord network protocol.
*/
package chord

import (
	"crypto/sha1"
	"fmt"
	"hash"
	"io"
	"time"
)

// Implements the methods needed for a Chord ring
type Transport interface {
	// Gets a list of the vnodes on the box
	ListVnodes(string) ([]*Vnode, error)

	// Ping a Vnode, check for liveness
	Ping(*Vnode) (bool, error)

	// Request a nodes predecessor
	GetPredecessor(*Vnode) (*Vnode, error)

	// Notify our successor of ourselves
	Notify(target, self *Vnode) ([]*Vnode, error)

	// Find a successor
	FindSuccessors(*Vnode, int, []byte) ([]*Vnode, error)

	// Clears a predecessor if it matches a given vnode. Used to leave.
	ClearPredecessor(target, self *Vnode) error

	// Instructs a node to skip a given successor. Used to leave.
	SkipSuccessor(target, self *Vnode) error

	// Register for an RPC callbacks
	Register(*Vnode, VnodeRPC)

	// Store operations
	GetKey(*Vnode, []byte) ([]byte, error)
	SetKey(*Vnode, []byte, []byte) error
	DeleteKey(*Vnode, []byte) error
	Snapshot(*Vnode) (io.ReadCloser, error)
	Restore(*Vnode, io.ReadCloser) error
}

// These are the methods to invoke on the registered vnodes
type VnodeRPC interface {
	GetPredecessor() (*Vnode, error)
	Notify(*Vnode) ([]*Vnode, error)
	FindSuccessors(int, []byte) ([]*Vnode, error)
	ClearPredecessor(*Vnode) error
	SkipSuccessor(*Vnode) error

	// Store operations
	GetKey(key []byte) ([]byte, error)
	SetKey(key []byte, value []byte) error
	DeleteKey(key []byte) error
	Snapshot() (io.ReadCloser, error)
	Restore(io.ReadCloser) error
}

// Store manager for a single vnode
type KVStore interface {
	Open(string) error
	Get(key []byte) ([]byte, error)
	Set(key []byte, v []byte) error
	Delete(key []byte) error
	Snapshot() (io.ReadCloser, error)
	Restore(io.ReadCloser) error
	Close() error
}

//
// Managing interface for all vnode stores on a node
//
type Store interface {
	// Create a new store and register it.
	New() KVStore
	// Get datastore for a single vnode.
	Get(string) (KVStore, error)
}

// Delegate to notify on ring events
type Delegate interface {
	NewPredecessor(local, remoteNew, remotePrev *Vnode)
	Leaving(local, pred, succ *Vnode)
	PredecessorLeaving(local, remote *Vnode)
	SuccessorLeaving(local, remote *Vnode)
	Shutdown()
}

// Configuration for Chord nodes
type Config struct {
	Hostname      string           // Local host name
	NumVnodes     int              // Number of vnodes per physical node
	HashFunc      func() hash.Hash // Hash function to use
	StabilizeMin  time.Duration    // Minimum stabilization time
	StabilizeMax  time.Duration    // Maximum stabilization time
	NumSuccessors int              // Number of successors to maintain
	Delegate      Delegate         // Invoked to handle ring events
	hashBits      int              // Bit size of the hash function
}

// Represents a Vnode, local or remote
type Vnode struct {
	Id   []byte // Virtual ID
	Host string // Host identifier
}

// Represents a local Vnode
type localVnode struct {
	Vnode
	ring        *Ring
	successors  []*Vnode
	finger      []*Vnode
	last_finger int
	predecessor *Vnode
	stabilized  time.Time
	timer       *time.Timer
	// Key-Value datastore
	store KVStore
}

// Stores the state required for a Chord ring
type Ring struct {
	config     *Config
	transport  Transport
	vnodes     []*localVnode
	delegateCh chan func()
	shutdown   chan bool
	// Datastore manager for all vnodes provided by the node
	store Store
}

// Returns the default Ring configuration
func DefaultConfig(hostname string) *Config {
	return &Config{
		hostname,
		8,                               // 8 vnodes
		sha1.New,                        // SHA1
		time.Duration(15 * time.Second), // Min stabilization time
		time.Duration(45 * time.Second), // Max stabilization time
		8,   // 8 successors
		nil, // No delegate
		160, // 160bit hash function
	}
}

func newRing(store Store) *Ring {
	if store == nil {
		return &Ring{store: NewInMemKVStore()}
	}
	return &Ring{store: store}
}

// Creates a new Chord ring given the config and transport
func Create(conf *Config, trans Transport, store Store) (*Ring, error) {
	// Initialize the hash bits
	conf.hashBits = conf.HashFunc().Size() * 8

	// Create and initialize a ring
	ring := newRing(store)
	ring.init(conf, trans)
	ring.setLocalSuccessors()
	ring.schedule()
	return ring, nil
}

// Joins an existing Chord ring
func Join(conf *Config, trans Transport, store Store, existing string) (*Ring, error) {
	// Initialize the hash bits
	conf.hashBits = conf.HashFunc().Size() * 8

	// Request a list of Vnodes from the remote host
	hosts, err := trans.ListVnodes(existing)
	if err != nil {
		return nil, err
	}
	if hosts == nil || len(hosts) == 0 {
		return nil, fmt.Errorf("Remote host has no vnodes!")
	}

	// Create a ring
	ring := newRing(store)
	ring.init(conf, trans)

	// Acquire a live successor for each Vnode
	for _, vn := range ring.vnodes {
		// Get the nearest remote vnode
		nearest := nearestVnodeToKey(hosts, vn.Id)

		// Query for a list of successors to this Vnode
		succs, err := trans.FindSuccessors(nearest, conf.NumSuccessors, vn.Id)
		if err != nil {
			return nil, fmt.Errorf("Failed to find successor for vnodes! Got %s", err)
		}
		if succs == nil || len(succs) == 0 {
			return nil, fmt.Errorf("Failed to find successor for vnodes! Got no vnodes!")
		}

		// Assign the successors
		for idx, s := range succs {
			vn.successors[idx] = s
		}
	}

	// Start delegate handler
	if ring.config.Delegate != nil {
		go ring.delegateHandler()
	}

	// Do a fast stabilization, will schedule regular execution
	for _, vn := range ring.vnodes {
		vn.stabilize()
	}
	return ring, nil
}

// Leaves a given Chord ring and shuts down the local vnodes
func (r *Ring) Leave() error {
	// Shutdown the vnodes first to avoid further stabilization runs
	r.stopVnodes()

	// Instruct each vnode to leave
	var err error
	for _, vn := range r.vnodes {
		err = mergeErrors(err, vn.leave())
	}

	// Wait for the delegate callbacks to complete
	r.stopDelegate()
	return err
}

// Shutdown shuts down the local processes in a given Chord ring
// Blocks until all the vnodes terminate.
func (r *Ring) Shutdown() {
	r.stopVnodes()
	r.stopDelegate()
}

// Does a key lookup for up to N successors of a key
func (r *Ring) Lookup(n int, key []byte) ([]*Vnode, error) {
	// Ensure that n is sane
	if n > r.config.NumSuccessors {
		return nil, fmt.Errorf("Cannot ask for more successors than NumSuccessors!")
	}

	// Hash the key
	h := r.config.HashFunc()
	h.Write(key)
	key_hash := h.Sum(nil)

	// Find the nearest local vnode
	nearest := r.nearestVnode(key_hash)

	// Use the nearest node for the lookup
	successors, err := nearest.FindSuccessors(n, key_hash)
	if err != nil {
		return nil, err
	}

	// Trim the nil successors
	for successors[len(successors)-1] == nil {
		successors = successors[:len(successors)-1]
	}
	return successors, nil
}

func (r *Ring) GetKey(n int, key []byte) ([]RingKeyValue, error) {
	vns, err := r.Lookup(n, key)
	if err != nil {
		return nil, err
	}

	out := make([]RingKeyValue, len(vns))
	for i, vn := range vns {
		r, e := r.transport.GetKey(vn, key)
		if e == nil {
			rkv := RingKeyValue{Key: key, Value: r, Vnode: vn}
			out[i] = rkv
			continue

		}
		err = e
	}
	return out, err
}

// Try to delete n keys on the ring.  Only the last error is visible previous
// errors are masked.
func (r *Ring) DeleteKey(n int, key []byte) error {
	vns, err := r.Lookup(n, key)
	if err == nil {
		for _, vn := range vns {
			if e := r.transport.DeleteKey(vn, key); e != nil {
				err = e
			}
		}
	}
	return err
}

func (r *Ring) SetKey(n int, key []byte, v []byte) ([]RingKeyValue, error) {
	vns, err := r.Lookup(n, key)
	if err != nil {
		return nil, err
	}

	out := make([]RingKeyValue, len(vns))
	for i, vn := range vns {
		e := r.transport.SetKey(vn, key, v)
		if e == nil {
			rkv := RingKeyValue{Key: key, Value: v, Vnode: vn}
			out[i] = rkv
			continue
		}
		err = e
	}
	return out, err
}

func (r *Ring) SnapshotVnode(vn *Vnode) (io.ReadCloser, error) {
	return r.transport.Snapshot(vn)
}

func (r *Ring) RestoreVnode(vn *Vnode, rd io.ReadCloser) error {
	return r.transport.Restore(vn, rd)
}

type RingKeyValue struct {
	Vnode *Vnode
	Key   []byte
	Value []byte
}
