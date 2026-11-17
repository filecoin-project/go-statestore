package statestore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	cborutil "github.com/filecoin-project/go-cbor-util"
	"github.com/ipfs/go-datastore"
	cbg "github.com/whyrusleeping/cbor-gen"
)

type Flarp struct {
	x byte
}

func (f *Flarp) UnmarshalCBOR(r io.Reader) error {
	p := make([]byte, 1)
	n, err := r.Read(p)
	if n != 1 {
		panic("somebody messed up")
	}
	f.x = p[0]
	return err
}

func (f *Flarp) MarshalCBOR(w io.Writer) error {
	xs := []byte{f.x}
	_, err := w.Write(xs)
	return err
}

func (f *Flarp) Blarg() string {
	return fmt.Sprintf("%d", f.x)
}

func TestList(t *testing.T) {
	x1 := byte(64)
	x2 := byte(42)

	ds := datastore.NewMapDatastore()

	e1, err := cborutil.Dump(&Flarp{x: x1})
	if err != nil {
		t.Fatal(err)
	}

	if err := ds.Put(context.TODO(), datastore.NewKey("/2"), e1); err != nil {
		t.Fatal(err)
	}

	e2, err := cborutil.Dump(&Flarp{x: x2})
	if err != nil {
		t.Fatal(err)
	}

	if err := ds.Put(context.TODO(), datastore.NewKey("/3"), e2); err != nil {
		t.Fatal(err)
	}

	st := &StateStore{ds: ds}

	var out []Flarp
	if err := st.List(&out); err != nil {
		t.Fatal(err)
	}

	if len(out) != 2 {
		t.Fatalf("wrong len (expected %d, got %d)", 2, len(out))
	}

	blargs := make(map[string]bool)
	for _, v := range out {
		blargs[v.Blarg()] = true
	}

	if !blargs[fmt.Sprintf("%d", x1)] {
		t.Fatalf("wrong data (missing Flarp#Blarg() == %d)", x1)
	}

	if !blargs[fmt.Sprintf("%d", x2)] {
		t.Fatalf("wrong data (missing Flarp#Blarg() == %d)", x2)
	}
}

func TestListDecodeErrors(t *testing.T) {
	ds := datastore.NewMapDatastore()
	for key, data := range map[string][]byte{
		"/valid":     {0x18, 0x2a},
		"/empty":     {},
		"/truncated": {0x19, 0x00},
	} {
		if err := ds.Put(context.Background(), datastore.NewKey(key), data); err != nil {
			t.Fatal(err)
		}
	}

	var out []cbg.CborInt
	err := New(ds).List(&out)
	for _, target := range []error{io.EOF, io.ErrUnexpectedEOF} {
		if !errors.Is(err, target) {
			t.Errorf("expected List error to contain %v, got %v", target, err)
		}
	}
	if len(out) != 1 || out[0] != 42 {
		t.Fatalf("expected valid state [42], got %v", out)
	}
}
