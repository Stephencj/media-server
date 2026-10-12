package library

import (
	"fmt"

	"github.com/stephencjuliano/media-server/internal/db"
)

// TidyStep is one movie row that a scan will move to where its folder says it
// belongs.
type TidyStep struct {
	Path   string `json:"path"`
	Title  string `json:"title"`
	Action string `json:"action"` // "remove duplicate", "to episode", "to extra"
	Detail string `json:"detail"`
}

// PlanTidy lists what the next scan will move out of Movies, without changing
// anything. It reads only the database.
func (s *Scanner) PlanTidy() ([]TidyStep, error) {
	movies, err := s.db.GetMediaByType(db.MediaTypeMovie, 1_000_000, 0)
	if err != nil {
		return nil, err
	}

	sourcePaths := map[int64]string{}
	var steps []TidyStep
	for _, media := range movies {
		sourcePath, known := sourcePaths[media.SourceID]
		if !known {
			if source, err := s.db.GetMediaSourceByID(media.SourceID); err == nil {
				sourcePath = source.Path
			}
			sourcePaths[media.SourceID] = sourcePath
		}

		step := TidyStep{Path: media.FilePath, Title: media.Title}
		place := PlaceFile(sourcePath, media.FilePath)

		switch place.Kind {
		case PlaceEpisode:
			if _, err := s.db.GetEpisodeByFilePath(media.FilePath); err == nil {
				step.Action = "remove duplicate"
				step.Detail = fmt.Sprintf("already an episode of %s", place.ShowTitle)
			} else {
				step.Action = "to episode"
				step.Detail = fmt.Sprintf("%s S%02dE%02d", place.ShowTitle, place.Season, place.Episode)
			}
		case PlaceFilmBonus, PlaceShowBonus:
			if _, err := s.db.GetExtraByFilePath(media.FilePath); err == nil {
				step.Action = "remove duplicate"
				step.Detail = "already an extra"
				break
			}
			step.Action = "to extra"
			owner := place.ShowTitle
			if place.Kind == PlaceFilmBonus {
				owner = "no film found in its folder"
				if film := s.filmForBonus(sourcePath, media.FilePath); film != nil {
					owner = film.Title
				}
			}
			step.Detail = fmt.Sprintf("%s, %s", place.Category, owner)
		default:
			continue
		}
		steps = append(steps, step)
	}
	return steps, nil
}
