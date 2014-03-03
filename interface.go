package fragbag

import (
	"github.com/TuftsBCB/seq"
	"github.com/TuftsBCB/structure"
)

// Library defines the base methods necessary for any value to be considered
// a fragment library. All libraries that do *not* wrap another library should
// implement either the Structure or Sequence library interfaces and never
// both.
type Library interface {
	// Name returns a canonical name for this fragment library.
	Name() string

	// Size returns the number of fragments in the library.
	Size() int

	// FragmentSize returns the size of every fragment in the library.
	// All fragments in a library must have the same size.
	FragmentSize() int

	// Tag returns a uniquely identifying string for this type of fragment
	// library. It is used to dispatch on when opening a fragment library.
	Tag() string

	// SubLibrary returns a library contained inside of this one and returns
	// nil otherwise. When non-nil, this library is a wrapper library which
	// may implement both the StructureLibrary and SequenceLibrary interfaces.
	// When nil, it is guaranteed that only one of the interfaces will be
	// satisfied.
	SubLibrary() Library

	// String returns a custom string representation of the library.
	// This may be anything.
	String() string

	// FragmentString returns a custom string representation of the
	// given fragment.
	FragmentString(fragNum int) string

	// Fragment returns a representation of the sequence fragment
	// corresponding to fragNum. The representation is specific to the
	// library.
	Fragment(fragNum int) interface{}
}

// StructureLibrary adds methods specific to the operations defined on a
// library of structure fragments.
type StructureLibrary interface {
	Library

	// BestStructureFragment returns the fragment number of the best matching
	// fragment against the alpha-carbon coordinates given. Note that there
	// must be N coordinates where N is the size of each fragment in this
	// library.
	//
	// If no "good" fragments can be found, then `-1` is returned.
	BestStructureFragment([]structure.Coords) int

	// Atoms returns a list of alpha-carbon coordinates for a particular
	// fragment.
	Atoms(fragNum int) []structure.Coords
}

// SequenceLibrary adds methods specific to the operations defined on a
// library of sequence fragments.
type SequenceLibrary interface {
	Library

	// BestSequenceFragment returns the fragment number of the best matching
	// fragment against the sequence given. Note that the sequence given must
	// have length N where N is the size of each fragment in this library.
	//
	// If no "good" fragments can be found, then `-1` is returned.
	BestSequenceFragment(seq.Sequence) int

	// AlignmentProb returns the probability (as a negative log-odds) that
	// a query sequence matches a particular fragment.
	AlignmentProb(fragNum int, query seq.Sequence) seq.Prob
}

// WeightedLibrary adds methods specific to the operations defined on a
// library of weighted fragments.
type WeightedLibrary interface {
	Library

	// AddWeights turns a raw frequency into a weighted frequency. The
	// frequency given should be related to the fragment given. (e.g., The
	// frequency is the number of times the fragment appeared in a particular
	// query.)
	AddWeights(fragNum int, frequency float32) float32
}
