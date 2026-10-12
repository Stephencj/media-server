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

		// Marks of a second copy of the same film.
		{"/m/The Wizard of Oz v4.mp4", "The Wizard of Oz", 0},
		{"/m/Shrek 2 T00.mp4", "Shrek 2", 0},
		{"/m/Shrek 2 HD.m4v", "Shrek 2", 0},
		{"/m/Trilogy/Compressed/The Two Towers Compressed.mp4", "The Two Towers", 0},
		{"/m/01 Trolls (1080p HD)(1).m4v", "Trolls", 0},
		{"/m/Ted (Unrated) (HD).mp4", "Ted", 0},
		{"/m/Star.Wars.1977.Despecialized.720p.x264.AC3.5.1.mp4", "Star Wars", 1977},
		{"/m/04 How to Train Your Dragon (1080p H.m4v", "How to Train Your Dragon", 0},

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

func TestIsMovieFeature(t *testing.T) {
	tests := []struct {
		source, path string
		want         bool
	}{
		{"/media/Movies", "/media/Movies/Gremlins/Gremlins.m4v", true},
		{"/media/Movies", "/media/Movies/Charade (1963).mkv", true},
		{"/media/Movies", "/media/Movies/Edward Scissor Hands/Edward Scissor Hands v1.m4v", true},
		{"/media/Movies", "/media/Movies/Trailer Park Boys The Movie.mkv", true},
		{"/media/Movies", "/media/Movies/The Interview (2014).mkv", true},
		{"/media/Movies", "/media/Movies/A Series of Unfortunate Events/A Series of Unfortunate Events.mkv", true},
		{"/media/Movies", "/media/Movies/Comanche Station (1960).mkv", true},
		{"/media/Movies", "/media/Movies/Showboat/Showboat.m4v", true},

		// Commentary tracks kept beside the film.
		{"/media/Movies", "/media/Movies/Gremlins/Gremlins Commentary 2.m4v", false},
		{"/media/Movies", "/media/Movies/Best in Show/Best In Show Commentary.m4v", false},
		{"/media/Movies", "/media/Movies/Shaun Of The Dead/Shaun Of The Dead Com1.m4v", false},
		{"/media/Movies", "/media/Movies/Austin Powers/ComAustin Powers2.m4v", false},
		{"/media/Movies", "/media/Movies/Scream 2/Scream2 Commentary (HD).m4v", false},

		// Bonus clips kept beside the film.
		{"/media/Movies", "/media/Movies/Aladdin (1992)/Aladdin (1992) - Deleted.mkv", false},
		{"/media/Movies", "/media/Movies/Aladdin (1992)/Aladdin (1992) - Featurette.mkv", false},
		{"/media/Movies", "/media/Movies/Showboat/Showboat theatrical Trailer.m4v", false},
		{"/media/Movies", "/media/Movies/Futurama/The Goonies_preview.m4v", false},

		// Sources and folders that aren't film collections.
		{"/media/TV Shows", "/media/TV Shows/Taskmaster/Series 04/01 A Fat Bald White Man (1080p HD).m4v", false},
		{"/media/TV Shows", "/media/TV Shows/Wilfred AU/01 There Is a Dog.mp4", false},
		{"/media/Stage Shows/Archive", "/media/Stage Shows/Archive/Grand Hotel.mp4", false},
		{"/media/Movie Special Features", "/media/Movie Special Features/Dracula/Dracula Archives.m4v", false},
		{"/media", "/media/TV Shows/Wilfred AU/01 There Is a Dog.mp4", false},
		{"/media", "/media/Movies/Gremlins/Gremlins.m4v", true},
	}

	for _, tt := range tests {
		if got := IsMovieFeature(tt.source, tt.path); got != tt.want {
			t.Errorf("IsMovieFeature(%q, %q) = %v, want %v", tt.source, tt.path, got, tt.want)
		}
	}
}

func TestMovieFolderTitle(t *testing.T) {
	tests := []struct{ path, want string }{
		// The folder says more about the same film.
		{"/m/Scott Pilgrim vs the World/Scott Pilgrim.m4v", "Scott Pilgrim vs the World"},
		{"/m/The Super Mario Bros Movie (2023)/Super Mario.m4v", "The Super Mario Bros Movie"},
		// The folder says the same, or less.
		{"/m/Gremlins/Gremlins.m4v", ""},
		{"/m/Rambo/Rambo II.m4v", ""},
		// The folder is a collection or a different name altogether.
		{"/m/Harry Potter/Harry Potter and the Goblet of Fire.m4v", ""},
		{"/m/Alfred Hitchcock/Blackmail.m4v", ""},
		{"/m/Pop Star/Popstar.m4v", ""},
		{"/Movies/Charade.mkv", ""},
	}

	for _, tt := range tests {
		if got := MovieFolderTitle(tt.path); got != tt.want {
			t.Errorf("MovieFolderTitle(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
