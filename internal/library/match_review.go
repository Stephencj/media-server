package library

import (
	"fmt"
	"strings"

	"github.com/stephencjuliano/media-server/internal/db"
	"github.com/stephencjuliano/media-server/pkg/tmdb"
)

// MovieFileFacts is what a movie's own file says about it, the evidence a
// match is judged against.
type MovieFileFacts struct {
	Title   string
	Year    int // 0 when the name carries none
	Runtime int // minutes, 0 when unknown
	// FolderTitle is the folder's fuller name for the same film, "" when the
	// folder adds nothing. See MovieFolderTitle.
	FolderTitle string
}

// FactsForMovie reads the evidence from a movie's path and duration.
func FactsForMovie(media *db.Media) MovieFileFacts {
	title, year := MovieNameFromPath(media.FilePath)
	return MovieFileFacts{
		Title:       title,
		Year:        year,
		Runtime:     runtimeMinutes(media.Duration),
		FolderTitle: MovieFolderTitle(media.FilePath),
	}
}

// MatchMovieFile finds the TMDB film for a file: by the title in its name,
// and failing that by its folder's fuller title.
func MatchMovieFile(client *tmdb.Client, facts MovieFileFacts) (*tmdb.MovieDetails, error) {
	details, err := client.MatchMovie(facts.Title, facts.Year, facts.Runtime)
	if err != nil || details != nil || facts.FolderTitle == "" {
		return details, err
	}
	return client.MatchMovie(facts.FolderTitle, facts.Year, facts.Runtime)
}

// ReviewMovieMatch re-runs matching for a movie already in the library and
// says whether its current match deserves a second look.
//
// suggested is what the matcher would pick today, nil if it would leave the
// file unmatched. flagged is whether that is worth showing someone, and reason
// says why in a sentence or two.
func ReviewMovieMatch(client *tmdb.Client, media *db.Media) (suggested *tmdb.MovieDetails, reason string, flagged bool, err error) {
	facts := FactsForMovie(media)
	suggested, err = MatchMovieFile(client, facts)
	if err != nil {
		return nil, "", false, err
	}
	reason, flagged = judgeMatch(media, facts, suggested)
	return suggested, reason, flagged, nil
}

// judgeMatch decides whether the difference between a movie's current match
// and the matcher's pick is worth raising.
//
// The matcher disagreeing is not enough on its own: two films with the same
// title and no year in the file name are separated only by popularity, and
// that is no reason to second-guess a match someone may have set by hand. The
// file has to say something the current match contradicts.
func judgeMatch(media *db.Media, facts MovieFileFacts, suggested *tmdb.MovieDetails) (reason string, flagged bool) {
	if media.TMDbID == 0 {
		if suggested == nil {
			return "", false
		}
		return "Nothing was matched before.", true
	}
	if suggested != nil && suggested.ID == media.TMDbID {
		return "", false
	}

	var concerns []string
	titleFits := func(title string) bool {
		return title != "" && (tmdb.TitlesAgree(title, media.Title) || tmdb.TitlesAgree(title, media.OriginalTitle))
	}
	if !titleFits(facts.Title) && !titleFits(facts.FolderTitle) {
		concerns = append(concerns, fmt.Sprintf("The file is named %q.", facts.Title))
	}
	if facts.Year > 0 && media.Year > 0 && absInt(facts.Year-media.Year) > 1 {
		concerns = append(concerns, fmt.Sprintf("The file says %d, the current match is from %d.", facts.Year, media.Year))
	}
	if tmdb.RuntimesConflict(facts.Runtime, media.Runtime) {
		concerns = append(concerns, fmt.Sprintf("The file runs %s, the current match runs %s.",
			FormatMinutes(facts.Runtime), FormatMinutes(media.Runtime)))
	}

	if len(concerns) > 0 {
		return strings.Join(concerns, " "), true
	}

	// No contradiction, but the file's length fits the other film and not
	// this one.
	if suggested != nil && tmdb.RuntimesAgree(facts.Runtime, suggested.Runtime) && !tmdb.RuntimesAgree(facts.Runtime, media.Runtime) {
		return fmt.Sprintf("The file runs %s, which fits the suggested film (%s) better than the current match (%s).",
			FormatMinutes(facts.Runtime), FormatMinutes(suggested.Runtime), FormatMinutes(media.Runtime)), true
	}

	return "", false
}

// FormatMinutes renders a running time the way the apps show it: "1h 8m".
func FormatMinutes(minutes int) string {
	if minutes <= 0 {
		return "an unknown length"
	}
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	if minutes%60 == 0 {
		return fmt.Sprintf("%dh", minutes/60)
	}
	return fmt.Sprintf("%dh %dm", minutes/60, minutes%60)
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
