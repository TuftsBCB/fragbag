/*
Package bowdb provides functions for reading, writing and searching databases
of Bowed values. Bowed values correspond to a bag-of-words (BOW) along with
meta data about the value on which the BOW was computed (like a PDB chain
identifier or SCOP domain).

While reading a database and searching it has been heavily optimized, the
search itself is still exhaustive. No attempt has been made yet at constructing
a reverse index.

Every BOW database is associated with one and only one fragment library. When a
BOW database is saved, a copy of the fragment library is embedded into the
database. This library---and only this library---should be used to compute
Bowed values for use with the Search function.
*/
package bowdb
