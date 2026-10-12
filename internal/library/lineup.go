package library

import (
	"fmt"
	"strings"

	"github.com/stephencjuliano/media-server/internal/db"
)

// A lineup is a set of channels written down: what each one is called and
// what it draws on. Applying it creates the channels that are missing and
// refreshes the ones that exist, so the file is the lineup and can be edited
// and applied again.
//
// A channel draws on whole shows directly. Anything chosen by rule (films of
// a genre or era, bonus clips of a kind) is gathered into a playlist the
// channel plays from, named after the channel. Re-applying refills those
// playlists, which is how a film added later finds its way onto the channels
// it belongs on.

// LineupSpec is the file.
type LineupSpec struct {
	Channels []LineupChannel `json:"channels"`
}

// LineupChannel is one channel.
type LineupChannel struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Icon        string         `json:"icon,omitempty"`
	Sources     []LineupSource `json:"sources"`
}

// LineupSource is one thing a channel draws on. Set exactly one of Show,
// Films and Bonus.
type LineupSource struct {
	// Show is a show's title. Seasons narrows it; Version is "main" (the
	// default), "commentary" or "both"; Extras adds the show's bonus clips
	// of those kinds.
	Show    string   `json:"show,omitempty"`
	Seasons []int    `json:"seasons,omitempty"`
	Version string   `json:"version,omitempty"`
	Extras  []string `json:"extras,omitempty"`

	Films *FilmFilter  `json:"films,omitempty"`
	Bonus *BonusFilter `json:"bonus,omitempty"`
}

// FilmFilter picks films. A film is in if its title is named in Titles, or
// if it passes the genre and year rules; an empty filter is every film. The
// length limits apply either way.
type FilmFilter struct {
	Genres     []string `json:"genres,omitempty"`     // any of these
	NotGenres  []string `json:"not_genres,omitempty"` // and none of these
	YearFrom   int      `json:"year_from,omitempty"`
	YearTo     int      `json:"year_to,omitempty"`
	Titles     []string `json:"titles,omitempty"` // matched anywhere in the title
	MinMinutes int      `json:"min_minutes,omitempty"`
	MaxMinutes int      `json:"max_minutes,omitempty"`
}

// BonusFilter picks bonus clips by kind. Parent is "film", "show" or empty
// for both.
type BonusFilter struct {
	Categories []string `json:"categories"`
	Parent     string   `json:"parent,omitempty"`
	MinMinutes int      `json:"min_minutes,omitempty"`
	MaxMinutes int      `json:"max_minutes,omitempty"`
}

// LineupReport says what applying a lineup did, or would do, for one channel.
type LineupReport struct {
	Channel string   `json:"channel"`
	Action  string   `json:"action"` // "create" or "update"
	Shows   []string `json:"shows,omitempty"`
	Films   []string `json:"films,omitempty"`
	Bonus   int      `json:"bonus_clips,omitempty"`
	// Set once the channel's schedule has been built.
	Items int     `json:"items,omitempty"`
	Hours float64 `json:"hours,omitempty"`
	// Things in the file that matched nothing.
	Problems []string `json:"problems,omitempty"`
}

// playlistFor names the playlist that holds a channel's chosen films and
// clips.
func playlistFor(channel string) string {
	return "Channel: " + channel
}

// ApplyLineup works out each channel in the spec and, with apply set, creates
// or refreshes it. Channels not in the spec are never touched.
func ApplyLineup(database *db.DB, spec LineupSpec, apply bool) ([]LineupReport, error) {
	films, err := lineupFilms(database)
	if err != nil {
		return nil, err
	}
	owner, err := database.ChannelOwner()
	if err != nil {
		return nil, err
	}

	var reports []LineupReport
	for _, channel := range spec.Channels {
		report := LineupReport{Channel: channel.Name, Action: "create"}
		existing, err := database.FindChannelByName(channel.Name)
		if err == nil {
			report.Action = "update"
		} else if err != db.ErrNotFound {
			return nil, err
		}

		type showSource struct {
			id      int64
			options *db.ChannelSourceOptions
		}
		var shows []showSource
		var picked []db.LineupItem
		seen := map[string]bool{}
		add := func(item db.LineupItem) bool {
			key := fmt.Sprintf("%s:%d", item.Type, item.ID)
			if seen[key] {
				return false
			}
			seen[key] = true
			picked = append(picked, item)
			return true
		}

		for _, source := range channel.Sources {
			switch {
			case source.Show != "":
				show, err := database.GetTVShowByTitle(source.Show)
				if err != nil || show == nil {
					report.Problems = append(report.Problems, fmt.Sprintf("no show called %q", source.Show))
					continue
				}
				var options *db.ChannelSourceOptions
				if len(source.Seasons) > 0 || source.Version != "" || len(source.Extras) > 0 {
					options = &db.ChannelSourceOptions{
						Seasons: source.Seasons, VersionMode: source.Version, ExtrasCategories: source.Extras,
					}
				}
				shows = append(shows, showSource{show.ID, options})
				report.Shows = append(report.Shows, show.Title)

			case source.Films != nil:
				matched := 0
				for _, film := range films {
					if source.Films.matches(film) && add(db.LineupItem{
						ID: film.ID, Type: db.MediaTypeMovie, Title: film.Title, Duration: film.Duration,
					}) {
						report.Films = append(report.Films, film.Title)
						matched++
					}
				}
				if matched == 0 {
					report.Problems = append(report.Problems, "a films rule matched nothing new")
				}

			case source.Bonus != nil:
				clips, err := database.LineupExtras(source.Bonus.Categories, source.Bonus.Parent)
				if err != nil {
					return nil, err
				}
				matched := 0
				for _, clip := range clips {
					minutes := clip.Duration / 60
					if minutes < source.Bonus.MinMinutes || (source.Bonus.MaxMinutes > 0 && minutes > source.Bonus.MaxMinutes) {
						continue
					}
					if add(clip) {
						matched++
					}
				}
				report.Bonus += matched
				if matched == 0 {
					report.Problems = append(report.Problems, "a bonus rule matched nothing new")
				}

			default:
				report.Problems = append(report.Problems, "a source with nothing in it")
			}
		}

		if len(shows) == 0 && len(picked) == 0 {
			report.Problems = append(report.Problems, "nothing to air, so the channel is left as it is")
			reports = append(reports, report)
			continue
		}
		if !apply {
			reports = append(reports, report)
			continue
		}

		target := existing
		if target == nil {
			target, err = database.CreateChannel(owner, channel.Name, channel.Description, channel.Icon)
			if err != nil {
				return nil, err
			}
		}
		if err := database.ClearChannelSources(target.ID); err != nil {
			return nil, err
		}
		for _, show := range shows {
			id := show.id
			if _, err := database.AddChannelSource(target.ID, db.ChannelSourceShow, &id, "", 1, true, show.options); err != nil {
				return nil, err
			}
		}
		if len(picked) > 0 {
			playlistID, err := database.EnsurePlaylist(target.UserID, playlistFor(channel.Name),
				"Filled by the channel lineup. Edits here are replaced when the lineup is applied.")
			if err != nil {
				return nil, err
			}
			if err := database.ReplacePlaylistItems(playlistID, picked); err != nil {
				return nil, err
			}
			if _, err := database.AddChannelSource(target.ID, db.ChannelSourcePlaylist, &playlistID, "", 1, true, nil); err != nil {
				return nil, err
			}
		}
		if err := database.GenerateChannelSchedule(target.ID); err != nil {
			return nil, err
		}

		items, seconds, err := database.ChannelScheduleTotals(target.ID)
		if err != nil {
			return nil, err
		}
		report.Items = items
		report.Hours = float64(seconds/360) / 10
		reports = append(reports, report)
	}
	return reports, nil
}

// lineupFilms returns the library's films, one entry per film however many
// copies there are, leaving out anything that is not a film in its own right
// and anything with no match to take a genre or year from.
func lineupFilms(database *db.DB) ([]*db.Media, error) {
	all, err := database.GetMediaByType(db.MediaTypeMovie, 1_000_000, 0)
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

	var features []*db.Media
	for _, movie := range all {
		if movie.TMDbID != 0 && movie.Duration > 0 && IsMovieFeature(sourcePaths[movie.SourceID], movie.FilePath) {
			features = append(features, movie)
		}
	}
	entries, _ := GroupMovieCopies(features)
	return entries, nil
}

func (f *FilmFilter) matches(film *db.Media) bool {
	minutes := film.Duration / 60
	if minutes < f.MinMinutes || (f.MaxMinutes > 0 && minutes > f.MaxMinutes) {
		return false
	}

	title := strings.ToLower(film.Title)
	for _, wanted := range f.Titles {
		if wanted != "" && strings.Contains(title, strings.ToLower(wanted)) {
			return true
		}
	}

	// Titles alone are a hand-picked list. Without them, or with rules as
	// well, the rules decide.
	hasRules := len(f.Genres) > 0 || len(f.NotGenres) > 0 || f.YearFrom > 0 || f.YearTo > 0
	if !hasRules {
		return len(f.Titles) == 0
	}

	genres := map[string]bool{}
	for _, genre := range strings.Split(film.Genres, ",") {
		genres[strings.ToLower(strings.TrimSpace(genre))] = true
	}
	for _, unwanted := range f.NotGenres {
		if genres[strings.ToLower(unwanted)] {
			return false
		}
	}
	if len(f.Genres) > 0 {
		any := false
		for _, wanted := range f.Genres {
			if genres[strings.ToLower(wanted)] {
				any = true
			}
		}
		if !any {
			return false
		}
	}
	if (f.YearFrom > 0 && film.Year < f.YearFrom) || (f.YearTo > 0 && (film.Year == 0 || film.Year > f.YearTo)) {
		return false
	}
	return true
}
