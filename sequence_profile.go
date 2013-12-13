package fragbag

import (
	"fmt"

	"github.com/TuftsBCB/seq"
)

var _ = SequenceLibrary(&sequenceProfile{})

// sequenceProfile represents a Fragbag sequence fragment library.
// Fragbag fragment libraries are fixed both in the number of fragments and in
// the size of each fragment.
type sequenceProfile struct {
	Ident     string
	Fragments []sequenceProfileFrag
	FragSize  int
}

// Fragment corresponds to a single sequence fragment in a fragment library.
// It holds the fragment number identifier and embeds a sequence profile.
type sequenceProfileFrag struct {
	FragNumber int
	*seq.Profile
}

// NewSequenceProfile initializes a new Fragbag sequence library with the
// given name and fragments. All sequence profiles given must have the same
// number of columns.
//
// Fragments for this library are represented as regular sequence profiles.
// Namely, each column plainly represents the composition of each amino acid.
func NewSequenceProfile(
	name string,
	fragments []*seq.Profile,
) (SequenceLibrary, error) {
	lib := new(sequenceProfile)
	lib.Ident = name
	for _, frag := range fragments {
		if err := lib.add(frag); err != nil {
			return nil, err
		}
	}
	return lib, nil
}

func (lib *sequenceProfile) SubLibrary() Library {
	return nil
}

// Add adds a sequence fragment to the library, where a sequence fragment
// corresponds to a profile of log-odds scores for each amino acid.
// The first call to Add may contain any number of columns in the profile.
// All subsequent adds must contain the same number of columns as the first.
func (lib *sequenceProfile) add(prof *seq.Profile) error {
	if lib.Fragments == nil || len(lib.Fragments) == 0 {
		frag := sequenceProfileFrag{0, prof}
		lib.Fragments = append(lib.Fragments, frag)
		lib.FragSize = prof.Len()
		return nil
	}

	frag := sequenceProfileFrag{len(lib.Fragments), prof}
	if lib.FragSize != prof.Len() {
		return fmt.Errorf("Fragment %d has length %d; expected length %d.",
			frag.FragNumber, prof.Len(), lib.FragSize)
	}
	lib.Fragments = append(lib.Fragments, frag)
	return nil
}

// Save saves the full fragment library to the writer provied.
func (lib *sequenceProfile) Tag() string {
	return libTagSequenceProfile
}

// Size returns the number of fragments in the library.
func (lib *sequenceProfile) Size() int {
	return len(lib.Fragments)
}

// FragmentSize returns the size of every fragment in the library.
func (lib *sequenceProfile) FragmentSize() int {
	return lib.FragSize
}

// String returns a string with the name of the library, the number of
// fragments in the library and the size of each fragment.
func (lib *sequenceProfile) String() string {
	return fmt.Sprintf("%s (%d, %d)",
		lib.Ident, len(lib.Fragments), lib.FragSize)
}

func (lib *sequenceProfile) Name() string {
	return lib.Ident
}

// Best returns the number of the fragment that best corresponds
// to the string of amino acids provided.
// The length of `sequence` must be equivalent to the fragment size.
//
// If no "good" fragments can be found, then `-1` is returned. This
// behavior will almost certainly change in the future.
func (lib *sequenceProfile) BestSequenceFragment(s seq.Sequence) int {
	// Since fragments are guaranteed not to have gaps by construction,
	// we can do a straight-forward summation of the negative log-odds
	// probabilities corresponding to the residues in `s`.
	var testAlign seq.Prob
	bestAlign, bestFragNum := seq.MinProb, -1
	for i := range lib.Fragments {
		testAlign = lib.AlignmentProb(i, s)
		if bestAlign.Less(testAlign) {
			bestAlign, bestFragNum = testAlign, i
		}
	}
	return bestFragNum
}

func (lib *sequenceProfile) FragmentString(fragNum int) string {
	return fmt.Sprintf("> %d\n%s", fragNum, lib.Fragments[fragNum].Profile)
}

// AlignmentProb computes the probability of the sequence `s` aligning
// with the profile in `frag`. The sequence must have length equivalent
// to the fragment size.
func (lib *sequenceProfile) AlignmentProb(fragi int, s seq.Sequence) seq.Prob {
	frag := lib.Fragments[fragi]
	if s.Len() != frag.Len() {
		panic(fmt.Sprintf("Sequence length %d != fragment size %d",
			s.Len(), frag.Len()))
	}
	prob := seq.Prob(0.0)
	for c := 0; c < s.Len(); c++ {
		prob += frag.Emissions[c].Lookup(s.Residues[c])
	}
	return prob
}
