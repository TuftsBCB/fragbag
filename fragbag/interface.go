package fragbag

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
)

type Library interface {
	// Save should save the fragment library to the writer provided.
	// No particular format is specified by the public API.
	Save(w io.Writer) error

	// Size returns the number of fragments in the library.
	Size() int

	// FragmentSize returns the size of every fragment in the library.
	// All fragments in a library must have the same size.
	FragmentSize() int

	// String returns a custom string representation of the library.
	// This may be anything.
	String() string

	// Name returns a canonical name for this fragment library.
	Name() string
}

type libKind byte

const (
	kindStructure libKind = iota
	kindSequence
)

func (k libKind) String() string {
	switch k {
	case kindStructure:
		return "structure"
	case kindSequence:
		return "sequence"
	}
	panic(fmt.Sprintf("Unknown fragment library type: %d", k))
}

// Open reads a library from the reader provided. If there is a problem
// reading or parsing the data as a library, an error is returned.
// If no error is returned, the Library returned is guarnateed to be one
// of the following types of fragment libraries:
// StructureLibrary, SequenceLibrary.
func Open(r io.Reader) (Library, error) {
	all, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	r = bytes.NewReader(all)
	switch libKind(all[0]) {
	case kindStructure:
		return OpenStructureLibrary(r)
	case kindSequence:
		return OpenSequenceLibrary(r)
	}
	return nil, fmt.Errorf("Unknown fragment library type: %d", all[0])
}

// writeKind writes the kind specified to the writer given.
// The library is used to provided a descriptive error message if things go
// bad.
func writeKind(w io.Writer, lib Library, k libKind) error {
	errPrefix := fmt.Sprintf("Error while saving '%s':", lib.Name())
	if n, err := w.Write([]byte{byte(k)}); err != nil {
		return fmt.Errorf("%s %s", errPrefix, err)
	} else if n != 1 {
		return fmt.Errorf("%s Wrote %d bytes instead of 1 byte.", errPrefix, n)
	}
	return nil
}

// readKind reads a single byte from the reader and returns an error if it
// does not match the kind expected.
func readKind(r io.Reader, expected libKind) error {
	kindBytes := make([]byte, 1)
	if n, err := r.Read(kindBytes); err != nil {
		return err
	} else if n != 1 {
		return fmt.Errorf("Error reading fragment library type: "+
			"Read %d bytes instead of 1 byte.", n)
	}

	kind := libKind(kindBytes[0])
	if kind != expected {
		return fmt.Errorf("Expected a fragment library with type '%s', "+
			"but got '%s' instead.", expected, kind)
	}
	return nil
}
