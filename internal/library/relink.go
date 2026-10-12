package library

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/stephencjuliano/media-server/internal/db"
)

// Relink is one library entry whose file was replaced by a conversion of
// itself: the same name in the same folder with a different extension.
type Relink struct {
	Kind    string `json:"kind"`
	Title   string `json:"title"`
	OldPath string `json:"old_path"`
	NewPath string `json:"new_path"`
	Error   string `json:"error,omitempty"`
}

// Containers to look for in place of a missing file, the ones that play
// directly first.
var relinkExtensions = []string{".mp4", ".m4v", ".mov", ".mkv", ".webm", ".avi", ".ts", ".m2ts", ".wmv", ".flv"}

// RelinkMovedFiles finds entries whose file is gone and whose folder holds
// the same name in another container, and with apply set, points each entry
// at that file and re-reads its details.
//
// The entry itself is kept, so watch progress, playlists and channel
// schedules carry over. A scan alone would not do this: it would add the new
// file as a second entry and leave the old one pointing at nothing.
func (s *Scanner) RelinkMovedFiles(apply bool) ([]Relink, error) {
	return s.relinkMovedFiles(apply, s.metadataExtractor.ExtractFileMetadata)
}

func (s *Scanner) relinkMovedFiles(apply bool, probe func(string) (*db.MediaFile, error)) ([]Relink, error) {
	files, err := s.db.ListLibraryFiles()
	if err != nil {
		return nil, err
	}

	taken := make(map[string]bool, len(files))
	for _, f := range files {
		taken[f.FilePath] = true
	}

	var relinks []Relink
	for _, f := range files {
		if _, err := os.Stat(f.FilePath); err == nil {
			continue
		}
		replacement := replacementFor(f.FilePath, taken)
		if replacement == "" {
			continue
		}

		relink := Relink{Kind: string(f.Kind), Title: f.Title, OldPath: f.FilePath, NewPath: replacement}
		if apply {
			details, err := probe(replacement)
			if err == nil {
				details.FilePath = replacement
				err = s.db.UpdateFileDetails(f.Kind, f.ID, details)
			}
			if err != nil {
				relink.Error = err.Error()
			} else {
				taken[replacement] = true
			}
		}
		relinks = append(relinks, relink)
	}
	return relinks, nil
}

// replacementFor returns the file standing in for a missing one: same folder,
// same name, another extension, and not already some other entry's file.
func replacementFor(missing string, taken map[string]bool) string {
	stem := strings.TrimSuffix(missing, filepath.Ext(missing))
	for _, ext := range relinkExtensions {
		candidate := stem + ext
		if candidate == missing || taken[candidate] {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}
