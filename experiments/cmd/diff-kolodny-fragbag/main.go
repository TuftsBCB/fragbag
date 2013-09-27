package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/TuftsBCB/fragbag"
	"github.com/TuftsBCB/fragbag/bow"
	"github.com/TuftsBCB/io/pdb"
	"github.com/TuftsBCB/structure"
	"github.com/TuftsBCB/tools/util"
)

var (
	flagFragbag  string
	flagOldStyle bool
)

type oldStyle struct {
	*pdb.Entry
}

func (e oldStyle) StructureBOW(lib fragbag.StructureLibrary) bow.BOW {
	smushed := make([]structure.Coords, 0)
	for _, chain := range e.Chains {
		for _, model := range chain.Models {
			smushed = append(smushed, model.CaAtoms()...)
		}
	}
	return bow.StructureBOW(lib, smushed)
}

type newStyle struct {
	*pdb.Entry
}

func (e newStyle) StructureBOW(lib fragbag.StructureLibrary) bow.BOW {
	bag := bow.NewBow(lib.Size())
	for _, chain := range e.Chains {
		for _, model := range chain.Models {
			bag = bag.Add(bow.StructureBOW(lib, model.CaAtoms()))
		}
	}
	return bag
}

func init() {
	flag.StringVar(&flagFragbag, "fragbag", "fragbag",
		"The old fragbag executable.")
	flag.BoolVar(&flagOldStyle, "oldstyle", false,
		"When true, PDB chains will be concatenated together as if they were "+
			"one chain to compute a BOW vector.")

	util.FlagParse(
		"library-file brk-file pdb-file [pdb-file ...]",
		"Note that if the old library and the new library don't have the\n"+
			"same number of fragments and the same fragment size, bad things\n"+
			"will happen.\n")
	util.AssertLeastNArg(3)
}

func stderrf(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, format, v...)
}

func main() {
	libFile := util.Arg(0)
	brkFile := util.Arg(1)
	lib := util.StructureLibrary(libFile)

	stderrf("Loading PDB files into memory...\n")
	entries := make([]*pdb.Entry, util.NArg()-2)
	for i, pdbfile := range flag.Args()[2:] {
		entries[i] = util.PDBRead(pdbfile)
	}

	stderrf("Comparing the results of old fragbag and new fragbag on " +
		"each PDB file...\n")
	for _, entry := range entries {
		stderrf("Testing %s...\n", entry.Path)
		fmt.Printf("Testing %s\n", entry.Path)

		// Try to run old fragbag first. The output is an old-style BOW.
		oldBowStr, err := runOldFragbag(brkFile, entry.Path, lib.Size(),
			lib.FragmentSize())
		if err != nil {
			fmt.Println(err)
			fmt.Printf("The output was:\n%s\n", oldBowStr)
			divider()
			continue
		}

		oldBow, err := bow.NewOldStyleBow(lib.Size(), oldBowStr)
		if err != nil {
			fmt.Printf("Could not parse the following as an old style "+
				"BOW:\n%s\n", oldBowStr)
			fmt.Printf("%s\n", err)
			divider()
			continue
		}

		// Now use package fragbag to compute a BOW.
		var newBow bow.BOW
		if flagOldStyle {
			newBow = oldStyle{entry}.StructureBOW(lib)
		} else {
			newBow = newStyle{entry}.StructureBOW(lib)
		}

		// Create a diff and check if they are the same. If so, we passed.
		// Otherwise, print an error report.
		diff := bow.NewBowDiff(oldBow, newBow)
		if diff.IsSame() {
			fmt.Println("PASSED.")
			divider()
			continue
		}

		// Ruh roh...
		fmt.Println("FAILED.")
		fmt.Printf("\nOld BOW:\n%s\n\nNew BOW:\n%s\n", oldBow, newBow)
		fmt.Printf("\nDiff:\n%s\n", diff)
		divider()
	}
	stderrf("Done!\n")
}

func runOldFragbag(libFile, pdbFile string, size, fraglen int) (string, error) {
	cmd := []string{
		flagFragbag,
		"-l", libFile,
		fmt.Sprintf("%d", size),
		"-z", fmt.Sprintf("%d", fraglen),
		"-p", pdbFile,
		"-c"}
	out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out)),
			fmt.Errorf("There was an error executing: %s\n%s",
				strings.Join(cmd, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func divider() {
	fmt.Println("----------------------------------------------------")
}
