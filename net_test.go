package chord

import (
	"fmt"
	"testing"
	"time"

	"gopkg.in/vmihailenco/msgpack.v2"
)

func prepRing2(port int) (*Config, *TCPTransport, error) {
	listen := fmt.Sprintf("127.0.0.1:%d", port)
	conf := DefaultConfig(listen)
	conf.StabilizeMin = time.Duration(15 * time.Millisecond)
	conf.StabilizeMax = time.Duration(45 * time.Millisecond)
	timeout := time.Duration(20 * time.Millisecond)
	trans, err := InitTCPTransport(listen, timeout)
	if err != nil {
		return nil, nil, err
	}
	return conf, trans, nil
}

func getVnodes(rn *Ring) []Vnode {
	vns := make([]Vnode, len(rn.vnodes))
	i := 0
	for _, v := range rn.vnodes {
		vns[i] = v.Vnode
		i++
	}
	return vns
}

func TestTCPJoin(t *testing.T) {
	// Prepare to create 2 nodes
	c1, t1, err := prepRing2(10035)
	if err != nil {
		t.Fatalf("unexpected err. %s", err)
	}
	c2, t2, err := prepRing2(10036)
	if err != nil {
		t.Fatalf("unexpected err. %s", err)
	}

	// Create initial ring
	r1, err := Create(c1, t1, nil)
	if err != nil {
		t.Fatalf("unexpected err. %s", err)
	}

	// Join ring
	r2, err := Join(c2, t2, nil, c1.Hostname)
	if err != nil {
		t.Fatalf("failed to join local node! Got %s", err)
	}

	// do requests
	vn := getVnodes(r1)[0]
	// set
	data, _ := msgpack.Marshal(map[string]interface{}{"id": "test"})
	if err := t1.SetKey(&vn, []byte("test"), data); err != nil {
		//t.Log("should have failed to set key")
		t.Fatal(err)
	}
	// get

	v, err := t1.GetKey(&vn, []byte("test"))
	if err != nil {
		t.Log(err)
		t.Fail()
	} else {
		t.Logf("%s", v)
	}
	// Shutdown
	r1.Shutdown()
	r2.Shutdown()
	t1.Shutdown()
	t2.Shutdown()
}

func TestTCPLeave(t *testing.T) {
	// Prepare to create 2 nodes
	c1, t1, err := prepRing2(10037)
	if err != nil {
		t.Fatalf("unexpected err. %s", err)
	}
	c2, t2, err := prepRing2(10038)
	if err != nil {
		t.Fatalf("unexpected err. %s", err)
	}

	// Create initial ring
	r1, err := Create(c1, t1, nil)
	if err != nil {
		t.Fatalf("unexpected err. %s", err)
	}

	// Join ring
	r2, err := Join(c2, t2, nil, c1.Hostname)
	if err != nil {
		t.Fatalf("failed to join local node! Got %s", err)
	}

	// Wait for some stabilization
	<-time.After(100 * time.Millisecond)

	// Node 1 should leave
	r1.Leave()
	t1.Shutdown()

	// Wait for stabilization
	<-time.After(100 * time.Millisecond)

	// Verify r2 ring is still in tact
	for _, vn := range r2.vnodes {
		if vn.successors[0].Host != r2.config.Hostname {
			t.Fatalf("bad successor! Got:%s:%s", vn.successors[0].Host,
				vn.successors[0])
		}
	}
}
