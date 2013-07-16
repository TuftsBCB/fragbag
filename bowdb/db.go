package bowdb

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"runtime"
	"sync"

	"github.com/TuftsBCB/fragbag"
	"github.com/TuftsBCB/fragbag/bow"
)

// DB represents a BOW database. It is always connected to a particular
// fragment library. In particular, the disk representation of the database is
// a directory with a copy of the fragment library used to create the database
// and a binary formatted file of all the frequency vectors computed.
type DB struct {
	Lib  fragbag.Library
	Path string
	Name string
	file *os.File

	// Only set when opened in reading mode.
	Entries []Entry

	// for reading only
	entryBuf []byte

	// for writing only
	writeBuf    *bytes.Buffer
	writing     chan bow.Bower
	wg          *sync.WaitGroup
	writingDone chan struct{}
	entries     chan Entry
}

// IsStructure returns true if the underlying fragment library associated with
// this BOW database is based on structure fragments. This value is guaranteed
// to be mutually exclusive with the return value of IsSequence.
func (db *DB) IsStructure() bool {
	_, ok := db.Lib.(*fragbag.StructureLibrary)
	return ok
}

// IsSequence returns true if the underlying fragment library associated with
// this BOW database is based on sequence fragments. This value is guaranteed
// to be mutually exclusive with the return value of IsStructure.
func (db *DB) IsSequence() bool {
	_, ok := db.Lib.(*fragbag.SequenceLibrary)
	return ok
}

// OpenDB opens a new BOW database for reading. In particular, all entries
// in the database will be loaded into memory.
func OpenDB(dir string) (*DB, error) {
	var err error

	db := &DB{
		Path: dir,
		Name: path.Base(dir),
	}

	libf, err := os.Open(db.filePath("frag.lib"))
	if err != nil {
		return nil, err
	}

	db.Lib, err = fragbag.Open(libf)
	if err != nil {
		return nil, err
	}

	db.file, err = os.Open(db.filePath("bow.db"))
	if err != nil {
		return nil, err
	}

	db.Entries = make([]Entry, 0, 1000)
	for {
		entry, err := db.read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		db.Entries = append(db.Entries, entry)
	}

	return db, nil
}

// CreateDB creates a new BOW database on disk at 'dir'. If the directory
// already exists or cannot be created, an error is returned.
//
// CreateDB starts GOMAXPROCS workers, where each worker computes a single
// BOW at a time. You should call `Add` to add any value implementing the
// Bower interface, and `Close` when finished adding.
//
// One a BOW database is created, it cannot be modified.
func CreateDB(lib fragbag.Library, dir string) (*DB, error) {
	var err error

	_, err = os.Stat(dir)
	if err == nil || !os.IsNotExist(err) {
		return nil, fmt.Errorf("BOW database '%s' already exists.", dir)
	}
	if err = os.MkdirAll(dir, 0777); err != nil {
		return nil, fmt.Errorf("Could not create '%s': %s", dir, err)
	}

	db := &DB{
		Lib:  lib,
		Path: dir,
		Name: path.Base(dir),

		writeBuf:    new(bytes.Buffer),
		writing:     make(chan bow.Bower),
		entries:     make(chan Entry),
		writingDone: make(chan struct{}),
		wg:          new(sync.WaitGroup),
	}

	fp := db.filePath("bow.db")
	db.file, err = os.Create(fp)
	if err != nil {
		return nil, fmt.Errorf("Could not create '%s': %s", fp, err)
	}

	libfp := db.filePath("frag.lib")
	libf, err := os.Create(libfp)
	if err != nil {
		return nil, fmt.Errorf("Could not create '%s': %s", libfp, err)
	}
	if err := db.Lib.Save(libf); err != nil {
		return nil, fmt.Errorf("Could not copy fragment library: %s", err)
	}

	// Spin up goroutines to compute BOWs.
	for i := 0; i < max(1, runtime.GOMAXPROCS(0)); i++ {
		go func() {
			db.wg.Add(1)
			for bower := range db.writing {
				db.entries <- db.NewEntry(bower)
			}
			db.wg.Done()
		}()
	}

	// Now spin up a goroutine that is responsible for writing entries.
	go func() {
		for entry := range db.entries {
			if err = db.write(entry); err != nil {
				log.Printf("Could not write to bow.db: %s", err)
			}
		}
		db.writingDone <- struct{}{}
	}()

	return db, nil
}

// Add will add any value implementing the Bower interface to the BOW
// database. It is safe to call `Add` from multiple goroutines.
// If the fragment library in the database is structure based, then bower
// must also implement StructureBower. Conversely, if the fragment library is
// sequence based, then bower must also implement SequenceBower.
// A violation of the aforementioned invariant will result in a type assertion
// panic.
//
// Note that `CreateDB` will already compute BOWs concurrently, which will
// take advantage of parallelism when multiple CPUs are present.
//
// Add will panic if it is called on a BOW database that been opened for
// reading.
func (db *DB) Add(bower bow.Bower) {
	if db.writing == nil {
		panic("Cannot add to a BOW database opened in read mode.")
	}
	db.writing <- bower
}

// filePath concatenates the BOW database path with a file name.
func (db *DB) filePath(name string) string {
	return path.Join(db.Path, name)
}

// Close should be called when done reading/writing a BOW db.
func (db *DB) Close() error {
	if db.writing != nil {
		close(db.writing)
		db.wg.Wait()
		close(db.entries)
		<-db.writingDone
	}
	return db.file.Close()
}

func (db *DB) String() string {
	return db.Name
}

// Entry corresponds to a single row in the BOW database. It is uniquely
// identified by Id, which is typically constructed as the concatenation
// of the 4 letter PDB Id Code with the single letter chain identifier.
type Entry struct {
	Id  string
	BOW bow.BOW
}

func (db *DB) NewEntry(bower bow.Bower) Entry {
	switch lib := db.Lib.(type) {
	case *fragbag.StructureLibrary:
		b := bower.(bow.StructureBower)
		return Entry{
			b.Id(),
			b.StructureBOW(lib),
		}
	case *fragbag.SequenceLibrary:
		b := bower.(bow.SequenceBower)
		return Entry{
			b.Id(),
			b.SequenceBOW(lib),
		}
	}
	panic(fmt.Sprintf("Unsupported fragment library: %T", db.Lib))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// read will read a single entry from the BOW database.
//
// It would be much nicer to use the binary package here (like we do for
// reading), but we need to be as fast here as possible. (It looks like
// there is a fair bit of allocation going on in the binary package.)
// Benchmarks are gone in the wind...
func (db *DB) read() (Entry, error) {
	libs := db.Lib.Size()

	// Find the number of bytes used by the next entry.
	entryLenBs := make([]byte, 4)
	if n, err := db.file.Read(entryLenBs); err != nil {
		// Test the first read to see if we're at the end.
		// This is the only place where it's OK to see an EOF.
		if err == io.EOF {
			return Entry{}, err
		}
		return Entry{}, fmt.Errorf("Error reading entry length: %s", err)
	} else if n != len(entryLenBs) {
		return Entry{},
			fmt.Errorf("Expected entry length with length %d, but got %d",
				len(entryLenBs), n)
	}
	entryLen := readUint32(entryLenBs)

	// Read in the full entry.
	if db.entryBuf == nil || int(entryLen) > cap(db.entryBuf) {
		db.entryBuf = make([]byte, entryLen)
	}
	entry := db.entryBuf[0:entryLen]
	if n, err := db.file.Read(entry); err != nil {
		return Entry{},
			fmt.Errorf("Error reading entry: %s", err)
	} else if n != len(entry) {
		return Entry{},
			fmt.Errorf("Expected entry with length %d, but got %d",
				len(entry), n)
	}

	// Now gobble up a null terminated id string and the BOW vector.
	id := string(entry[0 : len(entry)-(1+libs*2)])
	vector := entry[len(id)+1:]
	freqs := make([]uint32, libs)
	for i := 0; i < libs; i++ {
		freqs[i] = readUint16As32(vector[i*2:])
	}

	return Entry{
		Id:  id,
		BOW: bow.BOW{freqs},
	}, nil
}

// big endian
func readUint32(b []byte) uint32 {
	return uint32(b[0])<<24 |
		uint32(b[1])<<16 |
		uint32(b[2])<<8 |
		uint32(b[3])
}

// big endian
func readUint16As32(b []byte) uint32 {
	return uint32(b[0])<<8 | uint32(b[1])
}

func (db *DB) write(entry Entry) error {
	endian := binary.BigEndian
	idCode := fmt.Sprintf("%s%c", entry.Id, 0)
	libSize := db.Lib.Size()
	buf := db.writeBuf

	// Write the id code and BOW vector to a buffer.
	buf.Reset()
	if _, err := buf.WriteString(idCode); err != nil {
		return fmt.Errorf("Something bad has happened when trying to write "+
			"id: %s.", err)
	}
	for i := 0; i < libSize; i++ {
		f := int16(entry.BOW.Freqs[i])
		if err := binary.Write(buf, endian, f); err != nil {
			return fmt.Errorf("Something bad has happened when trying to "+
				"write BOW: %s.", err)
		}
	}

	// Write the number of bytes in this entry.
	// (To make reading easier.)
	entryLen := uint32(buf.Len())
	if err := binary.Write(db.file, endian, entryLen); err != nil {
		return fmt.Errorf("Something bad has happened when trying to write "+
			"entry size to the bow.db: %s.", err)
	}

	// Write the buffer to disk.
	if _, err := db.file.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("Something bad has happened when trying to write "+
			"to the bow.db: %s.", err)
	}

	return nil
}
