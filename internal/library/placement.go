package library

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/stephencjuliano/media-server/internal/db"
)

// File placement: what a file is, going by where it sits in the library.
//
// A library someone organised by hand says most of what there is to know in
// its folders. Everything under "TV Shows/Taskmaster/Series 04" is an episode
// of Taskmaster, series 4, whether or not its name carries an S04E01 marker,
// and a file called "Gremlins Commentary 2" beside "Gremlins.m4v" is a
// commentary on Gremlins, not a second film. Reading names alone gets both
// wrong, and gets them wrong as films.

// PlacementKind is the role a file's location gives it.
type PlacementKind int

const (
	// PlaceFeature is a film.
	PlaceFeature PlacementKind = iota
	// PlaceFilmBonus is a commentary track or bonus clip kept in a film's
	// folder.
	PlaceFilmBonus
	// PlaceEpisode is an episode of the show whose folder it is in.
	PlaceEpisode
	// PlaceShowBonus is anything else in a show's folder: disc menus, "play
	// all" rips, previews, featurettes.
	PlaceShowBonus
	// PlaceOther is a video that is neither, such as a home recording in a
	// folder with no show or film structure.
	PlaceOther
)

// Placement is what a file's path says about it.
type Placement struct {
	Kind PlacementKind

	// Episodes and show bonus files.
	ShowTitle    string
	Season       int
	Episode      int
	EpisodeTitle string // from the file name, "" when it has none

	// Bonus files.
	Title    string
	Category db.ExtraCategory
}

var (
	seasonFolderRegex = regexp.MustCompile(`(?i)^(?:season|series)\s*(\d{1,3})$`)

	// A disc or menu rip rather than an episode: "S01D02 Play All",
	// "S03D02 - Character Profiles", "P05D01 Play All".
	discRipRegex = regexp.MustCompile(`(?i)^[sp]\d{1,2}d\d{1,2}\b`)

	// Episode markers, most specific first. Each captures the season where
	// the name has one, then the episode.
	episodeMarkers = []struct {
		re        *regexp.Regexp
		hasSeason bool
	}{
		{regexp.MustCompile(`(?i)\bs(\d{1,2})\s*e(\d{1,3})\b`), true},                                     // S01E02
		{regexp.MustCompile(`^[Ss](\d{2})(\d{2})\b`), true},                                               // S0502
		{regexp.MustCompile(`(?i)\b(?:season|series)\s*(\d{1,3})\s*[-,]?\s*episode\s*(\d{1,3})\b`), true}, // Series 19 Episode 01
		{regexp.MustCompile(`\b(\d{1,2})x(\d{2,3})\b`), true},                                             // 1x02
		{regexp.MustCompile(`(?i)\b(?:episode|ep|e)\s*(\d{1,3})\b`), false},                               // E01, Episode 3
		{regexp.MustCompile(`^(\d{1,3})[\s._-]+`), false},                                                 // 01 Title
	}

	tvSourceWords = map[string]bool{
		"tv": true, "television": true, "show": true, "shows": true,
		"series": true, "episodes": true,
	}
)

// PlaceFile reads a file's role from its path within its source.
func PlaceFile(sourcePath, filePath string) Placement {
	rel, err := filepath.Rel(sourcePath, filePath)
	if err != nil || strings.HasPrefix(rel, "..") {
		rel = filepath.Base(filePath)
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	folders, file := parts[:len(parts)-1], parts[len(parts)-1]
	name := strings.TrimSuffix(file, filepath.Ext(file))

	// Television: the source is named for it, or the file sits under a
	// top-level TV folder of a source that holds everything.
	inTV := false
	for _, part := range []string{filepath.Base(sourcePath), filepath.Base(filepath.Dir(sourcePath))} {
		for _, word := range titleWords(part) {
			if tvSourceWords[word] {
				inTV = true
			}
		}
	}
	if len(folders) > 0 && tvFolderNames[strings.ToLower(folders[0])] {
		inTV = true
		folders = folders[1:]
	}

	if inTV {
		// A show needs a folder to be named by. Loose files in a TV source
		// belong to no show.
		if len(folders) == 0 {
			return Placement{Kind: PlaceOther, Title: readableName(name)}
		}
		return placeShowFile(folders, name)
	}

	if !IsMovieFeature(sourcePath, filePath) {
		return Placement{Kind: PlaceFilmBonus, Title: readableName(name), Category: bonusCategory(name)}
	}
	return Placement{Kind: PlaceFeature}
}

func placeShowFile(folders []string, name string) Placement {
	p := Placement{ShowTitle: folders[0], Season: 1}

	seasonFromFolder := false
	for _, folder := range folders[1:] {
		if m := seasonFolderRegex.FindStringSubmatch(folder); m != nil {
			p.Season, _ = strconv.Atoi(m[1])
			seasonFromFolder = true
		}
	}

	bonus := func() Placement {
		p.Kind = PlaceShowBonus
		p.Title = readableName(name)
		p.Category = bonusCategory(name)
		return p
	}

	if discRipRegex.MatchString(name) || bonusClipRegex.MatchString(name) || commentaryRegex.MatchString(name) {
		return bonus()
	}

	for _, marker := range episodeMarkers {
		loc := marker.re.FindStringSubmatchIndex(name)
		if loc == nil {
			continue
		}
		groups := marker.re.FindStringSubmatch(name)
		if marker.hasSeason {
			season, _ := strconv.Atoi(groups[1])
			p.Episode, _ = strconv.Atoi(groups[2])
			// The name's own season wins, except that "season 0" in a
			// season folder means the name simply didn't say.
			if season > 0 || !seasonFromFolder {
				p.Season = season
			}
		} else {
			p.Episode, _ = strconv.Atoi(groups[1])
		}
		if p.Episode == 0 && !marker.hasSeason {
			continue
		}

		// Whatever follows the marker is the episode's title, unless the
		// show's own name is all that precedes or follows it.
		title := readableName(name[loc[1]:])
		if title == "" {
			title = readableName(name[:loc[0]])
			if sameLetters(title, p.ShowTitle) {
				title = ""
			}
		}
		p.Kind = PlaceEpisode
		p.EpisodeTitle = title
		return p
	}

	return bonus()
}

// bonusCategory sorts a bonus file by its name.
func bonusCategory(name string) db.ExtraCategory {
	if commentaryRegex.MatchString(name) || commentaryPrefixRegex.MatchString(name) {
		return db.ExtraCategoryCommentary
	}
	return detectCategory(name)
}

// readableName tidies a file name for display without trying to turn it into
// a film title: quality tags and brackets go, the words stay.
func readableName(name string) string {
	if loc := movieTagRegex.FindStringIndex(name); loc != nil && loc[0] > 0 {
		name = name[:loc[0]]
	}
	name = bracketedRegex.ReplaceAllString(name, " ")
	name = strings.NewReplacer(".", " ", "_", " ", "(", " ", ")", " ").Replace(name)
	name = strings.TrimSpace(spacesRegex.ReplaceAllString(name, " "))
	return strings.TrimSpace(strings.Trim(name, "-"))
}
