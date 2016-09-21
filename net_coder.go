package chord

import (
	"encoding/binary"
	"fmt"
	"io"

	"gopkg.in/vmihailenco/msgpack.v2"
)

//
// This file contains the used decoder and encoder used for network serialization.
//

type Encoder interface {
	Encode(interface{}) error
}
type Decoder interface {
	Decode(interface{}) error
}

func NewEncoder(w io.Writer) Encoder {
	//return gob.NewEncoder(w)
	return &encoder{w: w}
}
func NewDecoder(r io.Reader) Decoder {
	//return gob.NewDecoder(r)
	return &decoder{r: r}
}

type encoder struct {
	w io.Writer
}

func (e *encoder) Encode(v interface{}) error {
	b, err := msgpack.Marshal(v)
	if err == nil {
		err = e.writeData(b)
	}
	return err
}

// Write data first writing the size of the payload then the payload
// itself.
func (e *encoder) writeData(data []byte) error {
	// Write 8 byte payload size
	plsize := uint64(len(data))
	err := binary.Write(e.w, binary.LittleEndian, plsize)
	if err == nil {
		// write data
		_, err = e.w.Write(data)
	}
	return err
}

type decoder struct {
	r io.Reader
}

func (d *decoder) Decode(v interface{}) error {
	data, err := d.readNext()
	if err == nil {
		err = msgpack.Unmarshal(data, v)
	}
	return err
}

func (d *decoder) readNext() ([]byte, error) {
	bsize := 8
	plsize, err := d.readPayloadSize(bsize)
	if err == nil {
		var (
			buff = make([]byte, plsize)
			n    int
		)

		if n, err = d.r.Read(buff); err == nil {
			if uint64(n) != plsize {
				return nil, fmt.Errorf("invalid payload size: have=%d want=%d", n, plsize)
			}
			return buff, nil
		}
	}

	return nil, err
}

// Read first 8 bytes for each dataset to determine dataset size
// TODO: add byte size argumen (i.e. 1, 2, 4, 8 byte payload header size)
func (d *decoder) readPayloadSize(bsize int) (uint64, error) {

	b := make([]byte, bsize)
	n, err := d.r.Read(b)
	if err != nil {
		return 0, err
	} else if n != bsize {
		return 0, fmt.Errorf("invalid payload size")
	}

	return binary.LittleEndian.Uint64(b), nil
}
