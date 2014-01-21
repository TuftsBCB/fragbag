package bow

import (
	"fmt"
	"strings"

	"github.com/TuftsBCB/fragbag"
	"github.com/TuftsBCB/io/pdb"
	"github.com/TuftsBCB/io/pdbx"
	"github.com/TuftsBCB/seq"
	"github.com/TuftsBCB/structure"
)

// Bowed corresponds to a bag-of-words with meta data about its source.
// For example, a PDB chain can have a BOW computed for it. Meta data might
// include that chain's identifier (e.g., 1ctfA) and perhaps that chain's
// sequence.
//
// Values of this type correspond to records in a BOW database.
type Bowed struct {
	// A globally unique identifier corresponding to the source of the bow.
	// e.g., a PDB identifier "1ctf" or a PDB identifier with a chain
	// identifier "1ctfA" or a sequence accession number.
	Id string

	// Arbitrary data associated with the source. May be empty.
	Data []byte

	// The bag-of-words.
	Bow Bow
}

// StructureBower corresponds to Bower values that can provide BOWs given
// a structure fragment library.
type StructureBower interface {
	// Computes a bag-of-words given a structure fragment library.
	// For example, to compute the bag-of-words of a chain in a PDB entry:
	//
	//     lib := someStructureFragmentLibrary()
	//     chain := somePdbChain()
	//     fmt.Println(BowerFromChain(chain).StructureBow(lib))
	//
	// This is made easier by using pre-defined types in this package that
	// implement this interface.
	StructureBow(lib fragbag.StructureLibrary) Bowed
}

type pdbChainStructure struct {
	*pdb.Chain
}

// BowerFromChain provides a reference implementation of the StructureBower
// interface for PDB chains.
func BowerFromChain(c *pdb.Chain) StructureBower {
	return pdbChainStructure{c}
}

func (c pdbChainStructure) id() string {
	switch {
	case len(c.Entry.Cath) > 0:
		return c.Entry.Cath
	case len(c.Entry.Scop) > 0:
		return c.Entry.Scop
	}
	return fmt.Sprintf("%s%c", strings.ToLower(c.Entry.IdCode), c.Ident)
}

func (c pdbChainStructure) StructureBow(lib fragbag.StructureLibrary) Bowed {
	return Bowed{
		Id:  c.id(),
		Bow: StructureBow(lib, c.CaAtoms()),
	}
}

type pdbModelStructure struct {
	*pdb.Model
}

// BowerFromModel provides a reference implementation of the StructureBower
// interface for PDB models.
func BowerFromModel(c *pdb.Model) StructureBower {
	return pdbModelStructure{c}
}

func (m pdbModelStructure) id() string {
	switch {
	case len(m.Entry.Scop) > 0:
		return m.Entry.Scop
	case len(m.Entry.Cath) > 0:
		return m.Entry.Cath
	}
	return fmt.Sprintf("%s%c%d",
		strings.ToLower(m.Entry.IdCode), m.Chain.Ident, m.Num)
}

func (m pdbModelStructure) StructureBow(lib fragbag.StructureLibrary) Bowed {
	return Bowed{
		Id:  m.id(),
		Bow: StructureBow(lib, m.CaAtoms()),
	}
}

type cifChainStructure struct {
	*pdbx.Chain
}

// BowerFromCifChain provides a reference implementation of the StructureBower
// interface for chains in PDBx/mmCIF formatted files.
func BowerFromCifChain(c *pdbx.Chain) StructureBower {
	return cifChainStructure{c}
}

func (c cifChainStructure) id() string {
	return fmt.Sprintf("%s%c", strings.ToLower(c.Entity.Entry.Id), c.Id)
}

func (c cifChainStructure) StructureBow(lib fragbag.StructureLibrary) Bowed {
	return Bowed{
		Id:  c.id(),
		Bow: StructureBow(lib, c.Models[0].AlphaCarbons),
	}
}

// StructureBow is a helper function to compute a bag-of-words given a
// structure fragment library and a list of alpha-carbon atoms.
//
// If the lib given is a weighted library, then the Bow returned will also
// be weighted.
//
// Note that this function should only be used when providing your own
// implementation of the StructureBower interface. Otherwise, BOWs should
// be computed using the StructureBow method of the interface.
func StructureBow(lib fragbag.StructureLibrary, atoms []structure.Coords) Bow {
	var best, uplimit int

	b := NewBow(lib.Size())
	libSize := lib.FragmentSize()
	uplimit = len(atoms) - libSize
	for i := 0; i <= uplimit; i++ {
		best = lib.BestStructureFragment(atoms[i : i+libSize])
		b.Freqs[best] += 1
	}
	if wlib, ok := lib.(fragbag.WeightedLibrary); ok {
		b = b.Weighted(wlib)
	}
	return b
}

// SequenceBower corresponds to Bower values that can provide BOWs given
// a sequence fragment library.
type SequenceBower interface {
	// Computes a bag-of-words given a sequence fragment library.
	SequenceBow(lib fragbag.SequenceLibrary) Bowed
}

type sequence struct {
	seq.Sequence
}

// BowerFromSequence provides a reference implementation of the SequenceBower
// interface for biological sequences.
func BowerFromSequence(s seq.Sequence) SequenceBower {
	return sequence{s}
}

func (s sequence) SequenceBow(lib fragbag.SequenceLibrary) Bowed {
	return Bowed{
		Id:   strings.Fields(s.Name)[0],
		Data: s.Bytes(),
		Bow:  SequenceBow(lib, s.Sequence),
	}
}

// SequenceBow is a helper function to compute a bag-of-words given a
// sequence fragment library and a query sequence.
//
// If the lib given is a weighted library, then the BOW returned will also
// be weighted.
//
// Note that this function should only be used when providing your own
// implementation of the SequenceBower interface. Otherwise, BOWs should
// be computed using the SequenceBow method of the interface.
func SequenceBow(lib fragbag.SequenceLibrary, s seq.Sequence) Bow {
	var best, uplimit int

	b := NewBow(lib.Size())
	libSize := lib.FragmentSize()
	uplimit = s.Len() - libSize
	for i := 0; i <= uplimit; i++ {
		best = lib.BestSequenceFragment(s.Slice(i, i+libSize))
		if best < 0 {
			continue
		}
		b.Freqs[best] += 1
	}
	if wlib, ok := lib.(fragbag.WeightedLibrary); ok {
		b = b.Weighted(wlib)
	}
	return b
}
