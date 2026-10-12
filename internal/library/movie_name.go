package library

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Movie file names.
//
// Everything a release name carries after its year or its first quality tag is
// noise ("Extended.1080p.BluRay.x264-GROUP"), so the title is what comes
// before whichever appears first. That replaces stripping tags one by one,
// which removed them from inside words as well ("Birthday" lost its "hd") and
// could never list every tag in use.

var (
	// A tag only counts when it stands alone between separators.
	movieTagRegex = regexp.MustCompile(`(?i)(?:^|[\s._\-\[\(])(?:2160p|1080p|1080i|720p|576p|480p|4k|uhd|hdr10|hdr|blu-?ray|bdrip|brrip|bdremux|remux|webrip|web-?dl|hdtv|dvdrip|x264|x265|h\.?264|h\.?265|hevc|xvid|divx|aac|ac3|dts|atmos|truehd|hd)(?:$|[\s._\-\]\)])`)

	movieYearRegex   = regexp.MustCompile(`(?:19|20)\d{2}`)
	imdbIDRegex      = regexp.MustCompile(`(?i)\btt\d{7,8}\b`)
	bracketedRegex   = regexp.MustCompile(`\([^)]*\)|\[[^\]]*\]`)
	leadingOrderRegx = regexp.MustCompile(`^0\d\s+`)
	spacesRegex      = regexp.MustCompile(`\s+`)

	// Edition names that trail a title when the name has no year to cut at,
	// and the marks of a second copy of the same film ("v4", "T00",
	// "Compressed").
	editionRegex = regexp.MustCompile(`(?i)\s+(?:(?:the\s+)?(?:director'?s|extended|final|theatrical|unrated|uncut|special|collector'?s|ultimate|anniversary)\s+(?:cut|edition)|extended|unrated|remastered|uncut|imax|proper|repack|limited|compressed|v\d+|t\d{2})$`)

	// A commentary track kept beside its film: "Gremlins Commentary 2",
	// "Shaun Of The Dead Com1", "ComAustin Powers2".
	commentaryRegex       = regexp.MustCompile(`(?i)\bcommentary\b|[\s._-]com\s*\d*$`)
	commentaryPrefixRegex = regexp.MustCompile(`^Com\d*[A-Z]`)

	// A bonus clip kept beside its film: "Aladdin (1992) - Deleted",
	// "Showboat theatrical Trailer", "..._preview". Only at the end of the
	// name, so "Trailer Park Boys" stays a film.
	bonusClipRegex = regexp.MustCompile(`(?i)[\s._-](?:deleted(?:\s+scenes?)?|featurettes?|(?:theatrical\s+)?trailer|bloopers?|gag\s+reel|behind\s+the\s+scenes|preview)$`)

	// Top-level folders that hold television, whatever the source is called.
	tvFolderNames = map[string]bool{
		"tv": true, "tv shows": true, "tv show": true, "tv series": true,
		"television": true, "shows": true, "series": true,
	}
	// Words in a source's own path that say it isn't a film collection.
	nonMovieSourceWords = map[string]bool{
		"tv": true, "television": true, "show": true, "shows": true,
		"series": true, "episodes": true,
	}
	wordSplitRegex = regexp.MustCompile(`[^a-z0-9]+`)

	// Names a disc ripper gives a file, which say nothing about the film.
	genericNameRegex = regexp.MustCompile(`(?i)^(?:title|movie|feature|main|video|disc|dvd|bluray)?[\s._\-]*t?\d*$`)

	// "Charade (1963)" — the folder layout most libraries use.
	folderYearRegex = regexp.MustCompile(`^(.+?)\s*\(((?:19|20)\d{2})\)`)
)

// MovieNameFromPath extracts a movie's title and release year from its file
// path. year is 0 when the name doesn't carry one.
//
// The file name is preferred. Its folder fills in when the file name is a
// ripper's placeholder ("title_t00.mkv") or names the same film without a
// year, since "Title (Year)/" folders are often more careful than the files
// inside them.
func MovieNameFromPath(filePath string) (title string, year int) {
	name := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	title, year = parseMovieName(name)

	folder := filepath.Base(filepath.Dir(filePath))
	if m := folderYearRegex.FindStringSubmatch(folder); m != nil {
		folderTitle, _ := parseMovieName(m[1])
		folderYear, _ := strconv.Atoi(m[2])
		if folderTitle != "" {
			if title == "" || genericNameRegex.MatchString(name) {
				return folderTitle, folderYear
			}
			if year == 0 && sameLetters(title, folderTitle) {
				year = folderYear
			}
		}
	}

	if title == "" {
		title = name
	}
	return title, year
}

// MovieFolderTitle returns the title of the folder a movie file sits in, when
// that folder names the film more fully than the file does: "Scott Pilgrim vs
// the World/Scott Pilgrim.m4v". It is "" otherwise, including for folders that
// hold a collection ("Harry Potter/...") or name something else entirely,
// since the file is then the better guide.
func MovieFolderTitle(filePath string) string {
	fileTitle, _ := MovieNameFromPath(filePath)
	folderTitle, _ := parseMovieName(filepath.Base(filepath.Dir(filePath)))

	fileWords := titleWords(fileTitle)
	folderWords := titleWords(folderTitle)
	if len(fileWords) == 0 || len(folderWords) <= len(fileWords) {
		return ""
	}
	inFolder := make(map[string]bool, len(folderWords))
	for _, word := range folderWords {
		inFolder[word] = true
	}
	for _, word := range fileWords {
		if !inFolder[word] {
			return ""
		}
	}
	return folderTitle
}

// IsMovieFeature reports whether a file is a film in its own right, going by
// where it is and what it is called. It is false for anything in a television
// or extras source, anything under a top-level TV folder, and commentary
// tracks and bonus clips kept in a film's folder. Such files should never be
// matched to a film: the best they can do is borrow the wrong poster.
func IsMovieFeature(sourcePath, filePath string) bool {
	if isExtrasSource(sourcePath) {
		return false
	}
	// The source's own name, and the folder it sits in.
	for _, part := range []string{filepath.Base(sourcePath), filepath.Base(filepath.Dir(sourcePath))} {
		for _, word := range titleWords(part) {
			if nonMovieSourceWords[word] {
				return false
			}
		}
	}

	if rel, err := filepath.Rel(sourcePath, filePath); err == nil {
		if top := strings.SplitN(filepath.ToSlash(rel), "/", 2); len(top) == 2 && tvFolderNames[strings.ToLower(top[0])] {
			return false
		}
	}

	name := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	// Quality tags trail the name and would hide a marker in front of them.
	if loc := movieTagRegex.FindStringIndex(name); loc != nil && loc[0] > 0 {
		name = name[:loc[0]]
	}
	name = strings.TrimSpace(bracketedRegex.ReplaceAllString(name, " "))

	return !commentaryRegex.MatchString(name) &&
		!commentaryPrefixRegex.MatchString(name) &&
		!bonusClipRegex.MatchString(name)
}

func titleWords(title string) []string {
	var words []string
	for _, word := range wordSplitRegex.Split(strings.ToLower(title), -1) {
		if word != "" {
			words = append(words, word)
		}
	}
	return words
}

// parseMovieName parses a file or folder name with its extension removed.
func parseMovieName(name string) (title string, year int) {
	name = imdbIDRegex.ReplaceAllString(name, " ")

	// Drop everything from the first quality tag on. A tag at the very start
	// is left alone: there would be no title left.
	if loc := movieTagRegex.FindStringIndex(name); loc != nil && loc[0] > 0 {
		name = name[:loc[0]]
	}

	if start, end, ok := findMovieYear(name); ok {
		year, _ = strconv.Atoi(name[start:end])
		before := cleanMovieTitle(name[:start])
		if before != "" {
			return before, year
		}
		// "(2010) Title": the year leads, so the title follows it.
		return cleanMovieTitle(name[end:]), year
	}

	return cleanMovieTitle(name), 0
}

// findMovieYear locates the release year in a name, if it has one.
//
// Titles contain years too ("2001: A Space Odyssey", "Blade Runner 2049",
// "1917"), so the first year-shaped number is not good enough:
//   - a year in brackets is always the release year
//   - otherwise the last one is, since the release year follows the title
//   - unless it opens the name, where it is the title itself
//   - and a year that hasn't happened yet is never a release year
func findMovieYear(name string) (start, end int, ok bool) {
	latest := time.Now().Year() + 1
	bareStart, bareEnd := -1, -1
	bracketStart, bracketEnd := -1, -1

	for _, loc := range movieYearRegex.FindAllStringIndex(name, -1) {
		s, e := loc[0], loc[1]
		if s > 0 && isDigit(name[s-1]) || e < len(name) && isDigit(name[e]) {
			continue // part of a longer number
		}
		if y, _ := strconv.Atoi(name[s:e]); y > latest {
			continue
		}
		bracketed := s > 0 && e < len(name) &&
			(name[s-1] == '(' && name[e] == ')' || name[s-1] == '[' && name[e] == ']')
		if bracketed {
			bracketStart, bracketEnd = s, e
		} else {
			bareStart, bareEnd = s, e
		}
	}

	if bracketStart >= 0 {
		return bracketStart, bracketEnd, true
	}
	if bareStart >= 0 && strings.Trim(name[:bareStart], " ._-") != "" {
		return bareStart, bareEnd, true
	}
	return 0, 0, false
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

func cleanMovieTitle(title string) string {
	title = bracketedRegex.ReplaceAllString(title, " ")
	title = strings.NewReplacer(".", " ", "_", " ", "-", " ", "(", " ", ")", " ", "[", " ", "]", " ").Replace(title)
	title = strings.TrimSpace(spacesRegex.ReplaceAllString(title, " "))
	title = leadingOrderRegx.ReplaceAllString(title, "")

	// Keep the words if they are all there is: "The Final Cut" is a film.
	if stripped := strings.TrimSpace(editionRegex.ReplaceAllString(" "+title, "")); stripped != "" {
		title = stripped
	}
	return title
}

// sameLetters reports whether two titles are the same ignoring case, spacing
// and punctuation.
func sameLetters(a, b string) bool {
	return lettersOnly(a) == lettersOnly(b)
}

func lettersOnly(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
