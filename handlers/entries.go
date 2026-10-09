package handlers

import (
	"net/http"
	"strconv"

	"gamelog/database"
	"gamelog/models"
	"gamelog/services"

	"github.com/gin-gonic/gin"
)

// EntryNewPage renders the "add journal entry" form for a game.
func EntryNewPage(c *gin.Context) {
	game, ok := loadOwnedGame(c)
	if !ok {
		return
	}

	render(c, http.StatusOK, "entry_new.html", gin.H{
		"Title":             "New Entry — " + game.Title,
		"User":              currentUser(c),
		"Game":              game,
		"CloudinaryEnabled": services.CloudinaryEnabled(),
	})
}

// EntryCreate handles the journal entry form submission.
// A journal entry captures:
//   - Stage:      where in the game you are (level, boss, chapter, etc.)
//   - Notes:      observations, tips, questions you want to remember
//   - Impediment: what's blocking you or making it hard
//   - Emotion:    how this gaming session made you feel
//   - ImageURL:   an optional screenshot uploaded to Cloudinary
func EntryCreate(c *gin.Context) {
	game, ok := loadOwnedGame(c)
	if !ok {
		return
	}
	user := currentUser(c)

	entry := models.JournalEntry{
		GameID:     game.ID,
		UserID:     user.ID,
		Stage:      c.PostForm("stage"),
		Notes:      c.PostForm("notes"),
		Impediment: c.PostForm("impediment"),
		Emotion:    c.PostForm("emotion"),
	}

	// Handle optional screenshot upload to Cloudinary
	if file, _, err := c.Request.FormFile("screenshot"); err == nil {
		defer file.Close()
		if url, uploadErr := services.UploadImage(file, "screenshots"); uploadErr == nil {
			entry.ImageURL = url
		}
	}

	if err := database.DB.Create(&entry).Error; err != nil {
		render(c, http.StatusInternalServerError, "entry_new.html", gin.H{
			"Title": "New Entry",
			"User":  user,
			"Game":  game,
			"Error": "Failed to save entry. Please try again.",
		})
		return
	}

	// Update the game's UpdatedAt so it bubbles to the top of the dashboard list
	database.DB.Model(&game).Update("updated_at", entry.CreatedAt)

	gameID := strconv.Itoa(int(game.ID))
	c.Redirect(http.StatusFound, "/games/"+gameID+"?success=Entry+added")
}
