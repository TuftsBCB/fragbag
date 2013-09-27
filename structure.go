package fragbag

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/TuftsBCB/structure"
)

// StructureAtoms represents a Fragbag structural fragment library.
// Fragbag fragment libraries are fixed both in the number of fragments and in
// the size of each fragment.
type StructureAtoms struct {
	Ident     string
	Fragments []StructureAtomsFrag
	FragSize  int
}

// NewStructureAtoms initializes a new Fragbag structure library with the
// given name. It is not written to disk until Save is called.
func NewStructureAtoms(name string) *StructureAtoms {
	lib := new(StructureAtoms)
	lib.Ident = name
	return lib
}

// Add adds a structural fragment to the library. The first call to Add may
// contain any number of coordinates. All subsequent adds must contain the
// same number of coordinates as the first.
func (lib *StructureAtoms) Add(coords []structure.Coords) error {
	if lib.Fragments == nil || len(lib.Fragments) == 0 {
		frag := StructureAtomsFrag{0, coords}
		lib.Fragments = append(lib.Fragments, frag)
		lib.FragSize = len(coords)
		return nil
	}

	frag := StructureAtomsFrag{len(lib.Fragments), coords}
	if lib.FragSize != len(coords) {
		return fmt.Errorf("Fragment %d has length %d; expected length %d.",
			frag.Number(), len(coords), lib.FragSize)
	}
	lib.Fragments = append(lib.Fragments, frag)
	return nil
}

// Save saves the full fragment library to the writer provied.
func (lib *StructureAtoms) Save(w io.Writer) error {
	return saveLibrary(w, kindStructureAtoms, lib)
}

// Open loads an existing structure fragment library from the reader provided.
func openStructureAtoms(r io.Reader) (*StructureAtoms, error) {
	var lib *StructureAtoms
	dec := json.NewDecoder(r)
	if err := dec.Decode(&lib); err != nil {
		return nil, err
	}
	return lib, nil
}

// Size returns the number of fragments in the library.
func (lib *StructureAtoms) Size() int {
	return len(lib.Fragments)
}

// FragmentSize returns the size of every fragment in the library.
func (lib *StructureAtoms) FragmentSize() int {
	return lib.FragSize
}

// Fragment returns the ith fragment in the library (starting from 0).
func (lib *StructureAtoms) Fragment(i int) StructureFragment {
	return lib.Fragments[i]
}

// String returns a string with the name of the library, the number of
// fragments in the library and the size of each fragment.
func (lib *StructureAtoms) String() string {
	return fmt.Sprintf("%s (%d, %d)",
		lib.Ident, len(lib.Fragments), lib.FragSize)
}

func (lib *StructureAtoms) Name() string {
	return lib.Ident
}

// rmsdMemory creates reusable memory for use with RMSD calculation with
// suitable size for this fragment library. Only one goroutine can use the
// memory at a time.
func (lib *StructureAtoms) rmsdMemory() structure.Memory {
	return structure.NewMemory(lib.FragSize)
}

// Best returns the number of the fragment that best corresponds
// to the region of atoms provided.
// The length of `atoms` must be equivalent to the fragment size.
func (lib *StructureAtoms) Best(atoms []structure.Coords) int {
	return lib.bestMem(atoms, lib.rmsdMemory())
}

// BestMem returns the number of the fragment that best corresponds
// to the region of atoms provided without allocating.
// The length of `atoms` must be equivalent to the fragment size.
//
// `mem` must be a region of reusable memory that should only be accessed
// from one goroutine at a time. Valid values can be constructed with
// rmsdMemory.
func (lib *StructureAtoms) bestMem(
	atoms []structure.Coords,
	mem structure.Memory,
) int {
	var testRmsd float64
	bestRmsd, bestFragNum := 0.0, -1
	for _, frag := range lib.Fragments {
		testRmsd = structure.RMSDMem(mem, atoms, frag.FragAtoms)
		if bestFragNum == -1 || testRmsd < bestRmsd {
			bestRmsd, bestFragNum = testRmsd, frag.FragNumber
		}
	}
	return bestFragNum
}

// Fragment corresponds to a single structural fragment in a fragment library.
// It holds the fragment number identifier and the 3 dimensional coordinates.
type StructureAtomsFrag struct {
	FragNumber int
	FragAtoms  []structure.Coords
}

func (frag StructureAtomsFrag) Number() int {
	return frag.FragNumber
}

func (frag StructureAtomsFrag) Atoms() []structure.Coords {
	return frag.FragAtoms
}

// String returns the fragment number, library and its corresponding atoms.
func (frag StructureAtomsFrag) String() string {
	atoms := make([]string, len(frag.FragAtoms))
	for i, atom := range frag.FragAtoms {
		atoms[i] = fmt.Sprintf("\t%s", atom)
	}
	return fmt.Sprintf("> %d\n%s", frag.FragNumber, strings.Join(atoms, "\n"))
}
