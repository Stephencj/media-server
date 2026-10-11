package library

import "testing"

func TestMovieNameFromPath(t *testing.T) {
	tests := []struct {
		path  string
		title string
		year  int
	}{
		// The common layouts.
		{"/m/Charade (1963).mkv", "Charade", 1963},
		{"/m/Charade.1963.1080p.BluRay.x264-GROUP.mkv", "Charade", 1963},
		{"/m/Charade [1963].mp4", "Charade", 1963},
		{"/m/Charade.mkv", "Charade", 0},
		{"/m/Charade (1080p HD).m4v", "Charade", 0},
		{"/m/His Girl Friday (1940) (1080p HD).m4v", "His Girl Friday", 1940},
		{"/m/The.Thing.tt0084787.1982.mkv", "The Thing", 1982},
		{"/m/01 Charade (1963).mkv", "Charade", 1963},

		// A year at the very end of the name.
		{"/m/Charade.1963.mkv", "Charade", 1963},
		{"/m/Charade 1963.mkv", "Charade", 1963},

		// Titles that contain a year-shaped number.
		{"/m/2001.A.Space.Odyssey.1968.mkv", "2001 A Space Odyssey", 1968},
		{"/m/2001 A Space Odyssey (1968).mkv", "2001 A Space Odyssey", 1968},
		{"/m/2001.A.Space.Odyssey.mkv", "2001 A Space Odyssey", 0},
		{"/m/Blade.Runner.2049.2017.2160p.UHD.mkv", "Blade Runner 2049", 2017},
		{"/m/Blade Runner 2049.mkv", "Blade Runner 2049", 0},
		{"/m/1917 (2019).mkv", "1917", 2019},
		{"/m/1917.mkv", "1917", 0},
		{"/m/2012.2009.720p.mkv", "2012", 2009},
		{"/m/Wonder Woman 1984 (2020).mkv", "Wonder Woman 1984", 2020},

		// Quality tags are whole words only, never letters inside one.
		{"/m/The Birthday Party (1968).mkv", "The Birthday Party", 1968},
		{"/m/Isaac.2008.DVDRip.mkv", "Isaac", 2008},
		{"/m/Withdrawn.mkv", "Withdrawn", 0},

		// Everything after the year or first tag is release noise.
		{"/m/Blade.Runner.1982.The.Final.Cut.1080p.mkv", "Blade Runner", 1982},
		{"/m/Blade Runner The Final Cut.mkv", "Blade Runner", 0},
		{"/m/Charade.Remastered.1080p.WEB-DL.AAC.mkv", "Charade", 0},
		{"/m/Charade (1963) [Criterion].mkv", "Charade", 1963},
		{"/m/The Final Cut (2004).mkv", "The Final Cut", 2004},
		{"/m/The Final Cut.mkv", "The Final Cut", 0},

		// A year that leads the name in brackets is still the year.
		{"/m/(1963) Charade.mkv", "Charade", 1963},

		// The folder fills in for ripper placeholders and missing years.
		{"/m/Charade (1963)/title_t00.mkv", "Charade", 1963},
		{"/m/Charade (1963)/Charade.mkv", "Charade", 1963},
		{"/m/The Kid (1921)/The.Kid.1080p.mkv", "The Kid", 1921},
		{"/m/1917 (2019)/1917.mkv", "1917", 2019},
		// ...but never overrides a file that names a different film.
		{"/m/Thin Man Collection (1934)/After the Thin Man.mkv", "After the Thin Man", 0},
		{"/m/Charade (1963)/Charade.1953.mkv", "Charade", 1953},
	}

	for _, tt := range tests {
		title, year := MovieNameFromPath(tt.path)
		if title != tt.title || year != tt.year {
			t.Errorf("MovieNameFromPath(%q) = %q, %d; want %q, %d", tt.path, title, year, tt.title, tt.year)
		}
	}
}

// Movies route through the new parser; episodes keep the existing one.
func TestParseFilenameRouting(t *testing.T) {
	title, year, mediaType, season, episode := parseFilename("/m/Blade.Runner.2049.2017.mkv")
	if title != "Blade Runner 2049" || year != 2017 || mediaType != "movie" || season != 0 || episode != 0 {
		t.Errorf("movie parsed as %q, %d, %q, S%dE%d", title, year, mediaType, season, episode)
	}

	title, _, mediaType, season, episode = parseFilename("/tv/Breaking.Bad.S01E02.720p.mkv")
	if title != "Breaking Bad" || mediaType != "tvshow" || season != 1 || episode != 2 {
		t.Errorf("episode parsed as %q, %q, S%dE%d", title, mediaType, season, episode)
	}
}
