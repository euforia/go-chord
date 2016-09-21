# Go Chord

This package provides a Golang implementation of the Chord protocol.
Chord is used to organize nodes along a ring in a consistent way. It can be
used to distribute work, build a key/value store, or serve as the underlying
organization for a ring overlay topology.

The protocol is seperated from the implementation of an underlying network
transport or RPC mechanism. Instead Chord relies on a transport implementation.
A TCPTransport is provided that can be used as a reliable Chord RPC mechanism.

This branch has a modified TCPTransport to support a key-value store.  A default
in-memory key-value store is provided.  A custom KV store can also be used.

### Key-Value Store
To use a custom key-value store the `Store` and `KVStore` interfaces need to be 
appropriately implemented

# Documentation

To view the online documentation, go [here](http://godoc.org/github.com/euforia/go-chord).

This code is a modified version of the origin code [armon/go-chord](http://godoc.org/github.com/armon/go-chord).
