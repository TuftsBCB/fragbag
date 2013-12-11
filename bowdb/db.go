package bowdb

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path"
	"runtime"
	"sync"

	"github.com/TuftsBCB/fragbag"
	"github.com/TuftsBCB/fragbag/bow"
)

const (
	fileBowDB   = "bow.db"
	fileFragLib = "frag-lib.json"
)

// DB represents a BOW database. It is always connected to a particular
// fragment library. In particular, the disk representation of the database is
// a directory with a copy of the fragment library used to create the database
// and a binary formatted file of all the frequency vectors computed.
type DB struct {
	Lib  fragbag.Library
	Path string
	Name string

	// The file containing the bow library and its buffer.
	file    *os.File
	fileBuf *bufio.Reader

	// Only set when opened in reading mode.
	entries []Entry

	// for reading only
	entryBuf []byte
	bowPool  []float32
	bowLast  int
	dataPool []byte
	dataLast int

	// for writing only
	writeBuf    *bytes.Buffer
	writing     chan bow.Bower
	wg          *sync.WaitGroup
	writingDone chan struct{}
	entryChan   chan Entry
}

// IsStructure returns true if the underlying fragment library associated with
// this BOW database is based on structure fragments. This value is guaranteed
// to be mutually exclusive with the return value of IsSequence.
func (db *DB) IsStructure() bool {
	_, ok := db.Lib.(fragbag.StructureLibrary)
	return ok
}

// IsSequence returns true if the underlying fragment library associated with
// this BOW database is based on sequence fragments. This value is guaranteed
// to be mutually exclusive with the return value of IsStructure.
func (db *DB) IsSequence() bool {
	_, ok := db.Lib.(fragbag.SequenceLibrary)
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

	libf, err := os.Open(db.filePath(fileFragLib))
	if err != nil {
		return nil, err
	}

	db.Lib, err = fragbag.Open(libf)
	if err != nil {
		return nil, err
	}

	db.file, err = os.Open(db.filePath(fileBowDB))
	if err != nil {
		return nil, err
	}
	db.fileBuf = bufio.NewReaderSize(db.file, 1<<20)
	return db, nil
}

// ReadAll reads all entries from disk and returns them in a slice.
// Subsequent calls do not read from disk; the already read entries are
// returned.
func (db *DB) ReadAll() ([]Entry, error) {
	if db.entries != nil {
		return db.entries, nil
	}
	db.entries = make([]Entry, 0, 10000)
	for {
		entry, err := db.read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		db.entries = append(db.entries, *entry)
	}
	return db.entries, nil
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
		entryChan:   make(chan Entry),
		writingDone: make(chan struct{}),
		wg:          new(sync.WaitGroup),
	}

	fp := db.filePath(fileBowDB)
	db.file, err = os.Create(fp)
	if err != nil {
		return nil, fmt.Errorf("Could not create '%s': %s", fp, err)
	}

	libfp := db.filePath(fileFragLib)
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
				db.entryChan <- db.NewEntry(bower)
			}
			db.wg.Done()
		}()
	}

	// Now spin up a goroutine that is responsible for writing entries.
	go func() {
		for entry := range db.entryChan {
			if err = db.write(entry); err != nil {
				log.Printf("Could not write to %s: %s", fileBowDB, err)
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
		close(db.entryChan)
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
	// Id is a uniquely identifying string for this row in the database.
	// In the case of a PDB chain structure, it is the 4 letter PDB id
	// code concatenated with the single letter chain identifier.
	Id string

	// Arbitrary data associated with this entry.
	// For example, the sequence header in a FASTA file.
	Data []byte

	// The fragment frequency vector.
	BOW bow.BOW
}

func (db *DB) NewEntry(bower bow.Bower) Entry {
	switch lib := db.Lib.(type) {
	case fragbag.StructureLibrary:
		b := bower.(bow.StructureBower)
		return Entry{
			b.Id(),
			bower.Data(),
			b.StructureBOW(lib),
		}
	case fragbag.SequenceLibrary:
		b := bower.(bow.SequenceBower)
		return Entry{
			b.Id(),
			bower.Data(),
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
func (db *DB) read() (*Entry, error) {
	libSize := db.Lib.Size()

	// Read in the id string.
	if err := db.readItem(); err != nil {
		return nil, err
	}
	id := string(db.entryBuf)

	// Read in the arbitrary data.
	if err := db.readItem(); err != nil {
		return nil, err
	}
	data := db.newData(len(db.entryBuf))
	copy(data, db.entryBuf)

	// Now read in the BOW.
	if err := db.readItem(); err != nil {
		return nil, err
	}
	if len(db.entryBuf) != libSize*4 {
		return nil, fmt.Errorf("Expected %d bytes for BOW vector but got %d",
			libSize*4, len(db.entryBuf))
	}

	freqs := db.newBow()
	for i := 0; i < libSize; i++ {
		freqs[i] = math.Float32frombits(
			binary.BigEndian.Uint32(db.entryBuf[i*4:]))
	}
	return &Entry{id, data, bow.BOW{freqs}}, nil
}

func (db *DB) newBow() []float32 {
	libSize := db.Lib.Size()
	if db.bowLast+libSize >= cap(db.bowPool) {
		db.bowPool = make([]float32, libSize*10000)
		db.bowLast = 0
	}
	b := db.bowPool[db.bowLast : db.bowLast+libSize]
	db.bowLast += libSize
	return b
}

func (db *DB) newData(size int) []byte {
	if size == 0 {
		return nil
	}
	if db.dataLast+size >= cap(db.dataPool) {
		db.dataPool = make([]byte, 5*(1<<20))
		db.dataLast = 0
	}
	d := db.dataPool[db.dataLast : db.dataLast+size]
	db.dataLast += size
	return d
}

func (db *DB) readItem() error {
	if err := db.readNBytes(4); err != nil {
		return err
	}
	itemLen := binary.BigEndian.Uint32(db.entryBuf)

	if err := db.readNBytes(int(itemLen)); err != nil {
		return err
	}
	return nil
}

func (db *DB) readNBytes(n int) error {
	if db.entryBuf == nil || n > cap(db.entryBuf) {
		db.entryBuf = make([]byte, n)
	}
	db.entryBuf = db.entryBuf[0:n]

	nread := 0
	for nread < n {
		if thisn, err := db.fileBuf.Read(db.entryBuf[nread:]); err != nil {
			if err == io.EOF {
				return io.EOF
			}
			return fmt.Errorf("Error reading item: %s", err)
		} else if thisn == 0 {
			return fmt.Errorf("Expected item with length %d, but got %d",
				n, nread)
		} else {
			nread += thisn
		}
	}
	return nil
}

func (db *DB) write(entry Entry) error {
	libSize := db.Lib.Size()

	db.writeBuf.WriteString(entry.Id)
	if err := db.writeItem(); err != nil {
		return err
	}

	db.writeBuf.Write(entry.Data)
	if err := db.writeItem(); err != nil {
		return err
	}

	for i := 0; i < libSize; i++ {
		f := entry.BOW.Freqs[i]
		if err := binary.Write(db.writeBuf, binary.BigEndian, f); err != nil {
			return fmt.Errorf("Could not write BOW for '%s': %s", entry.Id, err)
		}
	}
	if err := db.writeItem(); err != nil {
		return err
	}
	return nil
}

func (db *DB) writeItem() error {
	itemLen := uint32(db.writeBuf.Len())
	if err := binary.Write(db.file, binary.BigEndian, itemLen); err != nil {
		return fmt.Errorf("Could not write item size: %s", err)
	}
	if _, err := db.file.Write(db.writeBuf.Bytes()); err != nil {
		return fmt.Errorf("Could not write item: %s", err)
	}
	db.writeBuf.Reset()
	return nil
}
