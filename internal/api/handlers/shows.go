package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stephencjuliano/media-server/internal/db"
)

type ShowsHandler struct {
	db *db.DB
}

func NewShowsHandler(database *db.DB) *ShowsHandler {
	return &ShowsHandler{
		db: database,
	}
}

// ShowDetail includes show info with seasons
type ShowDetail struct {
	*db.TVShow
	Seasons []*db.Season `json:"seasons"`
}

// GetShows returns all TV shows with counts
func (h *ShowsHandler) GetShows(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}

	shows, total, err := h.db.GetAllTVShows(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch TV shows"})
		return
	}

	c.JSON(http.StatusOK, PaginatedResponse{
		Items:  shows,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// GetShow returns a single TV show with its seasons
func (h *ShowsHandler) GetShow(c *gin.Context) {
	idStr := c.Param("showId")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid show ID"})
		return
	}

	show, err := h.db.GetTVShowByID(id)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Show not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch show"})
		return
	}

	seasons, err := h.db.GetSeasonsByShowID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch seasons"})
		return
	}

	c.JSON(http.StatusOK, ShowDetail{
		TVShow:  show,
		Seasons: seasons,
	})
}

// GetSeasons returns all seasons for a show
func (h *ShowsHandler) GetSeasons(c *gin.Context) {
	idStr := c.Param("showId")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid show ID"})
		return
	}

	seasons, err := h.db.GetSeasonsByShowID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch seasons"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"seasons": seasons})
}

// GetSeason returns a single season with episode count
func (h *ShowsHandler) GetSeason(c *gin.Context) {
	showIDStr := c.Param("showId")
	showID, err := strconv.ParseInt(showIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid show ID"})
		return
	}

	seasonNumStr := c.Param("seasonNum")
	seasonNum, err := strconv.Atoi(seasonNumStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid season number"})
		return
	}

	season, err := h.db.GetSeasonByNumber(showID, seasonNum)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Season not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch season"})
		return
	}

	c.JSON(http.StatusOK, season)
}

// GetEpisodes returns all episodes for a season
func (h *ShowsHandler) GetEpisodes(c *gin.Context) {
	showIDStr := c.Param("showId")
	showID, err := strconv.ParseInt(showIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid show ID"})
		return
	}

	seasonNumStr := c.Param("seasonNum")
	seasonNum, err := strconv.Atoi(seasonNumStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid season number"})
		return
	}

	// Get the season first to get its ID
	season, err := h.db.GetSeasonByNumber(showID, seasonNum)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Season not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch season"})
		return
	}

	episodes, err := h.db.GetEpisodesBySeasonID(season.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch episodes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"episodes": episodes})
}

// GetAllEpisodes returns all episodes for a show
func (h *ShowsHandler) GetAllEpisodes(c *gin.Context) {
	idStr := c.Param("showId")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid show ID"})
		return
	}

	episodes, err := h.db.GetEpisodesByShowID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch episodes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"episodes": episodes})
}

// GetEpisode returns a single episode by ID
func (h *ShowsHandler) GetEpisode(c *gin.Context) {
	idStr := c.Param("episodeId")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid episode ID"})
		return
	}

	episode, err := h.db.GetEpisodeByID(id)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Episode not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch episode"})
		return
	}

	c.JSON(http.StatusOK, episode)
}

// RandomEpisodeResponse includes show info with the random episode
type RandomEpisodeResponse struct {
	Episode   *db.Episode `json:"episode"`
	ShowTitle string      `json:"show_title"`
}

// GetRandomEpisode returns a random episode from the show
func (h *ShowsHandler) GetRandomEpisode(c *gin.Context) {
	idStr := c.Param("showId")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid show ID"})
		return
	}

	show, err := h.db.GetTVShowByID(id)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Show not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch show"})
		return
	}

	episode, err := h.db.GetRandomEpisode(id)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "No episodes found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get random episode"})
		return
	}

	c.JSON(http.StatusOK, RandomEpisodeResponse{
		Episode:   episode,
		ShowTitle: show.Title,
	})
}

// GetRandomEpisodeFromSeason returns a random episode from a specific season
func (h *ShowsHandler) GetRandomEpisodeFromSeason(c *gin.Context) {
	showIDStr := c.Param("showId")
	showID, err := strconv.ParseInt(showIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid show ID"})
		return
	}

	seasonNumStr := c.Param("seasonNum")
	seasonNum, err := strconv.Atoi(seasonNumStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid season number"})
		return
	}

	show, err := h.db.GetTVShowByID(showID)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Show not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch show"})
		return
	}

	season, err := h.db.GetSeasonByNumber(showID, seasonNum)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Season not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch season"})
		return
	}

	episode, err := h.db.GetRandomEpisodeFromSeason(season.ID)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "No episodes found in this season"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get random episode"})
		return
	}

	c.JSON(http.StatusOK, RandomEpisodeResponse{
		Episode:   episode,
		ShowTitle: show.Title,
	})
}
