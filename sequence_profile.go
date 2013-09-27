package fragbag

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/TuftsBCB/seq"
)

// SequenceProfile represents a Fragbag sequence fragment library.
// Fragbag fragment libraries are fixed both in the number of fragments and in
// the size of each fragment.
type SequenceProfile struct {
	Ident     string
	Fragments []SequenceProfileFrag
	FragSize  int
}

// NewSequenceProfile initializes a new Fragbag sequence library with the
// given name. It is not written to disk until Save is called.
func NewSequenceProfile(name string) *SequenceProfile {
	lib := new(SequenceProfile)
	lib.Ident = name
	return lib
}

// Add adds a sequence fragment to the library, where a sequence fragment
// corresponds to a profile of log-odds scores for each amino acid.
// The first call to Add may contain any number of columns in the profile.
// All subsequent adds must contain the same number of columns as the first.
func (lib *SequenceProfile) Add(prof *seq.Profile) error {
	if lib.Fragments == nil || len(lib.Fragments) == 0 {
		frag := SequenceProfileFrag{0, prof}
		lib.Fragments = append(lib.Fragments, frag)
		lib.FragSize = prof.Len()
		return nil
	}

	frag := SequenceProfileFrag{len(lib.Fragments), prof}
	if lib.FragSize != prof.Len() {
		return fmt.Errorf("Fragment %d has length %d; expected length %d.",
			frag.Number(), prof.Len(), lib.FragSize)
	}
	lib.Fragments = append(lib.Fragments, frag)
	return nil
}

// Save saves the full fragment library to the writer provied.
func (lib *SequenceProfile) Save(w io.Writer) error {
	return saveLibrary(w, kindSequenceProfile, lib)
}

// Open loads an existing structure fragment library from the reader provided.
func openSequenceProfile(r io.Reader) (*SequenceProfile, error) {
	var lib *SequenceProfile
	dec := json.NewDecoder(r)
	if err := dec.Decode(&lib); err != nil {
		return nil, err
	}
	return lib, nil
}

// Size returns the number of fragments in the library.
func (lib *SequenceProfile) Size() int {
	return len(lib.Fragments)
}

// FragmentSize returns the size of every fragment in the library.
func (lib *SequenceProfile) FragmentSize() int {
	return lib.FragSize
}

// Fragment returns the ith fragment in this library (starting from 0).
func (lib *SequenceProfile) Fragment(i int) SequenceFragment {
	return lib.Fragments[i]
}

// String returns a string with the name of the library, the number of
// fragments in the library and the size of each fragment.
func (lib *SequenceProfile) String() string {
	return fmt.Sprintf("%s (%d, %d)",
		lib.Ident, len(lib.Fragments), lib.FragSize)
}

func (lib *SequenceProfile) Name() string {
	return lib.Ident
}

// Best returns the number of the fragment that best corresponds
// to the string of amino acids provided.
// The length of `sequence` must be equivalent to the fragment size.
//
// If no "good" fragments can be found, then `-1` is returned. This
// behavior will almost certainly change in the future.
func (lib *SequenceProfile) Best(s seq.Sequence) int {
	// Since fragments are guaranteed not to have gaps by construction,
	// we can do a straight-forward summation of the negative log-odds
	// probabilities corresponding to the residues in `s`.
	var testAlign seq.Prob
	bestAlign, bestFragNum := seq.MinProb, -1
	for _, frag := range lib.Fragments {
		testAlign = frag.AlignmentProb(s)
		if bestAlign.Less(testAlign) {
			bestAlign, bestFragNum = testAlign, frag.FragNumber
		}
	}
	return bestFragNum
}

// Fragment corresponds to a single sequence fragment in a fragment library.
// It holds the fragment number identifier and embeds a sequence profile.
type SequenceProfileFrag struct {
	FragNumber int
	*seq.Profile
}

// AlignmentProb computes the probability of the sequence `s` aligning
// with the profile in `frag`. The sequence must have length equivalent
// to the fragment size.
func (frag SequenceProfileFrag) AlignmentProb(s seq.Sequence) seq.Prob {
	if s.Len() != frag.Len() {
		panic(fmt.Sprintf("Sequence length %d != fragment size %d",
			s.Len(), frag.Len()))
	}
	prob := seq.Prob(0.0)
	for c := 0; c < s.Len(); c++ {
		prob += frag.Emissions[c][s.Residues[c]]
	}
	return prob
}

func (frag SequenceProfileFrag) Number() int {
	return frag.FragNumber
}

func (frag SequenceProfileFrag) String() string {
	return fmt.Sprintf("> %d\n%s", frag.FragNumber, frag.Profile)
}
