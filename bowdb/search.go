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
	Entry
	Cosine, Euclid float64
}

func newSearchResult(query, entry Entry) SearchResult {
	return SearchResult{
		Entry:  entry,
		Cosine: query.BOW.Cosine(entry.BOW),
		Euclid: query.BOW.Euclid(entry.BOW),
	}
}

// Search runs SearchEntry on the return value of NewEntry(query).
func (db *DB) Search(opts SearchOptions, query bow.Bower) []SearchResult {
	return db.SearchEntry(opts, db.NewEntry(query))
}

// SearchEntry currently performs an exhaustive search against the query
// entry. The best N results are returned with respect to the options given.
//
// At this point in time, the ReadAll method must be called before invoking
// a search, otherwise this function will panic. This requirement may be lifted
// if indexed searching is added (to avoid exhaustive searching).
func (db *DB) SearchEntry(opts SearchOptions, query Entry) []SearchResult {
	tree := new(bst)

	if db.entries == nil {
		panic("The ReadAll method must be called before searching.")
	}
	for _, entry := range db.entries {
		// Compute the distance between the query and the target.
		var dist float64
		switch opts.SortBy {
		case Cosine:
			dist = query.BOW.Cosine(entry.BOW)
		case Euclid:
			dist = query.BOW.Euclid(entry.BOW)
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
			results[i] = newSearchResult(query, n.Entry)
			i += 1
		})
	} else {
		tree.root.inorderReverse(func(n *node) {
			results[i] = newSearchResult(query, n.Entry)
			i += 1
		})
	}
	return results
}
