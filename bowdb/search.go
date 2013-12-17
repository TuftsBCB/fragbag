package bowdb

import (
	"fmt"
	"math"

	"github.com/TuftsBCB/fragbag/bow"
)

const (
	Euclid = iota
	Cosine
)

const (
	OrderAsc = iota
	OrderDesc
)

type SearchOptions struct {
	Limit  int
	Min    float64
	Max    float64
	SortBy int
	Order  int
}

var SearchDefault = SearchOptions{
	Limit:  25,
	Min:    0.0,
	Max:    math.MaxFloat64,
	SortBy: Cosine,
	Order:  OrderAsc,
}

var SearchClose = SearchOptions{
	Limit:  -1,
	Min:    0.0,
	Max:    0.35,
	SortBy: Cosine,
	Order:  OrderAsc,
}

type SearchResult struct {
	bow.Bowed
	Cosine, Euclid float64
}

func newSearchResult(query, entry bow.Bowed) SearchResult {
	return SearchResult{
		Bowed:  entry,
		Cosine: query.Bow.Cosine(entry.Bow),
		Euclid: query.Bow.Euclid(entry.Bow),
	}
}

// Search performs an exhaustive search against the query entry. The best N
// results are returned with respect to the options given. The query given
// must have been computed with this database's fragment library.
//
// Note that if the ReadAll method hasn't been called before, Search will
// call it for you. (This means that the first search could take longer than
// one would otherwise expect.)
func (db *DB) Search(opts SearchOptions, query bow.Bowed) []SearchResult {
	tree := new(bst)

	if db.entries == nil {
		db.ReadAll()
	}
	for _, entry := range db.entries {
		// Compute the distance between the query and the target.
		var dist float64
		switch opts.SortBy {
		case Cosine:
			dist = query.Bow.Cosine(entry.Bow)
		case Euclid:
			dist = query.Bow.Euclid(entry.Bow)
		default:
			panic(fmt.Sprintf("Unrecognized SortBy value: %d", opts.SortBy))
		}

		// If the distance isn't in the min/max thresholds specified, skip it.
		if dist > opts.Max || dist < opts.Min {
			continue
		}

		// If there is a limit and we're already at that limit, then
		// we'll skip inserting this element if it's not better than the
		// worst hit.
		if tree.size == opts.Limit {
			if opts.Order == OrderAsc && dist >= tree.max.distance {
				continue
			} else if opts.Order == OrderDesc && dist <= tree.min.distance {
				continue
			}
		}

		// This target is good enough, add it to our results.
		tree.insert(entry, dist)

		// This element is good enough, so lets throw away the worst
		// result we have.
		if opts.Limit >= 0 && tree.size == opts.Limit+1 {
			if opts.Order == OrderAsc {
				tree.deleteMax()
			} else {
				tree.deleteMin()
			}
		}

		// Sanity check.
		if opts.Limit >= 0 && tree.size > opts.Limit {
			panic(fmt.Sprintf("Tree size (%d) is bigger than limit (%d).",
				tree.size, opts.Limit))
		}
	}

	results := make([]SearchResult, tree.size)
	i := 0
	if opts.Order == OrderAsc {
		tree.root.inorder(func(n *node) {
			results[i] = newSearchResult(query, n.Bowed)
			i += 1
		})
	} else {
		tree.root.inorderReverse(func(n *node) {
			results[i] = newSearchResult(query, n.Bowed)
			i += 1
		})
	}
	return results
}
