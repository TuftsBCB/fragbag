/*
Package fragbag provides interfaces for using fragment libraries along with
several implementations of fragment libraries. This package makes it possible
for clients to define their own fragment libraries while reusing all of the
infrastructure which operates on fragment libraries.

The central type of this package is the Library interface, along with its
child interfaces: SequenceLibrary, StructureLibrary and WeightedLibrary. The
Library interface states that all libraries have names, some collection of
fragments of uniform size, possibly a sub library and a uniquely identifying
tag. The tag is used to recapitulate the type of the fragment library (from the
Openers map) when reading them from disk.

Libraries may also wrap other libraries to provide additional functionality.
For example, the WeightedLibrary interface describes any fragment library that
can weight the raw frequency of a fragment against a query. But this
functionality can be added to existing libraries by wrapping them with
additional information. (For example, see the implementation of the
WeightedTfIdf library.)

A central design decision of this package is that all fragment libraries are
immutable. Once they are created, they cannot be changed. Therefore, all
actions defined by the Library interfaces never mutate an existing library.
*/
package fragbag
