package tmdb

import (
	"sort"
	"strings"
)

// Movie matching.
//
// A search result is only a candidate if its title actually resembles the
// file's title; year, running time and popularity then choose between
// candidates. Nothing but the title can make an unrelated film a candidate,
// which is the property the previous scorer lacked: it gave every result half
// marks for its title, so a popular film from the right year could be accepted
// with a name that shared nothing with the file.

const (
	// minTitleSimilarity is the gate a result's title has to clear before
	// anything else about it is considered.
	minTitleSimilarity = 0.6

	// movieMatchThreshold is the score below which the file is left
	// unmatched. Showing the file's own name is better than showing the
	// wrong film's poster.
	movieMatchThreshold = 48.0

	// maxRuntimeLookups bounds the detail requests spent on telling
	// same-named candidates apart by running time.
	maxRuntimeLookups = 3
)

type movieCandidate struct {
	result *MovieSearchResult
	score  float64
}

// MatchMovie finds the TMDB movie a file most likely is, given the title and
// year parsed from its name and its running time in minutes. year and
// runtimeMinutes may be 0 when unknown. It returns nil with no error when
// nothing is a confident match.
func (c *Client) MatchMovie(title string, year, runtimeMinutes int) (*MovieDetails, error) {
	results, err := c.SearchMovieWithResults(title, year)
	if err != nil {
		return nil, err
	}

	// TMDB's year filter is exact, so a file whose year is off (a festival
	// date, a typo) comes back empty. Look again without it and let the
	// scoring decide whether title and running time are enough.
	if year > 0 && len(rankMovieCandidates(results, title, year)) == 0 {
		results, err = c.SearchMovieWithResults(title, 0)
		if err != nil {
			return nil, err
		}
	}

	return pickMovie(results, title, year, runtimeMinutes, c.GetMovieDetails)
}

// pickMovie chooses among search results. fetch is c.GetMovieDetails outside
// of tests.
func pickMovie(results []MovieSearchResult, title string, year, runtimeMinutes int, fetch func(int) (*MovieDetails, error)) (*MovieDetails, error) {
	candidates := rankMovieCandidates(results, title, year)
	if len(candidates) == 0 {
		return nil, nil
	}

	// Running time is only on the detail record. Without a file duration
	// there is nothing to compare it to, so only the leader is fetched.
	limit := maxRuntimeLookups
	if runtimeMinutes <= 0 {
		limit = 1
	}
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	var best *MovieDetails
	var bestScore float64
	var firstErr error
	for _, candidate := range candidates {
		details, err := fetch(candidate.result.ID)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		score := candidate.score + runtimeScore(runtimeMinutes, details.Runtime)
		if best == nil || score > bestScore {
			best, bestScore = details, score
		}
	}

	if best == nil {
		return nil, firstErr
	}
	if bestScore < movieMatchThreshold {
		return nil, nil
	}
	return best, nil
}

// rankMovieCandidates drops results whose title doesn't resemble the search
// title and orders the rest best first.
func rankMovieCandidates(results []MovieSearchResult, title string, year int) []movieCandidate {
	var candidates []movieCandidate
	for i := range results {
		if score, ok := movieSearchScore(&results[i], title, year); ok {
			candidates = append(candidates, movieCandidate{result: &results[i], score: score})
		}
	}
	sort.SliceStable(candidates, func(a, b int) bool {
		return candidates[a].score > candidates[b].score
	})
	return candidates
}

// movieSearchScore scores a search result from what the search itself returns:
//   - Title: 0-60, and the result is rejected outright below the gate
//   - Year: +25 exact, +18 one year out (release dates differ by country),
//     -25 further out. Unknown on either side is neutral.
//   - Popularity: 0-5, enough to break a tie and no more
func movieSearchScore(result *MovieSearchResult, title string, year int) (float64, bool) {
	similarity := movieTitleSimilarity(title, result.Title)
	if original := movieTitleSimilarity(title, result.OriginalTitle); original > similarity {
		similarity = original
	}
	if similarity < minTitleSimilarity {
		return 0, false
	}

	score := similarity * 60.0

	if resultYear := extractYearFromDate(result.ReleaseDate); year > 0 && resultYear > 0 {
		switch abs(resultYear - year) {
		case 0:
			score += 25.0
		case 1:
			score += 18.0
		default:
			score -= 25.0
		}
	}

	score += min(result.Popularity/20.0, 1.0) * 5.0

	return score, true
}

// runtimeScore compares the file's running time with TMDB's, in minutes.
// A near-identical length is strong evidence for a candidate and a very
// different one is strong evidence against. The band between is neutral:
// extended and theatrical cuts of the same film live there.
func runtimeScore(fileMinutes, tmdbMinutes int) float64 {
	if fileMinutes <= 0 || tmdbMinutes <= 0 {
		return 0
	}
	diff := abs(fileMinutes - tmdbMinutes)
	if diff <= 3 || diff*100 <= tmdbMinutes*3 {
		return 15.0
	}
	if diff > 10 && diff*100 > tmdbMinutes*20 {
		return -25.0
	}
	return 0
}

// movieTitleSimilarity compares a title parsed from a file name with a TMDB
// title:
//
//	1.0   the same once punctuation, case, accents and articles are ignored
//	0.75  the TMDB title is the file's title plus a subtitle
//	else  the share of words the two have in common, 0 when there are none
func movieTitleSimilarity(search, candidate string) float64 {
	a := titleTokens(search)
	b := titleTokens(candidate)
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	if strings.Join(a, " ") == strings.Join(b, " ") {
		return 1.0
	}
	if isSubtitled(a, b) {
		return 0.75
	}

	inA := make(map[string]bool, len(a))
	for _, token := range a {
		inA[token] = true
	}
	inB := make(map[string]bool, len(b))
	shared := 0
	for _, token := range b {
		if inA[token] && !inB[token] {
			shared++
		}
		inB[token] = true
	}

	longer := len(inA)
	if len(inB) > longer {
		longer = len(inB)
	}
	return float64(shared) / float64(longer)
}

// isSubtitled reports whether full is short followed by a subtitle, as in
// "Birdman" and "Birdman or (The Unexpected Virtue of Ignorance)". A trailing
// number is a sequel, not a subtitle: "Alien" is not "Alien 3".
func isSubtitled(short, full []string) bool {
	if len(full) <= len(short) {
		return false
	}
	for i, token := range short {
		if full[i] != token {
			return false
		}
	}
	for _, token := range full[len(short):] {
		if !isNumber(token) {
			return true
		}
	}
	return false
}

func isNumber(token string) bool {
	for _, r := range token {
		if r < '0' || r > '9' {
			return false
		}
	}
	return token != ""
}

var romanNumerals = map[string]string{
	"ii": "2", "iii": "3", "iv": "4", "v": "5",
	"vi": "6", "vii": "7", "viii": "8", "ix": "9",
}

// accentFolds covers the accented Latin letters and superscripts that turn up
// in film titles. File names usually drop the accent ("Amelie"), TMDB keeps it
// ("Amélie").
var accentFolds = map[rune]string{
	'à': "a", 'á': "a", 'â': "a", 'ã': "a", 'ä': "a", 'å': "a", 'ā': "a",
	'ç': "c", 'č': "c", 'ć': "c",
	'è': "e", 'é': "e", 'ê': "e", 'ë': "e", 'ē': "e", 'ę': "e",
	'ì': "i", 'í': "i", 'î': "i", 'ï': "i", 'ī': "i",
	'ñ': "n", 'ń': "n",
	'ò': "o", 'ó': "o", 'ô': "o", 'õ': "o", 'ö': "o", 'ø': "o", 'ō': "o",
	'ù': "u", 'ú': "u", 'û': "u", 'ü': "u", 'ū': "u",
	'ý': "y", 'ÿ': "y",
	'š': "s", 'ś': "s", 'ž': "z", 'ł': "l",
	'ß': "ss", 'æ': "ae", 'œ': "oe",
	// "Alien³" is "Alien 3", not "Alien".
	'²': " 2", '³': " 3",
}

// titleTokens reduces a title to the words that identify it: lowercased,
// accents folded, punctuation dropped, articles removed, and roman numerals
// written as digits so "Rocky II" and "Rocky 2" agree.
func titleTokens(title string) []string {
	var b strings.Builder
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '&':
			b.WriteString(" and ")
		case r == '\'' || r == '’':
			// "Don't" and "Dont" are the same word.
		default:
			if folded, ok := accentFolds[r]; ok {
				b.WriteString(folded)
			} else {
				b.WriteRune(' ')
			}
		}
	}

	fields := strings.Fields(b.String())
	tokens := make([]string, 0, len(fields))
	for _, field := range fields {
		switch field {
		case "the", "a", "an":
			continue
		}
		if digits, ok := romanNumerals[field]; ok {
			field = digits
		}
		tokens = append(tokens, field)
	}

	// A title made only of articles ("A") still has to compare as itself.
	if len(tokens) == 0 {
		return fields
	}
	return tokens
}

// TitlesAgree reports whether a title parsed from a file name and a TMDB title
// are close enough to be the same film.
func TitlesAgree(fileTitle, tmdbTitle string) bool {
	return movieTitleSimilarity(fileTitle, tmdbTitle) >= minTitleSimilarity
}

// RuntimesAgree reports whether a file's running time confirms a film's, in
// minutes. False when either is unknown.
func RuntimesAgree(fileMinutes, tmdbMinutes int) bool {
	return runtimeScore(fileMinutes, tmdbMinutes) > 0
}

// RuntimesConflict reports whether a file's running time rules a film out.
// It and RuntimesAgree are both false across the band where cuts of the same
// film differ.
func RuntimesConflict(fileMinutes, tmdbMinutes int) bool {
	return runtimeScore(fileMinutes, tmdbMinutes) < 0
}
