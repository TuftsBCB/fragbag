package main

import (
	"encoding/csv"
	"flag"
	"fmt"

	"github.com/TuftsBCB/io/pdb"
	"github.com/TuftsBCB/seq"
	"github.com/TuftsBCB/structure"
	"github.com/TuftsBCB/tools/util"
)

var flagAllFragments = false

func main() {
	flag.BoolVar(&flagAllFragments, "all-fragments", flagAllFragments,
		"When set, all fragments will be shown, even if the best fragment\n"+
			"of each ATOM set is the same.")
	util.FlagParse(
		"fraglib align.{fasta,ali,a2m,a3m} pdb-file out-csv",
		"Writes a CSV file to out-csv containing the best matching fragment\n"+
			"for each pairwise contiguous set of alpha-carbon atoms of the\n"+
			"first two proteins in the alignment and PDB file.")
	util.AssertNArg(4)
	flib := util.StructureLibrary(util.Arg(0))
	aligned := util.MSA(util.Arg(1))
	pentry := util.PDBRead(util.Arg(2))
	outcsv := util.CreateFile(util.Arg(3))

	csvWriter := csv.NewWriter(outcsv)
	csvWriter.Comma = '\t'
	defer csvWriter.Flush()

	pf := func(record ...string) {
		util.Assert(csvWriter.Write(record), "Problem writing to '%s'", outcsv)
	}
	pf("start1", "end1", "start2", "end2", "frag1", "frag2", "frag_rmsd")
	iter := newContiguous(
		flib.FragmentSize(),
		aligned.GetFasta(0), aligned.GetFasta(1),
		pentry.Chains[0], pentry.Chains[1])
	for iter.next() {
		best1, best2 := flib.Best(iter.atoms1), flib.Best(iter.atoms2)
		if !flagAllFragments && best1 == best2 {
			continue
		}
		bestRmsd := structure.RMSD(
			flib.Fragments[best1].Atoms,
			flib.Fragments[best2].Atoms,
		)
		pf(
			fmt.Sprintf("%d", iter.s1()),
			fmt.Sprintf("%d", iter.e1()),
			fmt.Sprintf("%d", iter.s2()),
			fmt.Sprintf("%d", iter.e2()),
			fmt.Sprintf("%d", best1),
			fmt.Sprintf("%d", best2),
			fmt.Sprintf("%f", bestRmsd),
		)
	}
}

type contiguous struct {
	// Initial (immutable) state.
	fragSize       int
	seq1, seq2     seq.Sequence
	chain1, chain2 *pdb.Chain

	// State that changes with each iteration.
	current        int // Index into alignment.
	seen1, seen2   int // Number of non-gapped residues seen.
	atoms1, atoms2 []structure.Coords
}

func newContiguous(
	fragSize int,
	seq1, seq2 seq.Sequence,
	chain1, chain2 *pdb.Chain,
) *contiguous {
	return &contiguous{
		fragSize: fragSize,
		seq1:     seq1, seq2: seq2,
		chain1: chain1, chain2: chain2,
		current: -1,
		seen1:   0, seen2: 0,
		atoms1: nil, atoms2: nil,
	}
}

func (cont *contiguous) next() bool {
	cont.current++
	for cont.current <= cont.seq1.Len()-cont.fragSize {
		s1, e1 := cont.s1(), cont.e1()
		s2, e2 := cont.s2(), cont.e2()
		if e1 > len(cont.chain1.Models[0].Residues) ||
			e2 > len(cont.chain2.Models[0].Residues) {
			return false
		}

		if cont.seq1.Residues[cont.current] != '-' {
			cont.seen1++
		}
		if cont.seq2.Residues[cont.current] != '-' {
			cont.seen2++
		}
		if cont.hasGap(cont.seq1) || cont.hasGap(cont.seq2) {
			cont.current++
			continue
		}

		cont.atoms1 = sliceNoGaps(cont.chain1, s1, e1)
		cont.atoms2 = sliceNoGaps(cont.chain2, s2, e2)
		if cont.atoms1 == nil || cont.atoms2 == nil {
			cont.atoms1, cont.atoms2 = nil, nil
			cont.current++
			continue
		} else {
			return true
		}
	}
	return false
}

func (cont *contiguous) s1() int { return cont.seen1 }
func (cont *contiguous) e1() int { return cont.s1() + cont.fragSize }
func (cont *contiguous) s2() int { return cont.seen2 }
func (cont *contiguous) e2() int { return cont.s2() + cont.fragSize }

func (cont *contiguous) hasGap(s seq.Sequence) bool {
	for _, r := range s.Residues[cont.current : cont.current+cont.fragSize] {
		if r == '-' {
			return true
		}
	}
	return false
}

func sliceNoGaps(chain *pdb.Chain, s, e int) []structure.Coords {
	m := chain.Models[0]
	if s < 0 || s >= e || e > len(m.Residues) {
		panic(fmt.Sprintf(
			"Invalid range [%d, %d). Must be in [%d, %d).",
			s, e, 0, len(m.Residues)))
	}
	result := make([]structure.Coords, e-s)
	last := m.Residues[s].SequenceNum
	for i := 0; i < len(result); i++ {
		r := m.Residues[s+i]
		if last+1 < r.SequenceNum {
			return nil
		}
		last = r.SequenceNum
		result[i] = caAtom(r)
	}
	return result
}

func caAtom(r *pdb.Residue) structure.Coords {
	for _, atom := range r.Atoms {
		if atom.Name == "CA" && !atom.Het {
			return atom.Coords
		}
	}
	panic(fmt.Sprintf("No CA atom for residue (%s, %d)", r.Name, r.SequenceNum))
}
