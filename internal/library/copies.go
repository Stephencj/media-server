package library

import (
	"fmt"
	"sort"

	"github.com/stephencjuliano/media-server/internal/db"
)

// Copies of a film.
//
// A library often holds the same film more than once: an HD re-rip beside the
// original, a compressed copy, a file a ripper named "T00". They are one film
// to whoever is browsing, so listings show one entry and the rest are reached
// from it.

// GroupMovieCopies folds movies that are the same film into one entry each.
// It returns the entries in the order their first file appeared, each with
// CopyCount set, and for every entry's ID the other files of that film.
//
// Films are the same when they share a TMDB match, or when unmatched, a
// folder, title and year. The entry shown is the fullest file: the longest,
// then the largest, since the reason for a second copy is usually that the
// first came out short.
func GroupMovieCopies(movies []*db.Media) (entries []*db.Media, copies map[int64][]*db.Media) {
	groups := map[string][]*db.Media{}
	var order []string
	for _, movie := range movies {
		key := fmt.Sprintf("tmdb:%d", movie.TMDbID)
		if movie.TMDbID == 0 {
			key = "file:" + copyKey(movie.FilePath)
		}
		if _, seen := groups[key]; !seen {
			order = append(order, key)
		}
		groups[key] = append(groups[key], movie)
	}

	copies = make(map[int64][]*db.Media, len(order))
	for _, key := range order {
		group := groups[key]
		sort.SliceStable(group, func(a, b int) bool {
			if group[a].Duration != group[b].Duration {
				return group[a].Duration > group[b].Duration
			}
			return group[a].FileSize > group[b].FileSize
		})

		entry := *group[0]
		entry.CopyCount = len(group) - 1
		entries = append(entries, &entry)
		copies[entry.ID] = group[1:]
	}
	return entries, copies
}

// MovieCopies returns the other files of the film a movie is, whichever of
// its files is asked about.
func MovieCopies(movies []*db.Media, mediaID int64) []*db.Media {
	entries, copies := GroupMovieCopies(movies)
	for _, entry := range entries {
		group := append([]*db.Media{entry}, copies[entry.ID]...)
		for i, movie := range group {
			if movie.ID != mediaID {
				continue
			}
			others := make([]*db.Media, 0, len(group)-1)
			for j, other := range group {
				if j != i {
					// Compare by ID: the entry is a copy carrying CopyCount.
					plain := *other
					plain.CopyCount = 0
					others = append(others, &plain)
				}
			}
			return others
		}
	}
	return nil
}
