// GameLog — A cloud-native Go app for tracking your video game progress.
//
// Technologies used (and WHERE to find them in this project):
//
//   🔐 JWT (JSON Web Tokens)    → auth/jwt.go, middleware/middleware.go, handlers/auth.go
//   🛡  RBAC (Casbin)           → auth/casbin.go, middleware/middleware.go, rbac/
//   ☁️  Cloudinary              → services/cloudinary.go, handlers/games.go, handlers/entries.go
//   🗄  GORM + SQLite           → database/database.go, models/models.go
//   🌐 Gin (HTTP framework)     → main.go (router), handlers/
//   🎨 Tailwind CSS             → templates/ (via CDN, no build step)
//
// Start here, then follow the imports to understand how each piece fits together.
package main

import (
	"log"

	"gamelog/auth"
	"gamelog/config"
	"gamelog/database"
	"gamelog/handlers"
	"gamelog/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	// ── 1. Load config ────────────────────────────────────────────────────────
	// Reads .env file (if present) then environment variables.
	// All secrets and settings live in config.App after this call.
	config.Load()

	// ── 2. Connect to database ────────────────────────────────────────────────
	// Opens (or creates) the SQLite file and runs auto-migration to create tables.
	database.Connect(config.App.DBPath)

	// ── 3. Initialize RBAC ────────────────────────────────────────────────────
	// Loads rbac/model.conf (evaluation rules) and rbac/policy.csv (permissions).
	// After this, auth.Can("player", "access_admin") returns false, etc.
	auth.InitCasbin("rbac/model.conf", "rbac/policy.csv")

	// ── 4. Create Gin router ──────────────────────────────────────────────────
	// LEARN: gin.Default() adds two built-in middlewares automatically:
	//   • Logger   — prints every request: method, path, status, latency
	//   • Recovery — catches panics and returns 500 instead of crashing
	router := gin.Default()

	// Increase max upload size to 10MB for game covers and screenshots
	router.MaxMultipartMemory = 10 << 20

	// ── 5. Register routes ────────────────────────────────────────────────────
	registerRoutes(router)

	// ── 6. Start server ───────────────────────────────────────────────────────
	addr := ":" + config.App.Port
	log.Printf("")
	log.Printf("  ╔══════════════════════════════════════════╗")
	log.Printf("  ║  🎮 GameLog is running!                  ║")
	log.Printf("  ║                                          ║")
	log.Printf("  ║  Open: http://localhost%s              ║", addr)
	log.Printf("  ║                                          ║")
	log.Printf("  ║  JWT     → auth/jwt.go                   ║")
	log.Printf("  ║  RBAC    → rbac/ + auth/casbin.go        ║")
	log.Printf("  ║  Cloudinary → services/cloudinary.go     ║")
	log.Printf("  ╚══════════════════════════════════════════╝")
	log.Printf("")

	if err := router.Run(addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func registerRoutes(r *gin.Engine) {
	// ── Public routes (no login required) ────────────────────────────────────
	r.GET("/", handlers.Index)
	r.GET("/login", handlers.LoginPage)
	r.POST("/login", handlers.Login)
	r.GET("/register", handlers.RegisterPage)
	r.POST("/register", handlers.Register)
	r.POST("/logout", handlers.Logout)

	// ── Protected routes (JWT cookie required) ────────────────────────────────
	// LEARN: r.Group() creates a sub-router that shares a path prefix and middlewares.
	// Every route in this group runs middleware.JWTRequired() before the handler.
	// If the token is missing/invalid, JWTRequired redirects to /login — the handler never runs.
	protected := r.Group("/")
	protected.Use(middleware.JWTRequired())
	{
		protected.GET("/dashboard", handlers.Dashboard)

		// Game CRUD
		protected.GET("/games/new", handlers.GameNewPage)
		protected.POST("/games", handlers.GameCreate)
		protected.GET("/games/:id", handlers.GameDetail)
		protected.GET("/games/:id/edit", handlers.GameEditPage)
		protected.POST("/games/:id/update", handlers.GameUpdate)
		protected.POST("/games/:id/delete", handlers.GameDelete)

		// Journal entries
		protected.GET("/games/:id/entries/new", handlers.EntryNewPage)
		protected.POST("/games/:id/entries", handlers.EntryCreate)
	}

	// ── Admin-only routes (JWT + RBAC check) ──────────────────────────────────
	// LEARN: This group uses TWO middleware in sequence:
	//   1. JWTRequired  → "are you logged in?"
	//   2. RBACRequired → "does your role have the 'access_admin' permission?"
	// Casbin evaluates: does policy.csv have a row matching (user.Role, "access_admin")?
	// For role=player: No → blocked.  For role=admin: Yes (via inheritance) → allowed.
	admin := r.Group("/admin")
	admin.Use(middleware.JWTRequired(), middleware.RBACRequired("access_admin"))
	{
		admin.GET("", handlers.AdminPanel)
		admin.GET("/users", handlers.AdminUsers)
		admin.POST("/users/:id/delete", handlers.AdminDeleteUser)
	}
}
