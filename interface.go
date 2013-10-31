package fragbag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/TuftsBCB/seq"
	"github.com/TuftsBCB/structure"
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

type StructureLibrary interface {
	Library
	Best([]structure.Coords) int
	Fragment(i int) StructureFragment
}

type StructureFragment interface {
	Number() int
	String() string
	Atoms() []structure.Coords
}

type SequenceLibrary interface {
	Library
	Best(seq.Sequence) int
	Fragment(i int) SequenceFragment
}

type SequenceFragment interface {
	Number() int
	String() string
	AlignmentProb(seq.Sequence) seq.Prob
}

type libKind string

const (
	kindStructureAtoms  libKind = "structure-atoms"
	kindSequenceProfile         = "sequence-profile"
	kindSequenceHMM             = "sequence-hmm"
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

	bytesr := bytes.NewReader(jsonlib.Library)
	switch jsonlib.Kind {
	case kindStructureAtoms:
		return openStructureAtoms(bytesr)
	case kindSequenceProfile:
		return openSequenceProfile(bytesr)
	case kindSequenceHMM:
		return openSequenceHMM(bytesr)
	}
	return nil, fmt.Errorf("Unknown fragment library type: %s", jsonlib.Kind)
}

// IsSequence returns true if the given library is a sequence fragment library.
// Returns false otherwise.
func IsSequence(lib Library) bool {
	_, ok := lib.(SequenceLibrary)
	return ok
}

// IsStructure returns true if the given library is a structure fragment
// library. Returns false otherwise.
func IsStructure(lib Library) bool {
	_, ok := lib.(StructureLibrary)
	return ok
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
