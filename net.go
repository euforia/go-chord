package chord

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	tcpPing byte = iota
	tcpListReq
	tcpGetPredReq
	tcpNotifyReq
	tcpFindSucReq
	tcpClearPredReq
	tcpSkipSucReq
)

const (
	tcpSetKeyReq byte = 10
	tcpGetKeyReq byte = 11
	tcpDelKeyReq byte = 12

	tcpSnapshotReq byte = 20
	tcpRestoreReq  byte = 21
)

type tcpOutConn struct {
	host   string
	sock   *net.TCPConn
	header tcpHeader // Request header
	enc    Encoder
	dec    Decoder
	used   time.Time
}

type tcpHeader struct {
	ReqType byte
}

// Potential body types
type tcpBodyError struct {
	Err error
}
type tcpBodyString struct {
	S string
}
type tcpBodyVnode struct {
	Vn *Vnode
}
type tcpBodyTwoVnode struct {
	Target *Vnode
	Vn     *Vnode
}
type tcpBodyFindSuc struct {
	Target *Vnode
	Num    int
	Key    []byte
}
type tcpBodyVnodeError struct {
	Vnode *Vnode
	Err   error
}
type tcpBodyVnodeListError struct {
	Vnodes []*Vnode
	Err    error
}
type tcpBodyBoolError struct {
	B   bool
	Err error
}
type tcpBodyKey struct {
	VN *Vnode
	K  []byte
}
type tcpBodyKeyValue struct {
	VN *Vnode
	K  []byte
	V  []byte
}
type tcpBodyKeyErr struct {
	K   []byte
	Err string
}
type tcpBodyKeyValueErr struct {
	K   []byte
	V   []byte
	Err string
}

/*
TCPTransport provides a TCP based Chord transport layer. This allows Chord to be implemented over a network, instead of only using the LocalTransport. It is meant to be a simple implementation, optimizing for simplicity instead of performance.  Messages are sent with a header frame, followed by a body frame. All data is encoded/decoded using a separate Encoder/Decoder interface.

Internally, there is 1 Goroutine listening for inbound connections, 1 Goroutine PER inbound connection.

This implementation contains a key value store rpc calls on top of the original provided TCPTransport
*/
type TCPTransport struct {
	sock     *net.TCPListener
	timeout  time.Duration
	maxIdle  time.Duration
	lock     sync.RWMutex
	local    map[string]*localRPC
	inbound  map[*net.TCPConn]struct{}
	poolLock sync.Mutex
	pool     map[string][]*tcpOutConn
	shutdown int32
}

// Creates a new TCP transport on the given listen address with the
// configured timeout duration.
func InitTCPTransport(listen string, timeout time.Duration) (*TCPTransport, error) {
	// Try to start the listener
	sock, err := net.Listen("tcp", listen)
	if err != nil {
		return nil, err
	}

	// allocate maps
	local := make(map[string]*localRPC)
	inbound := make(map[*net.TCPConn]struct{})
	pool := make(map[string][]*tcpOutConn)

	// Maximum age of a connection
	maxIdle := time.Duration(300 * time.Second)

	// Setup the transport
	tcp := &TCPTransport{
		sock:    sock.(*net.TCPListener),
		timeout: timeout,
		maxIdle: maxIdle,
		local:   local,
		inbound: inbound,
		pool:    pool,
	}

	// Listen for connections
	go tcp.listen()

	// Reap old connections
	go tcp.reapOld()

	// Done
	return tcp, nil
}

// Checks for a local vnode
func (t *TCPTransport) get(vn *Vnode) (VnodeRPC, bool) {
	key := vn.String()
	t.lock.RLock()
	defer t.lock.RUnlock()
	w, ok := t.local[key]
	if ok {
		return w.obj, ok
	} else {
		return nil, ok
	}
}

// Gets an outbound connection to a host
func (t *TCPTransport) getConn(host string) (*tcpOutConn, error) {
	// Check if we have a conn cached
	var out *tcpOutConn
	t.poolLock.Lock()
	if atomic.LoadInt32(&t.shutdown) == 1 {
		t.poolLock.Unlock()
		return nil, fmt.Errorf("TCP transport is shutdown")
	}
	list, ok := t.pool[host]
	if ok && len(list) > 0 {
		out = list[len(list)-1]
		list = list[:len(list)-1]
		t.pool[host] = list
	}
	t.poolLock.Unlock()
	if out != nil {
		// Verify that the socket is valid. Might be closed.
		if _, err := out.sock.Read(nil); err == nil {
			return out, nil
		}

		// explicitly close ??
		out.sock.Close()

	}
	// Try to establish a connection
	conn, err := net.DialTimeout("tcp", host, t.timeout)
	if err != nil {
		return nil, err
	}

	// Setup the socket
	sock := conn.(*net.TCPConn)
	t.setupConn(sock)
	enc := NewEncoder(sock)
	dec := NewDecoder(sock)
	now := time.Now()

	// Wrap the sock
	out = &tcpOutConn{host: host, sock: sock, enc: enc, dec: dec, used: now}
	return out, nil
}

// Returns an outbound TCP connection to the pool
func (t *TCPTransport) returnConn(o *tcpOutConn) {
	// Update the last used time
	o.used = time.Now()

	// Push back into the pool
	t.poolLock.Lock()
	defer t.poolLock.Unlock()
	if atomic.LoadInt32(&t.shutdown) == 1 {
		o.sock.Close()
		return
	}
	list, _ := t.pool[o.host]
	t.pool[o.host] = append(list, o)
}

// Setup a connection
func (t *TCPTransport) setupConn(c *net.TCPConn) {
	c.SetNoDelay(true)
	c.SetKeepAlive(true)
}

// Gets a list of the vnodes on the box
func (t *TCPTransport) ListVnodes(host string) ([]*Vnode, error) {
	// Get a conn
	out, err := t.getConn(host)
	if err != nil {
		return nil, err
	}

	// Response channels
	respChan := make(chan []*Vnode, 1)
	errChan := make(chan error, 1)

	go func() {
		// Send a list command
		out.header.ReqType = tcpListReq
		body := tcpBodyString{S: host}
		if err := out.enc.Encode(&out.header); err != nil {
			errChan <- err
			return
		}
		if err := out.enc.Encode(&body); err != nil {
			errChan <- err
			return
		}

		// Read in the response
		resp := tcpBodyVnodeListError{}
		if err := out.dec.Decode(&resp); err != nil {
			errChan <- err
		}

		// Return the connection
		t.returnConn(out)
		if resp.Err == nil {
			respChan <- resp.Vnodes
		} else {
			errChan <- resp.Err
		}
	}()

	select {
	case <-time.After(t.timeout):
		return nil, fmt.Errorf("ListVnodes timed out!")
	case err := <-errChan:
		return nil, err
	case res := <-respChan:
		return res, nil
	}
}

// Ping a Vnode, check for liveness
func (t *TCPTransport) Ping(vn *Vnode) (bool, error) {
	// Get a conn
	out, err := t.getConn(vn.Host)
	if err != nil {
		return false, err
	}

	// Response channels
	respChan := make(chan bool, 1)
	errChan := make(chan error, 1)

	go func() {
		// Send a list command
		out.header.ReqType = tcpPing
		body := tcpBodyVnode{Vn: vn}
		if err := out.enc.Encode(&out.header); err != nil {
			errChan <- err
			return
		}
		if err := out.enc.Encode(&body); err != nil {
			errChan <- err
			return
		}

		// Read in the response
		resp := tcpBodyBoolError{}
		if err := out.dec.Decode(&resp); err != nil {
			errChan <- err
			return
		}

		// Return the connection
		t.returnConn(out)
		if resp.Err == nil {
			respChan <- resp.B
		} else {
			errChan <- resp.Err
		}
	}()

	select {
	case <-time.After(t.timeout):
		return false, fmt.Errorf("Ping timed out!")
	case err := <-errChan:
		return false, err
	case res := <-respChan:
		return res, nil
	}
}

func (t *TCPTransport) SetKey(vn *Vnode, key []byte, value []byte) error {
	// Get a conn
	out, err := t.getConn(vn.Host)
	if err != nil {
		return err
	}

	// Response channels
	respChan := make(chan bool, 1)
	errChan := make(chan error, 1)

	go func() {
		// Send a get key cmd
		out.header.ReqType = tcpSetKeyReq
		body := tcpBodyKeyValue{K: key, V: value, VN: vn}
		if err := out.enc.Encode(&out.header); err != nil {
			errChan <- err
			return
		}
		if err := out.enc.Encode(&body); err != nil {
			errChan <- err
			return
		}

		// Read in the response
		resp := tcpBodyKeyErr{}
		if err := out.dec.Decode(&resp); err != nil {
			errChan <- err
			return
		}

		// Return the connection
		t.returnConn(out)
		if resp.Err != "" {
			errChan <- fmt.Errorf(resp.Err)
		} else {
			respChan <- true
		}
	}()

	select {
	case <-time.After(t.timeout):
		return fmt.Errorf("SetKey timed out!")
	case err := <-errChan:
		return err
	case <-respChan:
		return nil
	}
	return nil
}

func (t *TCPTransport) DeleteKey(vn *Vnode, key []byte) error {
	// Get a conn
	out, err := t.getConn(vn.Host)
	if err != nil {
		return err
	}
	// Response channels
	respChan := make(chan bool, 1)
	errChan := make(chan error, 1)

	go func() {
		// Send a get key cmd
		out.header.ReqType = tcpDelKeyReq
		body := tcpBodyKey{VN: vn, K: key}
		if err := out.enc.Encode(&out.header); err != nil {
			errChan <- err
			return
		}
		if err := out.enc.Encode(&body); err != nil {
			errChan <- err
			return
		}

		// Read in the response
		resp := tcpBodyKeyErr{}
		if err := out.dec.Decode(&resp); err != nil {
			errChan <- err
			return
		}

		// Return the connection
		t.returnConn(out)
		if resp.Err != "" {
			errChan <- fmt.Errorf(resp.Err)
		} else {
			respChan <- true
		}
	}()

	select {
	case <-time.After(t.timeout):
		return fmt.Errorf("DeleteKey timed out!")
	case err := <-errChan:
		return err
	case <-respChan:
		return nil
	}
	return nil
}

func (t *TCPTransport) GetKey(vn *Vnode, key []byte) ([]byte, error) {
	// Get a conn
	out, err := t.getConn(vn.Host)
	if err != nil {
		return nil, err
	}
	// Response channels
	respChan := make(chan []byte, 1)
	errChan := make(chan error, 1)

	go func() {
		// Send a get key cmd
		out.header.ReqType = tcpGetKeyReq
		body := tcpBodyKey{K: key, VN: vn}
		if err := out.enc.Encode(&out.header); err != nil {
			errChan <- err
			return
		}
		if err := out.enc.Encode(&body); err != nil {
			errChan <- err
			return
		}

		// Read in the response
		resp := tcpBodyKeyValueErr{}
		if err := out.dec.Decode(&resp); err != nil {
			errChan <- err
			return
		}

		// Return the connection
		t.returnConn(out)
		if resp.Err != "" {
			errChan <- fmt.Errorf(resp.Err)
		} else {
			respChan <- resp.V
		}
	}()

	select {
	case <-time.After(t.timeout):
		return nil, fmt.Errorf("GetKey timed out!")
	case err := <-errChan:
		return nil, err
	case resp := <-respChan:
		return resp, nil
	}
	return nil, nil
}

// Request a nodes predecessor
func (t *TCPTransport) GetPredecessor(vn *Vnode) (*Vnode, error) {
	// Get a conn
	out, err := t.getConn(vn.Host)
	if err != nil {
		return nil, err
	}

	respChan := make(chan *Vnode, 1)
	errChan := make(chan error, 1)

	go func() {
		// Send a list command
		out.header.ReqType = tcpGetPredReq
		body := tcpBodyVnode{Vn: vn}
		if err := out.enc.Encode(&out.header); err != nil {
			errChan <- err
			return
		}
		if err := out.enc.Encode(&body); err != nil {
			errChan <- err
			return
		}

		// Read in the response
		resp := tcpBodyVnodeError{}
		if err := out.dec.Decode(&resp); err != nil {
			errChan <- err
			return
		}

		// Return the connection
		t.returnConn(out)
		if resp.Err == nil {
			respChan <- resp.Vnode
		} else {
			errChan <- resp.Err
		}
	}()

	select {
	case <-time.After(t.timeout):
		return nil, fmt.Errorf("GetPredecessor timed out!")
	case err := <-errChan:
		return nil, err
	case res := <-respChan:
		return res, nil
	}
}

// Notify our successor of ourselves
func (t *TCPTransport) Notify(target, self *Vnode) ([]*Vnode, error) {
	// Get a conn
	out, err := t.getConn(target.Host)
	if err != nil {
		return nil, err
	}

	respChan := make(chan []*Vnode, 1)
	errChan := make(chan error, 1)

	go func() {
		// Send a list command
		out.header.ReqType = tcpNotifyReq
		body := tcpBodyTwoVnode{Target: target, Vn: self}
		if err := out.enc.Encode(&out.header); err != nil {
			errChan <- err
			return
		}
		if err := out.enc.Encode(&body); err != nil {
			errChan <- err
			return
		}

		// Read in the response
		resp := tcpBodyVnodeListError{}
		if err := out.dec.Decode(&resp); err != nil {
			errChan <- err
			return
		}

		// Return the connection
		t.returnConn(out)
		if resp.Err == nil {
			respChan <- resp.Vnodes
		} else {
			errChan <- resp.Err
		}
	}()

	select {
	case <-time.After(t.timeout):
		return nil, fmt.Errorf("Notify timed out!")
	case err := <-errChan:
		return nil, err
	case res := <-respChan:
		return res, nil
	}
}

// Find a successor
func (t *TCPTransport) FindSuccessors(vn *Vnode, n int, k []byte) ([]*Vnode, error) {
	// Get a conn
	out, err := t.getConn(vn.Host)
	if err != nil {
		return nil, err
	}

	respChan := make(chan []*Vnode, 1)
	errChan := make(chan error, 1)

	go func() {
		// Send a list command
		out.header.ReqType = tcpFindSucReq
		body := tcpBodyFindSuc{Target: vn, Num: n, Key: k}
		if err := out.enc.Encode(&out.header); err != nil {
			errChan <- err
			return
		}
		if err := out.enc.Encode(&body); err != nil {
			errChan <- err
			return
		}

		// Read in the response
		resp := tcpBodyVnodeListError{}
		if err := out.dec.Decode(&resp); err != nil {
			errChan <- err
			return
		}

		// Return the connection
		t.returnConn(out)
		if resp.Err == nil {
			respChan <- resp.Vnodes
		} else {
			errChan <- resp.Err
		}
	}()

	select {
	case <-time.After(t.timeout):
		return nil, fmt.Errorf("FindSuccessors timed out!")
	case err := <-errChan:
		return nil, err
	case res := <-respChan:
		return res, nil
	}
}

// Clears a predecessor if it matches a given vnode. Used to leave.
func (t *TCPTransport) ClearPredecessor(target, self *Vnode) error {
	// Get a conn
	out, err := t.getConn(target.Host)
	if err != nil {
		return err
	}

	respChan := make(chan bool, 1)
	errChan := make(chan error, 1)

	go func() {
		// Send a list command
		out.header.ReqType = tcpClearPredReq
		body := tcpBodyTwoVnode{Target: target, Vn: self}
		if err := out.enc.Encode(&out.header); err != nil {
			errChan <- err
			return
		}
		if err := out.enc.Encode(&body); err != nil {
			errChan <- err
			return
		}

		// Read in the response
		resp := tcpBodyError{}
		if err := out.dec.Decode(&resp); err != nil {
			errChan <- err
			return
		}

		// Return the connection
		t.returnConn(out)
		if resp.Err == nil {
			respChan <- true
		} else {
			errChan <- resp.Err
		}
	}()

	select {
	case <-time.After(t.timeout):
		return fmt.Errorf("ClearPredecessor timed out!")
	case err := <-errChan:
		return err
	case <-respChan:
		return nil
	}
}

// Instructs a node to skip a given successor. Used to leave.
func (t *TCPTransport) SkipSuccessor(target, self *Vnode) error {
	// Get a conn
	out, err := t.getConn(target.Host)
	if err != nil {
		return err
	}

	respChan := make(chan bool, 1)
	errChan := make(chan error, 1)

	go func() {
		// Send a list command
		out.header.ReqType = tcpSkipSucReq
		body := tcpBodyTwoVnode{Target: target, Vn: self}
		if err := out.enc.Encode(&out.header); err != nil {
			errChan <- err
			return
		}
		if err := out.enc.Encode(&body); err != nil {
			errChan <- err
			return
		}

		// Read in the response
		resp := tcpBodyError{}
		if err := out.dec.Decode(&resp); err != nil {
			errChan <- err
			return
		}

		// Return the connection
		t.returnConn(out)
		if resp.Err == nil {
			respChan <- true
		} else {
			errChan <- resp.Err
		}
	}()

	select {
	case <-time.After(t.timeout):
		return fmt.Errorf("SkipSuccessor timed out!")
	case err := <-errChan:
		return err
	case <-respChan:
		return nil
	}
}

func (t *TCPTransport) Snapshot(vn *Vnode) (io.ReadCloser, error) {
	// Get a conn
	out, err := t.getConn(vn.Host)
	if err != nil {
		return nil, err
	}

	respChan := make(chan io.ReadCloser, 1)
	errChan := make(chan error, 1)

	go func() {
		// Send a list command
		out.header.ReqType = tcpSnapshotReq
		if err := out.enc.Encode(&out.header); err != nil {
			errChan <- err
			return
		}

		body := tcpBodyVnode{Vn: vn}
		if err := out.enc.Encode(&body); err != nil {
			errChan <- err
			return
		}

		respChan <- out.sock
	}()

	// TODO: increase time out for snapshots/restore
	select {
	case <-time.After(t.timeout):
		return nil, fmt.Errorf("Snapshot timed out!")
	case err := <-errChan:
		return nil, err
	case res := <-respChan:
		return res, nil
	}
}

func (t *TCPTransport) Restore(vn *Vnode, r io.ReadCloser) error {
	// Get a conn
	out, err := t.getConn(vn.Host)
	if err != nil {
		return err
	}

	respChan := make(chan bool, 1)
	errChan := make(chan error, 1)

	go func() {
		// Send a list command
		out.header.ReqType = tcpRestoreReq
		if err := out.enc.Encode(&out.header); err != nil {
			errChan <- err
			return
		}

		body := tcpBodyVnode{Vn: vn}
		if err := out.enc.Encode(&body); err != nil {
			errChan <- err
			return
		}

		if _, err = io.Copy(out.sock, r); err != nil {
			errChan <- err
			return
		}
		respChan <- true
	}()

	// TODO: increase time out for snapshots/restore
	select {
	case <-time.After(t.timeout):
		return fmt.Errorf("Restore timed out!")
	case err := <-errChan:
		return err
	case <-respChan:
		return nil
	}
}

// Register for an RPC callbacks
func (t *TCPTransport) Register(v *Vnode, o VnodeRPC) {
	key := v.String()
	t.lock.Lock()
	t.local[key] = &localRPC{v, o}
	t.lock.Unlock()
}

// Shutdown the TCP transport
func (t *TCPTransport) Shutdown() {
	atomic.StoreInt32(&t.shutdown, 1)
	t.sock.Close()

	// Close all the inbound connections
	t.lock.RLock()
	for conn := range t.inbound {
		conn.Close()
	}
	t.lock.RUnlock()

	// Close all the outbound
	t.poolLock.Lock()
	for _, conns := range t.pool {
		for _, out := range conns {
			out.sock.Close()
		}
	}
	t.pool = nil
	t.poolLock.Unlock()
}

// Closes old outbound connections
func (t *TCPTransport) reapOld() {
	for {
		if atomic.LoadInt32(&t.shutdown) == 1 {
			return
		}
		time.Sleep(30 * time.Second)
		t.reapOnce()
	}
}

func (t *TCPTransport) reapOnce() {
	t.poolLock.Lock()
	defer t.poolLock.Unlock()
	for host, conns := range t.pool {
		max := len(conns)
		for i := 0; i < max; i++ {
			if time.Since(conns[i].used) > t.maxIdle {
				conns[i].sock.Close()
				conns[i], conns[max-1] = conns[max-1], nil
				max--
				i--
			}
		}
		// Trim any idle conns
		t.pool[host] = conns[:max]
	}
}

// Listens for inbound connections
func (t *TCPTransport) listen() {
	for {
		conn, err := t.sock.AcceptTCP()
		if err != nil {
			if atomic.LoadInt32(&t.shutdown) == 0 {
				fmt.Printf("[ERR] Error accepting TCP connection! %s", err)
				continue
			} else {
				return
			}
		}

		// Setup the conn
		t.setupConn(conn)

		// Register the inbound conn
		t.lock.Lock()
		t.inbound[conn] = struct{}{}
		t.lock.Unlock()

		// Start handler
		go t.handleConn(conn)
	}
}

// Handles inbound TCP connections
func (t *TCPTransport) handleConn(conn *net.TCPConn) {
	// Defer the cleanup
	defer func() {
		t.lock.Lock()
		delete(t.inbound, conn)
		t.lock.Unlock()
		conn.Close()
	}()

	dec := NewDecoder(conn)
	enc := NewEncoder(conn)
	header := tcpHeader{}
	var sendResp interface{}
	for {
		// Get the header
		if err := dec.Decode(&header); err != nil {
			if atomic.LoadInt32(&t.shutdown) == 0 && err.Error() != "EOF" {
				log.Printf("[ERR] Failed to decode TCP header! Got %s", err)
			}
			return
		}

		// Read in the body and process request
		switch header.ReqType {
		case tcpPing:
			body := tcpBodyVnode{}
			if err := dec.Decode(&body); err != nil {
				log.Printf("[ERR] Failed to decode TCP body! Got %s", err)
				return
			}

			// Generate a response
			_, ok := t.get(body.Vn)
			if ok {
				sendResp = tcpBodyBoolError{B: ok, Err: nil}
			} else {
				sendResp = tcpBodyBoolError{
					B: ok,
					Err: fmt.Errorf("Target VN not found! Target %s:%s",
						body.Vn.Host, body.Vn.String()),
				}
			}

		case tcpListReq:
			body := tcpBodyString{}
			if err := dec.Decode(&body); err != nil {
				log.Printf("[ERR] Failed to decode TCP body! Got %s", err)
				return
			}

			// Generate all the local clients
			res := make([]*Vnode, 0, len(t.local))

			// Build list
			t.lock.RLock()
			for _, v := range t.local {
				res = append(res, v.vnode)
			}
			t.lock.RUnlock()

			// Make response
			sendResp = tcpBodyVnodeListError{Vnodes: trimSlice(res)}

		case tcpGetPredReq:
			body := tcpBodyVnode{}
			if err := dec.Decode(&body); err != nil {
				log.Printf("[ERR] Failed to decode TCP body! Got %s", err)
				return
			}

			// Generate a response
			obj, ok := t.get(body.Vn)
			resp := tcpBodyVnodeError{}
			sendResp = &resp
			if ok {
				node, err := obj.GetPredecessor()
				resp.Vnode = node
				resp.Err = err
			} else {
				resp.Err = fmt.Errorf("Target VN not found! Target %s:%s",
					body.Vn.Host, body.Vn.String())
			}

		case tcpNotifyReq:
			body := tcpBodyTwoVnode{}
			if err := dec.Decode(&body); err != nil {
				log.Printf("[ERR] Failed to decode TCP body! Got %s", err)
				return
			}

			// Generate a response
			obj, ok := t.get(body.Target)
			resp := tcpBodyVnodeListError{}
			sendResp = &resp
			if ok {
				nodes, err := obj.Notify(body.Vn)
				resp.Vnodes = trimSlice(nodes)
				resp.Err = err
			} else {
				resp.Err = fmt.Errorf("Target VN not found! Target %s:%s",
					body.Target.Host, body.Target.String())
			}

		case tcpFindSucReq:
			body := tcpBodyFindSuc{}
			if err := dec.Decode(&body); err != nil {
				log.Printf("[ERR] Failed to decode TCP body! Got %s", err)
				return
			}

			// Generate a response
			obj, ok := t.get(body.Target)
			resp := tcpBodyVnodeListError{}
			sendResp = &resp
			if ok {
				nodes, err := obj.FindSuccessors(body.Num, body.Key)
				resp.Vnodes = trimSlice(nodes)
				resp.Err = err
			} else {
				resp.Err = fmt.Errorf("Target VN not found! Target %s:%s",
					body.Target.Host, body.Target.String())
			}

		case tcpClearPredReq:
			body := tcpBodyTwoVnode{}
			if err := dec.Decode(&body); err != nil {
				log.Printf("[ERR] Failed to decode TCP body! Got %s", err)
				return
			}

			// Generate a response
			obj, ok := t.get(body.Target)
			resp := tcpBodyError{}
			sendResp = &resp
			if ok {
				resp.Err = obj.ClearPredecessor(body.Vn)
			} else {
				resp.Err = fmt.Errorf("Target VN not found! Target %s:%s",
					body.Target.Host, body.Target.String())
			}

		case tcpSkipSucReq:
			body := tcpBodyTwoVnode{}
			if err := dec.Decode(&body); err != nil {
				log.Printf("[ERR] Failed to decode TCP body! Got %s", err)
				return
			}

			// Generate a response
			obj, ok := t.get(body.Target)
			resp := tcpBodyError{}
			sendResp = &resp
			if ok {
				resp.Err = obj.SkipSuccessor(body.Vn)
			} else {
				resp.Err = fmt.Errorf("Target VN not found! Target %s:%s",
					body.Target.Host, body.Target.String())
			}

		case tcpSnapshotReq:
			body := tcpBodyVnode{}
			if err := dec.Decode(&body); err != nil {
				log.Printf("[ERR] Failed to decode TCP body! Got %s", err)
				return
			}

			// Get target vnode
			obj, ok := t.get(body.Vn)
			if !ok {
				log.Printf("[snapshot] ERR Target VN not found: %s/%s", body.Vn.Host, body.Vn.String())
				return
			}

			s, err := obj.Snapshot()
			if err == nil {
				_, err = io.Copy(conn, s)
			}

			if err != nil {
				log.Printf("[snapshot] ERR %s", err)
			}
			return

		case tcpRestoreReq:
			body := tcpBodyVnode{}
			if err := dec.Decode(&body); err != nil {
				log.Printf("[ERR] Failed to decode TCP body! Got %s", err)
				return
			}

			// Get target vnode
			obj, ok := t.get(body.Vn)
			if !ok {
				log.Printf("[restore] ERR Target VN not found: %s/%s", body.Vn.Host, body.Vn.String())
				return
			}

			if err := obj.Restore(conn); err != nil {
				log.Printf("[restore] ERR %s", err)
			}
			return

		case tcpSetKeyReq:
			body := tcpBodyKeyValue{}
			if err := dec.Decode(&body); err != nil {
				log.Printf("[ERR] Failed to decode TCP body! Got %s", err)
				return
			}

			obj, ok := t.get(body.VN)
			resp := tcpBodyKeyErr{K: body.K}
			if ok {
				if err := obj.SetKey(body.K, body.V); err != nil {
					resp.Err = err.Error()
				}

			} else {
				resp.Err = fmt.Sprintf("Target VN not found! Target %s:%s",
					body.VN.Host, body.VN.String())
			}

			sendResp = &resp

		case tcpGetKeyReq:
			body := tcpBodyKey{}
			if err := dec.Decode(&body); err != nil {
				log.Printf("[ERR] Failed to decode TCP body! Got %s", err)
				return
			}

			obj, ok := t.get(body.VN)
			resp := tcpBodyKeyValueErr{K: body.K}
			if ok {
				vv, err := obj.GetKey(body.K)
				if err != nil {
					resp.Err = err.Error()
				} else {
					resp.V = vv
				}

			} else {
				resp.Err = fmt.Sprintf("Target VN not found! Target %s:%s",
					body.VN.Host, body.VN.String())
			}

			sendResp = &resp

		case tcpDelKeyReq:
			body := tcpBodyKey{}
			if err := dec.Decode(&body); err != nil {
				log.Printf("[ERR] Failed to decode TCP body! Got %s", err)
				return
			}

			obj, ok := t.get(body.VN)
			resp := tcpBodyKeyErr{K: body.K}
			if ok {
				//log.Printf("[Delete] Key: %s %+v\n", body.K, obj)
				if err := obj.DeleteKey(body.K); err != nil {
					resp.Err = err.Error()
				}

			} else {
				resp.Err = fmt.Sprintf("Target VN not found! Target %s:%s",
					body.VN.Host, body.VN.String())
			}

			sendResp = &resp

		default:
			log.Printf("[ERR] Unknown request type! Got %d", header.ReqType)
			return
		}

		// Send the response
		if err := enc.Encode(sendResp); err != nil {
			log.Printf("[ERR] Failed to send TCP body! Got %s", err)
			return
		}
	}
}

// Trims the slice to remove nil elements
func trimSlice(vn []*Vnode) []*Vnode {
	if vn == nil {
		return vn
	}
	// Find a non-nil index
	idx := len(vn) - 1
	for vn[idx] == nil {
		idx--
	}
	return vn[:idx+1]
}
