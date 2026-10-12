package library

import (
	"testing"

	"github.com/stephencjuliano/media-server/internal/db"
)

func TestPlaceFile(t *testing.T) {
	const tv, movies, all = "/media/TV Shows", "/media/Movies", "/media"

	tests := []struct {
		source, path string
		want         Placement
	}{
		// Episodes named every way a hand-organised library names them.
		{tv, tv + "/The Office/The Office/S01E01 - Pilot.m4v",
			Placement{Kind: PlaceEpisode, ShowTitle: "The Office", Season: 1, Episode: 1, EpisodeTitle: "Pilot"}},
		{tv, tv + "/Taskmaster/Series 04/01 A Fat Bald White Man (1080p HD).m4v",
			Placement{Kind: PlaceEpisode, ShowTitle: "Taskmaster", Season: 4, Episode: 1, EpisodeTitle: "A Fat Bald White Man"}},
		{tv, tv + "/Taskmaster/Series 19/Taskmaster - Series 19 Episode 03.mkv",
			Placement{Kind: PlaceEpisode, ShowTitle: "Taskmaster", Season: 19, Episode: 3}},
		{tv, tv + "/Breaking Bad/Breaking Bad/S0503 Madrigal (HD).mp4",
			Placement{Kind: PlaceEpisode, ShowTitle: "Breaking Bad", Season: 5, Episode: 3, EpisodeTitle: "Madrigal"}},
		{tv, tv + "/Wilfred AU/02 Dog Day Afterglow.mp4",
			Placement{Kind: PlaceEpisode, ShowTitle: "Wilfred AU", Season: 1, Episode: 2, EpisodeTitle: "Dog Day Afterglow"}},
		{tv, tv + "/BeetleJuice Cartoon/Animated BeetleJuice E03.m4v",
			Placement{Kind: PlaceEpisode, ShowTitle: "BeetleJuice Cartoon", Season: 1, Episode: 3, EpisodeTitle: "Animated BeetleJuice"}},
		{tv, tv + "/Doctor Who/Doctor Who/S02E00 - The Christmas Invasion.m4v",
			Placement{Kind: PlaceEpisode, ShowTitle: "Doctor Who", Season: 2, Episode: 0, EpisodeTitle: "The Christmas Invasion"}},
		{tv, tv + "/Some Show/Season 2/Some.Show.2x07.The.One.720p.mkv",
			Placement{Kind: PlaceEpisode, ShowTitle: "Some Show", Season: 2, Episode: 7, EpisodeTitle: "The One"}},

		// The show comes from the folder, never from the episode's own title.
		{tv, tv + "/The Venture Brothers/The Venture Brothers/S02E01 - Powerless in the Face of Death.m4v",
			Placement{Kind: PlaceEpisode, ShowTitle: "The Venture Brothers", Season: 2, Episode: 1, EpisodeTitle: "Powerless in the Face of Death"}},

		// Things in a show's folder that are not episodes.
		{tv, tv + "/Dragon Ball Z Kai/S01D02 Play All.mkv",
			Placement{Kind: PlaceShowBonus, ShowTitle: "Dragon Ball Z Kai", Season: 1, Title: "S01D02 Play All", Category: db.ExtraCategoryOther}},
		{tv, tv + "/The Office/The Office/S07D05 - Blooper Reel.m4v",
			Placement{Kind: PlaceShowBonus, ShowTitle: "The Office", Season: 1, Title: "S07D05 - Blooper Reel", Category: db.ExtraCategoryGagReel}},
		{tv, tv + "/Futurama/The Goonies_preview.m4v",
			Placement{Kind: PlaceShowBonus, ShowTitle: "Futurama", Season: 1, Title: "The Goonies preview", Category: db.ExtraCategoryOther}},
		{tv, tv + "/Breaking Bad/Breaking Bad/S05103 Season 5 Sneak Peak (HD).mp4",
			Placement{Kind: PlaceShowBonus, ShowTitle: "Breaking Bad", Season: 1, Title: "S05103 Season 5 Sneak Peak", Category: db.ExtraCategoryOther}},

		{tv, tv + "/Breaking Bad/Breaking Bad/S0502 Inside Breaking Bad_ Episode 501.mp4",
			Placement{Kind: PlaceShowBonus, ShowTitle: "Breaking Bad", Season: 5, Title: "Inside Breaking Bad Episode 501", Category: db.ExtraCategoryBehindTheScenes}},

		// A source that holds everything, with a TV folder at the top.
		{all, all + "/TV Shows/Wilfred AU/02 Dog Day Afterglow.mp4",
			Placement{Kind: PlaceEpisode, ShowTitle: "Wilfred AU", Season: 1, Episode: 2, EpisodeTitle: "Dog Day Afterglow"}},
		{all, all + "/Movies/Gremlins/Gremlins.m4v", Placement{Kind: PlaceFeature}},

		// Films and what is kept beside them.
		{movies, movies + "/Gremlins/Gremlins.m4v", Placement{Kind: PlaceFeature}},
		{movies, movies + "/01 Trolls (1080p HD).m4v", Placement{Kind: PlaceFeature}},
		{movies, movies + "/Gremlins/Gremlins Commentary 2.m4v",
			Placement{Kind: PlaceFilmBonus, Title: "Gremlins Commentary 2", Category: db.ExtraCategoryCommentary}},
		{movies, movies + "/Austin Powers/ComAustin Powers2.m4v",
			Placement{Kind: PlaceFilmBonus, Title: "ComAustin Powers2", Category: db.ExtraCategoryCommentary}},
		{movies, movies + "/Aladdin (1992)/Aladdin (1992) - Deleted.mkv",
			Placement{Kind: PlaceFilmBonus, Title: "Aladdin - Deleted", Category: db.ExtraCategoryDeletedScene}},

		// Recordings in a "shows" source with no show folders belong to no
		// show.
		{"/media/Stage Shows/Archive", "/media/Stage Shows/Archive/Grand Hotel.mp4",
			Placement{Kind: PlaceOther, Title: "Grand Hotel"}},
	}

	for _, tt := range tests {
		if got := PlaceFile(tt.source, tt.path); got != tt.want {
			t.Errorf("PlaceFile(%q)\n got  %+v\n want %+v", tt.path, got, tt.want)
		}
	}
}
