package fragbag

import (
	"fmt"
	"strings"

	"github.com/TuftsBCB/structure"
)

var _ = StructureLibrary(&structureAtoms{})

// structureAtoms represents a Fragbag structural fragment library.
// Fragbag fragment libraries are fixed both in the number of fragments and in
// the size of each fragment.
type structureAtoms struct {
	Ident     string
	Fragments []structureAtomsFrag
	FragSize  int
}

// Fragment corresponds to a single structural fragment in a fragment library.
// It holds the fragment number identifier and the 3 dimensional coordinates.
type structureAtomsFrag struct {
	FragNumber int
	FragAtoms  []structure.Coords
}

// NewStructureAtoms initializes a new Fragbag structure library with the
// given name and fragments. All fragments given must have exactly the same
// size.
func NewStructureAtoms(
	name string,
	fragments [][]structure.Coords,
) (StructureLibrary, error) {
	lib := new(structureAtoms)
	lib.Ident = name
	for _, frag := range fragments {
		if err := lib.add(frag); err != nil {
			return nil, err
		}
	}
	return lib, nil
}

func (lib *structureAtoms) SubLibrary() Library {
	return nil
}

// add adds a structural fragment to the library. The first call to Add may
// contain any number of coordinates. All subsequent adds must contain the
// same number of coordinates as the first.
func (lib *structureAtoms) add(coords []structure.Coords) error {
	if lib.Fragments == nil || len(lib.Fragments) == 0 {
		frag := structureAtomsFrag{0, coords}
		lib.Fragments = append(lib.Fragments, frag)
		lib.FragSize = len(coords)
		return nil
	}

	frag := structureAtomsFrag{len(lib.Fragments), coords}
	if lib.FragSize != len(coords) {
		return fmt.Errorf("Fragment %d has length %d; expected length %d.",
			frag.FragNumber, len(coords), lib.FragSize)
	}
	lib.Fragments = append(lib.Fragments, frag)
	return nil
}

func (lib *structureAtoms) Tag() string {
	return libTagStructureAtoms
}

// Size returns the number of fragments in the library.
func (lib *structureAtoms) Size() int {
	return len(lib.Fragments)
}

// FragmentSize returns the size of every fragment in the library.
func (lib *structureAtoms) FragmentSize() int {
	return lib.FragSize
}

// String returns a string with the name of the library, the number of
// fragments in the library and the size of each fragment.
func (lib *structureAtoms) String() string {
	return fmt.Sprintf("%s (%d, %d)",
		lib.Ident, len(lib.Fragments), lib.FragSize)
}

func (lib *structureAtoms) Name() string {
	return lib.Ident
}

// rmsdMemory creates reusable memory for use with RMSD calculation with
// suitable size for this fragment library. Only one goroutine can use the
// memory at a time.
func (lib *structureAtoms) rmsdMemory() structure.Memory {
	return structure.NewMemory(lib.FragSize)
}

// Best returns the number of the fragment that best corresponds
// to the region of atoms provided.
// The length of `atoms` must be equivalent to the fragment size.
func (lib *structureAtoms) BestStructureFragment(atoms []structure.Coords) int {
	var testRmsd float64
	mem := lib.rmsdMemory()
	bestRmsd, bestFragNum := 0.0, -1
	for _, frag := range lib.Fragments {
		testRmsd = structure.RMSDMem(mem, atoms, frag.FragAtoms)
		if bestFragNum == -1 || testRmsd < bestRmsd {
			bestRmsd, bestFragNum = testRmsd, frag.FragNumber
		}
	}
	return bestFragNum
}

func (lib *structureAtoms) Atoms(fragNum int) []structure.Coords {
	return lib.Fragments[fragNum].FragAtoms
}

// String returns the fragment number, library and its corresponding atoms.
func (lib *structureAtoms) FragmentString(fragNum int) string {
	atoms := lib.Atoms(fragNum)
	satoms := make([]string, len(atoms))
	for i, atom := range atoms {
		satoms[i] = fmt.Sprintf("\t%s", atom)
	}
	return fmt.Sprintf("> %d\n%s", fragNum, strings.Join(satoms, "\n"))
}
