package fragbag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Tags for libraries defined in this library.
const (
	libTagStructureAtoms  = "structure-atoms"
	libTagSequenceProfile = "sequence-profile"
	libTagSequenceHMM     = "sequence-hmm"
	libTagWeightedTfIdf   = "weighted-tfidf"
)

// MakeEmptyLib represents a function that returns an empty value whose type
// implements the Library interface. This is used inside the Open function.
// Namely, when a fragment library file is opened, its tag is used to look up
// a function with this type in the Openers map. Once the empty value is
// retrieved, it is initialized with data from the fragment library file.
//
// The subTags parameter is used when opening a library which wraps another
// library. Namely, it will contain all tags of libraries within it.
// It will be empty for a library that doesn't wrap another library.
type MakeEmptyLib func(subTags ...string) (Library, error)

// Openers stores initializers for each type of fragment library. The keys
// should be values returned by the Tag method in the Library interface.
// Clients may add to this map, which will enable the Open function in this
// package to return your custom libraries. (N.B. This is not required if you
// don't want to use the Open function.)
var Openers map[string]MakeEmptyLib

func init() {
	Openers = make(map[string]MakeEmptyLib, 10)

	// Add pre-defined libraries to opener dispatcher.
	Openers[libTagStructureAtoms] = func(...string) (Library, error) {
		return &structureAtoms{}, nil
	}
	Openers[libTagSequenceProfile] = func(...string) (Library, error) {
		return &sequenceProfile{}, nil
	}
	Openers[libTagSequenceHMM] = func(...string) (Library, error) {
		return &sequenceHMM{}, nil
	}
	Openers[libTagWeightedTfIdf] = makeWeightedTfIdf
}

// Open reads a library from the reader provided. If there is a problem
// reading or parsing the data as a library, an error is returned.
// If no error is returned, the Library returned is guarnateed to satisfy
// either the StructureLibrary or SequenceLibrary interfaces.
// It is possible that a wrapper library is returned which satisfy both the
// StructureLibrary and SequenceLibrary interfaces. This type of library can
// be inspected with the SubLibrary interface method, along with the IsStructure
// and IsSequence functions in this module.
func Open(r io.Reader) (Library, error) {
	type jsonLibrary struct {
		Tags    []string
		Library json.RawMessage
	}

	var jsonlib jsonLibrary
	dec := json.NewDecoder(r)
	if err := dec.Decode(&jsonlib); err != nil {
		return nil, err
	}
	if len(jsonlib.Tags) == 0 {
		return nil, fmt.Errorf("Corrupt fragment library. No tags founds.")
	}

	empty, err := makeEmptySubLibrary(jsonlib.Tags...)
	if err != nil {
		return nil, err
	}

	dec = json.NewDecoder(bytes.NewReader(jsonlib.Library))
	if err := dec.Decode(&empty); err != nil {
		return nil, err
	}
	return empty, nil
}

// makeEmptySubLibrary recursively dispatches the opener functions so that
// libraries with sub-libraries are reconstructed with the right types.
func makeEmptySubLibrary(subTags ...string) (Library, error) {
	subOpener := Openers[subTags[0]]
	if subOpener == nil {
		return nil, fmt.Errorf("Unrecognized library tag '%s'.", subTags[0])
	}

	var empty Library
	var err error
	if len(subTags) > 1 {
		empty, err = subOpener(subTags[1:]...)
	} else {
		empty, err = subOpener()
	}
	if err != nil {
		return nil, err
	}
	return empty, nil
}

// Save stores the given fragment library with the writer provided.
func Save(w io.Writer, lib Library) error {
	return niceJson(w, map[string]interface{}{
		"Tags":    fullTag(lib),
		"Library": lib,
	})
}

// IsSequence returns true if the given library is a sequence fragment library.
// Returns false otherwise.
// This also works on wrapped libraries. Namely, it will be recursively called
// on sub libraries.
func IsSequence(lib Library) bool {
	if sub := lib.SubLibrary(); sub != nil {
		return IsSequence(sub)
	}
	_, ok := lib.(SequenceLibrary)
	return ok
}

// IsStructure returns true if the given library is a structure fragment
// library. Returns false otherwise.
// This also works on wrapped libraries. Namely, it will be recursively called
// on sub libraries.
func IsStructure(lib Library) bool {
	if sub := lib.SubLibrary(); sub != nil {
		return IsStructure(sub)
	}
	_, ok := lib.(StructureLibrary)
	return ok
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

// fullTag returns the complete tag for a particular library, including
// information from its sub libraries.
func fullTag(lib Library) []string {
	if sub := lib.SubLibrary(); sub == nil {
		return []string{lib.Tag()}
	} else {
		return append([]string{lib.Tag()}, fullTag(sub)...)
	}
}
