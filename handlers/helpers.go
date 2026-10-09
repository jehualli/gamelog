package handlers

import (
	"html/template"
	"log"
	"net/http"
	"time"

	"gamelog/models"

	"github.com/gin-gonic/gin"
)

// render parses base.html + a page template together and executes "base".
// LEARN: Go templates use a composition model. base.html defines the layout with
// {{block "content" .}}{{end}} as a placeholder. Each page template overrides that
// block with {{define "content"}}...{{end}}. Together they produce a full page.
//
// We re-parse on every request so template changes appear without restarting the server.
// In production you'd cache the parsed templates for performance.
func render(c *gin.Context, status int, page string, data gin.H) {
	// Ensure the User field is always set from the JWT context (if logged in)
	if _, exists := data["User"]; !exists {
		data["User"] = currentUser(c)
	}

	t, err := template.New("").Funcs(funcMap()).ParseFiles(
		"templates/base.html",
		"templates/"+page,
	)
	if err != nil {
		log.Printf("template parse error (%s): %v", page, err)
		c.Data(http.StatusInternalServerError, "text/plain", []byte("Template error: "+err.Error()))
		return
	}

	c.Status(status)
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(c.Writer, "base", data); err != nil {
		log.Printf("template execute error (%s): %v", page, err)
	}
}

// funcMap returns template helper functions available in all HTML templates.
func funcMap() template.FuncMap {
	return template.FuncMap{
		"formatDate":  FormatDate,
		"statusColor": StatusColor,
		"add":         func(a, b int) int { return a + b },
		"seq": func(n int) []int {
			s := make([]int, n)
			for i := range s {
				s[i] = i + 1
			}
			return s
		},
	}
}

// currentUser extracts the logged-in user info from the Gin context.
// The JWT middleware sets these values; returns nil if not authenticated.
func currentUser(c *gin.Context) *models.User {
	id, exists := c.Get("userID")
	if !exists {
		return nil
	}
	return &models.User{
		ID:       id.(uint),
		Username: c.GetString("username"),
		Role:     c.GetString("userRole"),
	}
}

// FormatDate formats a time for display in templates.
func FormatDate(t time.Time) string {
	return t.Format("Jan 02, 2006 · 3:04 PM")
}

// StatusColor returns Tailwind CSS classes for a game status badge.
func StatusColor(status string) string {
	switch status {
	case "playing":
		return "bg-green-500/20 text-green-400 border border-green-500/30"
	case "completed":
		return "bg-blue-500/20 text-blue-400 border border-blue-500/30"
	case "paused":
		return "bg-yellow-500/20 text-yellow-400 border border-yellow-500/30"
	case "dropped":
		return "bg-red-500/20 text-red-400 border border-red-500/30"
	case "backlog":
		return "bg-gray-500/20 text-gray-400 border border-gray-500/30"
	default:
		return "bg-gray-500/20 text-gray-400 border border-gray-500/30"
	}
}
