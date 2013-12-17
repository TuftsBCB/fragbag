/*
Package bow provides a representation of a bag-of-words (BOW) along with
definitions of common operations. These operations include computing the cosine
or euclidean distance between two BOWs, comparing BOWs and producing BOWs from
values of other types (like a PDB chain or a biological sequence).

This package also includes special interoperable functions with the original
FragBag implementation written by Rachel Kolodny. Namely, BOWs in the original
implementation are encoded as strings (Bow.StringOldStyle writes them and
NewOldStyleBow reads them).
*/
package bow
