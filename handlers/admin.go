package handlers

import (
	"net/http"
	"strconv"

	"gamelog/database"
	"gamelog/models"

	"github.com/gin-gonic/gin"
)

// AdminPanel shows an overview for admin users.
// LEARN: This route is protected by TWO middleware layers:
//  1. JWTRequired — verifies the user is logged in
//  2. RBACRequired("access_admin") — verifies the user's role allows admin access
//
// If a player navigates to /admin, Casbin returns false for ("player", "access_admin")
// and the middleware redirects them before this handler ever runs.
func AdminPanel(c *gin.Context) {
	var userCount, gameCount, entryCount int64
	database.DB.Model(&models.User{}).Count(&userCount)
	database.DB.Model(&models.Game{}).Count(&gameCount)
	database.DB.Model(&models.JournalEntry{}).Count(&entryCount)

	render(c, http.StatusOK, "admin.html", gin.H{
		"Title":      "Admin Panel",
		"User":       currentUser(c),
		"UserCount":  userCount,
		"GameCount":  gameCount,
		"EntryCount": entryCount,
	})
}

// AdminUsers lists all registered users with their game counts.
func AdminUsers(c *gin.Context) {
	// LEARN: db.Preload("Games") runs a second SELECT to load each user's games.
	// This is called "eager loading" — we load related records upfront to avoid
	// the N+1 problem (making one extra DB query per user in a loop).
	var users []models.User
	database.DB.Preload("Games").Order("created_at DESC").Find(&users)

	render(c, http.StatusOK, "admin_users.html", gin.H{
		"Title": "All Users",
		"User":  currentUser(c),
		"Users": users,
	})
}

// AdminDeleteUser deletes a user and all their games + entries.
// LEARN: This cascades manually because SQLite doesn't enforce foreign keys by default.
// In Postgres you'd use ON DELETE CASCADE in the schema instead.
func AdminDeleteUser(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/users")
		return
	}

	// Prevent admin from deleting themselves
	if currentUser(c).ID == uint(userID) {
		c.Redirect(http.StatusFound, "/admin/users?error=Cannot+delete+yourself")
		return
	}

	// Cascade delete: entries → games → user
	var gameIDs []uint
	database.DB.Model(&models.Game{}).Where("user_id = ?", userID).Pluck("id", &gameIDs)
	if len(gameIDs) > 0 {
		database.DB.Where("game_id IN ?", gameIDs).Delete(&models.JournalEntry{})
		database.DB.Where("id IN ?", gameIDs).Delete(&models.Game{})
	}
	database.DB.Delete(&models.User{}, userID)

	c.Redirect(http.StatusFound, "/admin/users?success=User+deleted")
}
