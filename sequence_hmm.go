package fragbag

import (
	"fmt"

	"github.com/TuftsBCB/seq"
)

var _ = SequenceLibrary(&sequenceHMM{})

// sequenceHMM represents a Fragbag sequence fragment library.
// Fragbag fragment libraries are fixed both in the number of fragments and in
// the size of each fragment.
type sequenceHMM struct {
	Ident     string
	Fragments []sequenceHMMFrag
	FragSize  int
}

// Fragment corresponds to a single sequence fragment in a fragment library.
// It holds the fragment number identifier and embeds an HMM.
type sequenceHMMFrag struct {
	FragNumber int
	*seq.HMM
}

// NewSequenceHMM initializes a new Fragbag sequence library with the
// given name and fragments.
//
// Fragments for this library are represented as profile HMMs. Computing the
// best fragment for any particular sequence uses the score produced by
// Viterbi.
func NewSequenceHMM(
	name string,
	fragments []*seq.HMM,
) (SequenceLibrary, error) {
	lib := new(sequenceHMM)
	lib.Ident = name
	for _, frag := range fragments {
		if err := lib.add(frag); err != nil {
			return nil, err
		}
	}
	return lib, nil
}

func (lib *sequenceHMM) SubLibrary() Library {
	return nil
}

// Add adds a sequence fragment to the library, where a sequence fragment
// corresponds to an HMM.
//
// The first time Add is called, the HMM may have any number of nodes. All
// subsequent calls to Add must supply an HMM with the number of nodes
// equal to the first HMM added.
func (lib *sequenceHMM) add(hmm *seq.HMM) error {
	if lib.Fragments == nil || len(lib.Fragments) == 0 {
		frag := sequenceHMMFrag{0, hmm}
		lib.Fragments = append(lib.Fragments, frag)
		lib.FragSize = len(hmm.Nodes)
		return nil
	}

	frag := sequenceHMMFrag{len(lib.Fragments), hmm}
	if lib.FragSize != len(hmm.Nodes) {
		return fmt.Errorf("Fragment %d has length %d; expected length %d.",
			frag.FragNumber, len(hmm.Nodes), lib.FragSize)
	}
	lib.Fragments = append(lib.Fragments, frag)
	return nil
}

func (lib *sequenceHMM) Tag() string {
	return libTagSequenceHMM
}

// Size returns the number of fragments in the library.
func (lib *sequenceHMM) Size() int {
	return len(lib.Fragments)
}

// FragmentSize returns the size of every fragment in the library.
func (lib *sequenceHMM) FragmentSize() int {
	return lib.FragSize
}

// String returns a string with the name of the library, the number of
// fragments in the library and the size of each fragment.
func (lib *sequenceHMM) String() string {
	return fmt.Sprintf("%s (%d, %d)",
		lib.Ident, len(lib.Fragments), lib.FragSize)
}

func (lib *sequenceHMM) Name() string {
	return lib.Ident
}

// Best returns the number of the fragment that best corresponds
// to the string of amino acids provided.
// The length of `sequence` must be equivalent to the fragment size.
//
// If no "good" fragments can be found, then `-1` is returned. This
// behavior will almost certainly change in the future.
func (lib *sequenceHMM) BestSequenceFragment(s seq.Sequence) int {
	if s.Len() != lib.FragmentSize() {
		panic(fmt.Sprintf("Sequence length %d != fragment size %d",
			s.Len(), lib.FragmentSize()))
	}
	var testAlign seq.Prob
	dynamicTable := seq.AllocTable(lib.FragmentSize(), s.Len())
	bestAlign, bestFragNum := seq.MinProb, -1
	for _, frag := range lib.Fragments {
		testAlign = frag.ViterbiScoreMem(s, dynamicTable)
		if bestAlign.Less(testAlign) {
			bestAlign, bestFragNum = testAlign, frag.FragNumber
		}
	}
	return bestFragNum
}

// AlignmentProb computes the probability of the sequence `s` aligning
// with the HMM in `frag`. The sequence must have length equivalent
// to the fragment size.
func (lib *sequenceHMM) AlignmentProb(fragi int, s seq.Sequence) seq.Prob {
	frag := lib.Fragments[fragi]
	if s.Len() != len(frag.Nodes) {
		panic(fmt.Sprintf("Sequence length %d != fragment size %d",
			s.Len(), len(frag.Nodes)))
	}
	return frag.ViterbiScore(s)
}

func (lib *sequenceHMM) FragmentString(fragNum int) string {
	return fmt.Sprintf("> %d\n%s", fragNum, lib.Fragments[fragNum].HMM)
}
