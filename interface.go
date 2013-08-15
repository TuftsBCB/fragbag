package fragbag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

type libKind string

const (
	kindStructure libKind = "structure"
	kindSequence          = "sequence"
)

func (k libKind) String() string {
	return string(k)
}

type jsonLibrary struct {
	Kind    libKind
	Library json.RawMessage
}

// Open reads a library from the reader provided. If there is a problem
// reading or parsing the data as a library, an error is returned.
// If no error is returned, the Library returned is guarnateed to be one
// of the following types of fragment libraries:
// StructureLibrary, SequenceLibrary.
func Open(r io.Reader) (Library, error) {
	var jsonlib jsonLibrary
	dec := json.NewDecoder(r)
	if err := dec.Decode(&jsonlib); err != nil {
		return nil, err
	}

	switch jsonlib.Kind {
	case kindStructure:
		return openStructureLibrary(bytes.NewReader(jsonlib.Library))
	case kindSequence:
		return openSequenceLibrary(bytes.NewReader(jsonlib.Library))
	}
	return nil, fmt.Errorf("Unknown fragment library type: %s", jsonlib.Kind)
}

// saveLibrary writes any library with the given kind as JSON data to w.
func saveLibrary(w io.Writer, kind libKind, library interface{}) error {
	jsonlib := map[string]interface{}{
		"Kind":    kind,
		"Library": library,
	}
	return niceJson(w, jsonlib)
}

// niceJson is a convenience function for encoding a JSON value and writing
// it with `json.Indent`.
func niceJson(w io.Writer, v interface{}) error {
	raw, dst := new(bytes.Buffer), new(bytes.Buffer)

	enc := json.NewEncoder(raw)
	if err := enc.Encode(v); err != nil {
		return err
	}
	if err := json.Indent(dst, raw.Bytes(), "", "\t"); err != nil {
		return err
	}
	_, err := io.Copy(w, dst)
	return err
}
