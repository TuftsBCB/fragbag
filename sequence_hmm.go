package fragbag

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/TuftsBCB/seq"
)

// SequenceHMM represents a Fragbag sequence fragment library.
// Fragbag fragment libraries are fixed both in the number of fragments and in
// the size of each fragment.
type SequenceHMM struct {
	Ident     string
	Fragments []SequenceHMMFrag
	FragSize  int
}

// NewSequenceHMM initializes a new Fragbag sequence library with the
// given name. It is not written to disk until Save is called.
func NewSequenceHMM(name string) *SequenceHMM {
	lib := new(SequenceHMM)
	lib.Ident = name
	return lib
}

// Add adds a sequence fragment to the library, where a sequence fragment
// corresponds to an HMM.
//
// The first time Add is called, the HMM may have any number of nodes. All
// subsequent calls to Add must supply an HMM with the number of nodes
// equal to the first HMM added.
func (lib *SequenceHMM) Add(hmm *seq.HMM) error {
	if lib.Fragments == nil || len(lib.Fragments) == 0 {
		frag := SequenceHMMFrag{0, hmm}
		lib.Fragments = append(lib.Fragments, frag)
		lib.FragSize = len(hmm.Nodes)
		return nil
	}

	frag := SequenceHMMFrag{len(lib.Fragments), hmm}
	if lib.FragSize != len(hmm.Nodes) {
		return fmt.Errorf("Fragment %d has length %d; expected length %d.",
			frag.Number(), len(hmm.Nodes), lib.FragSize)
	}
	lib.Fragments = append(lib.Fragments, frag)
	return nil
}

// Save saves the full fragment library to the writer provied.
func (lib *SequenceHMM) Save(w io.Writer) error {
	return saveLibrary(w, kindSequenceHMM, lib)
}

// Open loads an existing structure fragment library from the reader provided.
func openSequenceHMM(r io.Reader) (*SequenceHMM, error) {
	var lib *SequenceHMM
	dec := json.NewDecoder(r)
	if err := dec.Decode(&lib); err != nil {
		return nil, err
	}
	return lib, nil
}

// Size returns the number of fragments in the library.
func (lib *SequenceHMM) Size() int {
	return len(lib.Fragments)
}

// FragmentSize returns the size of every fragment in the library.
func (lib *SequenceHMM) FragmentSize() int {
	return lib.FragSize
}

// Fragment returns the ith fragment in this library (starting from 0).
func (lib *SequenceHMM) Fragment(i int) SequenceFragment {
	return lib.Fragments[i]
}

// String returns a string with the name of the library, the number of
// fragments in the library and the size of each fragment.
func (lib *SequenceHMM) String() string {
	return fmt.Sprintf("%s (%d, %d)",
		lib.Ident, len(lib.Fragments), lib.FragSize)
}

func (lib *SequenceHMM) Name() string {
	return lib.Ident
}

// Best returns the number of the fragment that best corresponds
// to the string of amino acids provided.
// The length of `sequence` must be equivalent to the fragment size.
//
// If no "good" fragments can be found, then `-1` is returned. This
// behavior will almost certainly change in the future.
func (lib *SequenceHMM) Best(s seq.Sequence) int {
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

// Fragment corresponds to a single sequence fragment in a fragment library.
// It holds the fragment number identifier and embeds an HMM.
type SequenceHMMFrag struct {
	FragNumber int
	*seq.HMM
}

// AlignmentProb computes the probability of the sequence `s` aligning
// with the HMM in `frag`. The sequence must have length equivalent
// to the fragment size.
func (frag SequenceHMMFrag) AlignmentProb(s seq.Sequence) seq.Prob {
	if s.Len() != len(frag.Nodes) {
		panic(fmt.Sprintf("Sequence length %d != fragment size %d",
			s.Len(), len(frag.Nodes)))
	}
	return frag.ViterbiScore(s)
}

func (frag SequenceHMMFrag) Number() int {
	return frag.FragNumber
}

func (frag SequenceHMMFrag) String() string {
	return fmt.Sprintf("> %d\n%s", frag.FragNumber, frag.HMM)
}
