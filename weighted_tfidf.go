package fragbag

import (
	"fmt"

	"github.com/TuftsBCB/seq"
	"github.com/TuftsBCB/structure"
)

var (
	_ = WeightedLibrary(&weightedTfIdf{})
	_ = StructureLibrary(&weightedTfIdf{})
	_ = SequenceLibrary(&weightedTfIdf{})
)

// weightedTfIdf wraps any fragment library so that all BOWs are weighted
// according to a simple tf-idf scheme.
//
// A weightedTfIdf can satisfy either the Structure or Sequence library
// interfaces, but only one will work, depending upon the underlying value
// of the wrapped library.
type weightedTfIdf struct {
	Library
	FragIDFs []float32
}

// NewWeightedTfIdf wraps any fragment library and stores a list of inverse
// document frequencies for each fragment in the wrapped library.
//
// Note that this library satisfies both the Structure and Sequence library
// interfaces.
//
// When computing a BOW from this library, the AddWeights method should be
// applied to the regular unweighted BOW. Note that this is done for you if
// you're using the bow sub-package.
func NewWeightedTfIdf(lib Library, idfs []float32) (WeightedLibrary, error) {
	if len(idfs) != lib.Size() {
		return nil, fmt.Errorf("Cannot wrap library with weights since the "+
			"library has %d fragments but %d weights were given.",
			lib.Size(), len(idfs))
	}
	return &weightedTfIdf{lib, idfs}, nil
}

func (lib *weightedTfIdf) SubLibrary() Library {
	return lib.Library
}

// AddWeights returns the tf-idf weight given the frequency of a particular
// fragment. The idf portion of the computation is already computed as part
// of the representation of the underlying fragment library.
func (lib *weightedTfIdf) AddWeights(fragNum int, frequency float32) float32 {
	return frequency * lib.FragIDFs[fragNum]
}

func (lib *weightedTfIdf) Tag() string {
	return libTagWeightedTfIdf
}

func makeWeightedTfIdf(subTags ...string) (Library, error) {
	if len(subTags) == 0 {
		return nil, fmt.Errorf("The weighted-tfidf fragment library must " +
			"have a sub-tag specified for its sub fragment library.")
	}
	empty, err := makeEmptySubLibrary(subTags...)
	if err != nil {
		return nil, err
	}
	return &weightedTfIdf{empty, nil}, nil
}

// BestStructureFragment calls the corresponding method on the underlying
// fragment library.
func (lib *weightedTfIdf) BestStructureFragment(atoms []structure.Coords) int {
	return lib.Library.(StructureLibrary).BestStructureFragment(atoms)
}

// Atoms calls the corresponding method on the underlying fragment library.
func (lib *weightedTfIdf) Atoms(fragNum int) []structure.Coords {
	return lib.Library.(StructureLibrary).Atoms(fragNum)
}

// BestSequenceFragment calls the corresponding method on the underlying
// fragment library.
func (lib *weightedTfIdf) BestSequenceFragment(s seq.Sequence) int {
	return lib.Library.(SequenceLibrary).BestSequenceFragment(s)
}

// AlignmentProb calls the corresponding method on the underlying fragment
// library.
func (lib *weightedTfIdf) AlignmentProb(fragNum int, s seq.Sequence) seq.Prob {
	return lib.Library.(SequenceLibrary).AlignmentProb(fragNum, s)
}
