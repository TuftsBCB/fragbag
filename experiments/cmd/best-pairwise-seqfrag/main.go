package main

import (
	"encoding/csv"
	"flag"
	"fmt"

	"github.com/TuftsBCB/seq"
	"github.com/TuftsBCB/tools/util"
)

var flagAllFragments = false

func main() {
	flag.BoolVar(&flagAllFragments, "all-fragments", flagAllFragments,
		"When set, all fragments will be shown, even if the best fragment\n"+
			"of each residue set is the same.")
	util.FlagParse(
		"fraglib align.{fasta,ali,a2m,a3m} out-csv",
		"Writes a CSV file to out-csv containing the best matching fragment\n"+
			"for each pairwise contiguous set of residues between the\n"+
			"first two proteins in the alignment.")
	util.AssertNArg(3)
	flib := util.SequenceLibrary(util.Arg(0))
	aligned := util.MSA(util.Arg(1))
	outcsv := util.CreateFile(util.Arg(2))

	csvWriter := csv.NewWriter(outcsv)
	csvWriter.Comma = '\t'
	defer csvWriter.Flush()

	pf := func(record ...string) {
		util.Assert(csvWriter.Write(record), "Problem writing to '%s'", outcsv)
	}
	pf("start1", "end1", "start2", "end2", "frag1", "frag2", "rat1", "rat2")
	iter := newContiguous(
		flib.FragmentSize(), aligned.GetFasta(0), aligned.GetFasta(1))
	for iter.next() {
		best1, best2 := flib.Best(iter.res1), flib.Best(iter.res2)
		if !flagAllFragments && best1 == best2 {
			continue
		}
		if best1 == -1 || best2 == -1 {
			continue
		}
		p1 := flib.Fragments[best1].AlignmentProb(iter.res1)
		p2 := flib.Fragments[best2].AlignmentProb(iter.res2)
		if p1.Distance(p2) > 0.14 {
			continue
		}
		pf(
			fmt.Sprintf("%d", iter.s1()),
			fmt.Sprintf("%d", iter.e1()),
			fmt.Sprintf("%d", iter.s2()),
			fmt.Sprintf("%d", iter.e2()),
			fmt.Sprintf("%d", best1),
			fmt.Sprintf("%d", best2),
			fmt.Sprintf("%f", p1),
			fmt.Sprintf("%f", p2),
		)
	}
}

type contiguous struct {
	// Initial (immutable) state.
	fragSize   int
	seq1, seq2 seq.Sequence

	// State that changes with each iteration.
	current      int // Index into alignment.
	seen1, seen2 int // Number of non-gapped residues seen.
	res1, res2   seq.Sequence
}

func newContiguous(fragSize int, seq1, seq2 seq.Sequence) *contiguous {
	return &contiguous{
		fragSize: fragSize,
		seq1:     seq1, seq2: seq2,
		current: -1,
		seen1:   0, seen2: 0,
		res1: seq.Sequence{}, res2: seq.Sequence{},
	}
}

func (cont *contiguous) next() bool {
	cont.current++
	for cont.current <= cont.seq1.Len()-cont.fragSize {
		s1, e1 := cont.s1(), cont.e1()
		s2, e2 := cont.s2(), cont.e2()
		if e1 > cont.seq1.Len() || e2 > cont.seq2.Len() {
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
		cont.res1 = cont.seq1.Slice(s1, e1)
		cont.res2 = cont.seq2.Slice(s2, e2)
		return true
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
