package handlers

import (
	"net/http"
	"strconv"

	"gamelog/auth"
	"gamelog/database"
	"gamelog/models"
	"gamelog/services"

	"github.com/gin-gonic/gin"
)

// Dashboard lists all games for the logged-in user.
// LEARN: We use GORM's Where().Find() to fetch only the current user's games.
// The userID comes from the JWT middleware — the handler never touches the token directly.
func Dashboard(c *gin.Context) {
	user := currentUser(c)

	// Check RBAC — does this role have dashboard access?
	// LEARN: auth.Can() is an explicit check you can call anywhere; middleware.RBACRequired()
	// does the same check but aborts before reaching the handler.
	if !auth.Can(user.Role, "access_dashboard") {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	var games []models.Game
	database.DB.Where("user_id = ?", user.ID).Order("updated_at DESC").Find(&games)

	render(c, http.StatusOK, "dashboard.html", gin.H{
		"Title":   "My Games",
		"User":    user,
		"Games":   games,
		"Error":   c.Query("error"),
		"Success": c.Query("success"),
	})
}

// GameNewPage renders the "add game" form.
func GameNewPage(c *gin.Context) {
	render(c, http.StatusOK, "game_new.html", gin.H{
		"Title":              "Add Game",
		"User":               currentUser(c),
		"CloudinaryEnabled":  services.CloudinaryEnabled(),
	})
}

// GameCreate handles the "add game" form submission.
// LEARN: For file uploads, Gin provides c.FormFile() which returns a *multipart.FileHeader.
// We open the file, pass it to the Cloudinary service, and store the returned URL.
func GameCreate(c *gin.Context) {
	user := currentUser(c)

	title := c.PostForm("title")
	if title == "" {
		render(c, http.StatusUnprocessableEntity, "game_new.html", gin.H{
			"Title":             "Add Game",
			"User":              user,
			"Error":             "Title is required.",
			"CloudinaryEnabled": services.CloudinaryEnabled(),
		})
		return
	}

	rating, _ := strconv.Atoi(c.PostForm("rating"))

	game := models.Game{
		UserID:   user.ID,
		Title:    title,
		Platform: c.PostForm("platform"),
		Genre:    c.PostForm("genre"),
		Status:   orDefault(c.PostForm("status"), "playing"),
		Rating:   rating,
		Notes:    c.PostForm("notes"),
	}

	// Handle optional cover image upload to Cloudinary
	if file, header, err := c.Request.FormFile("cover_image"); err == nil {
		_ = header // we don't use the filename
		defer file.Close()

		// LEARN: services.UploadImage() calls the Cloudinary API.
		// If it fails (no credentials, network error), we continue without an image.
		if url, uploadErr := services.UploadImage(file, "covers"); uploadErr == nil {
			game.CoverURL = url
		}
	}

	if err := database.DB.Create(&game).Error; err != nil {
		render(c, http.StatusInternalServerError, "game_new.html", gin.H{
			"Title": "Add Game",
			"User":  user,
			"Error": "Failed to save game. Please try again.",
		})
		return
	}

	c.Redirect(http.StatusFound, "/games/"+strconv.Itoa(int(game.ID)))
}

// GameDetail shows a game and all its journal entries.
// LEARN: Preload("Entries") tells GORM to run a second SELECT to load all entries
// for this game and attach them to game.Entries. Without Preload, game.Entries would be nil.
func GameDetail(c *gin.Context) {
	game, ok := loadOwnedGame(c)
	if !ok {
		return
	}

	// Load journal entries, newest first
	database.DB.Where("game_id = ?", game.ID).Order("created_at DESC").Find(&game.Entries)

	render(c, http.StatusOK, "game_detail.html", gin.H{
		"Title":   game.Title,
		"User":    currentUser(c),
		"Game":    game,
		"Success": c.Query("success"),
	})
}

// GameEditPage renders the edit form pre-filled with existing game data.
func GameEditPage(c *gin.Context) {
	game, ok := loadOwnedGame(c)
	if !ok {
		return
	}

	render(c, http.StatusOK, "game_edit.html", gin.H{
		"Title":             "Edit: " + game.Title,
		"User":              currentUser(c),
		"Game":              game,
		"CloudinaryEnabled": services.CloudinaryEnabled(),
	})
}

// GameUpdate handles the edit form submission.
func GameUpdate(c *gin.Context) {
	game, ok := loadOwnedGame(c)
	if !ok {
		return
	}

	title := c.PostForm("title")
	if title == "" {
		render(c, http.StatusUnprocessableEntity, "game_edit.html", gin.H{
			"Title": "Edit Game",
			"User":  currentUser(c),
			"Game":  game,
			"Error": "Title is required.",
		})
		return
	}

	rating, _ := strconv.Atoi(c.PostForm("rating"))
	game.Title = title
	game.Platform = c.PostForm("platform")
	game.Genre = c.PostForm("genre")
	game.Status = orDefault(c.PostForm("status"), "playing")
	game.Rating = rating
	game.Notes = c.PostForm("notes")

	// Update cover image if a new one was uploaded
	if file, _, err := c.Request.FormFile("cover_image"); err == nil {
		defer file.Close()
		if url, uploadErr := services.UploadImage(file, "covers"); uploadErr == nil {
			game.CoverURL = url
		}
	}

	// LEARN: db.Save() issues an UPDATE for all fields of the record.
	// db.Updates() (with a map) only updates specified fields — useful to avoid overwriting fields
	// you didn't mean to touch.
	database.DB.Save(&game)
	c.Redirect(http.StatusFound, "/games/"+strconv.Itoa(int(game.ID))+"?success=Game+updated")
}

// GameDelete deletes a game and all its entries.
func GameDelete(c *gin.Context) {
	game, ok := loadOwnedGame(c)
	if !ok {
		return
	}

	// Delete entries first to avoid orphaned rows (foreign key constraint)
	database.DB.Where("game_id = ?", game.ID).Delete(&models.JournalEntry{})
	// LEARN: db.Delete() performs a soft delete by setting deleted_at, not removing the row.
	// The row stays in the database but GORM automatically excludes it from future queries.
	database.DB.Delete(&game)

	c.Redirect(http.StatusFound, "/dashboard?success=Game+deleted")
}

// loadOwnedGame is a helper that fetches a game by ID and verifies the current user owns it.
// LEARN: This is "resource ownership" authorization — different from RBAC (which checks roles).
// Even if two users are both "players", they should only see their own data.
// We check user_id = current_user_id at the database query level, not in application code.
func loadOwnedGame(c *gin.Context) (*models.Game, bool) {
	user := currentUser(c)
	gameID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Redirect(http.StatusFound, "/dashboard")
		return nil, false
	}

	var game models.Game
	// The WHERE clause enforces ownership: even if someone guesses another game's ID,
	// this query will find no rows because user_id won't match.
	result := database.DB.Where("id = ? AND user_id = ?", gameID, user.ID).First(&game)
	if result.Error != nil {
		c.Redirect(http.StatusFound, "/dashboard?error=Game+not+found")
		return nil, false
	}

	return &game, true
}

func orDefault(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
