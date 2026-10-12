package library

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/stephencjuliano/media-server/internal/db"
	"github.com/stephencjuliano/media-server/pkg/tmdb"
)

// File health: files that look like a bad rip or conversion, found from what
// the library already knows about them. Nothing here opens a file.
//
// "definite" means the file is wrong on its face: no running time, no video,
// or far shorter than the film it certainly is. "potential" means a sign
// worth a look: somewhat short, a name the converter cut off, a duplicate
// copy, a leftover preview, a starved bitrate.

const (
	SeverityDefinite  = "definite"
	SeverityPotential = "potential"
)

// FileIssue is one file with something wrong, or possibly wrong, with it.
type FileIssue struct {
	Path     string   `json:"path"`
	Kind     string   `json:"kind"` // movie, episode, extra
	Title    string   `json:"title"`
	Severity string   `json:"severity"`
	Reasons  []string `json:"reasons"`
	// Minutes. ExpectedMinutes is the matched film's running time, 0 when
	// there is nothing to compare with.
	DurationMinutes int    `json:"duration_minutes"`
	ExpectedMinutes int    `json:"expected_minutes,omitempty"`
	Resolution      string `json:"resolution,omitempty"`
	SizeBytes       int64  `json:"size_bytes"`
}

// Below this a video is starved even at DVD size.
const starvedKbps = 400

// CheckFileHealth examines every file and returns those with a problem, the
// definite ones first. sourcePaths maps a source ID to its root folder.
func CheckFileHealth(files []db.LibraryFile, sourcePaths map[int64]string) []FileIssue {
	copies := countCopies(files, sourcePaths)

	var issues []FileIssue
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f.FilePath), filepath.Ext(f.FilePath))
		minutes := runtimeMinutes(f.Duration)
		issue := FileIssue{
			Path: f.FilePath, Kind: string(f.Kind), Title: f.Title,
			DurationMinutes: minutes, Resolution: f.Resolution, SizeBytes: f.FileSize,
		}
		definite := func(reason string) {
			issue.Severity = SeverityDefinite
			issue.Reasons = append(issue.Reasons, reason)
		}
		potential := func(reason string) {
			if issue.Severity == "" {
				issue.Severity = SeverityPotential
			}
			issue.Reasons = append(issue.Reasons, reason)
		}

		if f.Duration <= 0 {
			definite("No running time could be read from the file.")
		}
		if f.VideoCodec == "" {
			definite("No video stream was found in the file.")
		}

		// Length against the film, only where the match is not in doubt:
		// otherwise a wrong match would be reported as a bad file.
		isFeature := f.Kind == db.MediaTypeMovie && IsMovieFeature(sourcePaths[f.SourceID], f.FilePath)
		if isFeature && f.TMDbID != 0 && f.MatchedRuntime > 0 && minutes > 0 && matchIsSound(f) {
			issue.ExpectedMinutes = f.MatchedRuntime
			short := f.MatchedRuntime - minutes
			switch {
			case short > 10 && minutes*100 < f.MatchedRuntime*70:
				definite(fmt.Sprintf("Runs %s; the film runs %s. The file stops early or is incomplete.",
					FormatMinutes(minutes), FormatMinutes(f.MatchedRuntime)))
			case short > 10 && minutes*100 < f.MatchedRuntime*90:
				potential(fmt.Sprintf("Runs %s; the film runs %s.",
					FormatMinutes(minutes), FormatMinutes(f.MatchedRuntime)))
			case -short > 15 && minutes*100 > f.MatchedRuntime*120:
				potential(fmt.Sprintf("Runs %s, well over the film's %s. It may hold more than the film.",
					FormatMinutes(minutes), FormatMinutes(f.MatchedRuntime)))
			}
		}

		if strings.Count(name, "(") > strings.Count(name, ")") {
			potential("The file name is cut off mid-word, the mark of an older conversion.")
		}
		if strings.HasSuffix(strings.ToLower(name), "_preview") {
			potential("A preview clip left behind by a conversion.")
		}
		if discRipRegex.MatchString(name) && strings.Contains(strings.ToLower(name), "play all") {
			potential("A whole disc in one file, not split into episodes.")
		}
		if isFeature {
			if n := copies[copyKey(f.FilePath)]; n > 1 {
				potential(fmt.Sprintf("One of %d copies of this film in the same folder.", n))
			}
		}
		if f.Duration > 0 && f.FileSize > 0 {
			if kbps := f.FileSize * 8 / int64(f.Duration) / 1000; kbps < starvedKbps {
				potential(fmt.Sprintf("Very low bitrate (%d kbps).", kbps))
			}
		}

		if issue.Severity != "" {
			issues = append(issues, issue)
		}
	}

	sort.SliceStable(issues, func(a, b int) bool {
		if issues[a].Severity != issues[b].Severity {
			return issues[a].Severity == SeverityDefinite
		}
		return issues[a].Path < issues[b].Path
	})
	return issues
}

// matchIsSound reports whether a movie's TMDB match agrees with its file's
// name and year, so that its running time can be held against the file.
func matchIsSound(f db.LibraryFile) bool {
	title, year := MovieNameFromPath(f.FilePath)
	if !tmdb.TitlesAgree(title, f.MatchedTitle) && !tmdb.TitlesAgree(MovieFolderTitle(f.FilePath), f.MatchedTitle) {
		return false
	}
	return year == 0 || f.MatchedYear == 0 || absInt(year-f.MatchedYear) <= 1
}

// copyKey identifies a film within its folder, so that "Shrek 2", "Shrek 2
// HD" and "Shrek 2 T00" count as copies of one film.
func copyKey(filePath string) string {
	title, year := MovieNameFromPath(filePath)
	return fmt.Sprintf("%s\x00%s\x00%d", filepath.Dir(filePath), lettersOnly(title), year)
}

func countCopies(files []db.LibraryFile, sourcePaths map[int64]string) map[string]int {
	copies := map[string]int{}
	for _, f := range files {
		if f.Kind == db.MediaTypeMovie && IsMovieFeature(sourcePaths[f.SourceID], f.FilePath) {
			copies[copyKey(f.FilePath)]++
		}
	}
	return copies
}

// LibraryFileHealth runs CheckFileHealth over the whole library.
func LibraryFileHealth(database *db.DB) ([]FileIssue, error) {
	files, err := database.ListLibraryFiles()
	if err != nil {
		return nil, err
	}
	sources, err := database.GetAllMediaSources()
	if err != nil {
		return nil, err
	}
	sourcePaths := make(map[int64]string, len(sources))
	for _, source := range sources {
		sourcePaths[source.ID] = source.Path
	}
	return CheckFileHealth(files, sourcePaths), nil
}
