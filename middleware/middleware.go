// Package middleware provides Gin middleware for JWT authentication and RBAC authorization.
//
// LEARN — What is middleware?
// Middleware is code that runs between the HTTP request arriving and your handler running.
// Think of it as a security checkpoint at the door of each route.
//
// In Gin, the request lifecycle looks like:
//   HTTP Request
//     → Logger middleware (logs the request)
//     → JWTRequired middleware (checks the token)
//     → RBACRequired middleware (checks the role)
//     → Your handler (does the actual work)
//     → Response
//
// Each middleware calls c.Next() to pass control forward, or c.Abort() to stop the chain.
package middleware

import (
	"net/http"

	"gamelog/auth"

	"github.com/gin-gonic/gin"
)

// JWTRequired validates the JWT cookie on every request to protected routes.
// If the token is missing or invalid, the user is redirected to /login.
func JWTRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// LEARN: We store the JWT in an HTTP-only cookie (not localStorage).
		// HTTP-only means JavaScript cannot read it, which defends against XSS attacks.
		// The downside: it only works for browser clients, not mobile apps / APIs.
		tokenString, err := c.Cookie("jwt_token")
		if err != nil {
			// No cookie → not logged in
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			// Token expired or tampered → clear bad cookie, force re-login
			c.SetCookie("jwt_token", "", -1, "/", "", false, true)
			c.Redirect(http.StatusFound, "/login?error=session_expired")
			c.Abort()
			return
		}

		// LEARN: c.Set() stores arbitrary values in the request context.
		// These are available to every handler downstream via c.Get() / c.GetString() etc.
		// This is how we pass "who is this user?" from middleware to handlers.
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("userRole", claims.Role)

		c.Next() // hand off to the next middleware or handler
	}
}

// RBACRequired checks that the user's role is allowed to perform the given action.
// Call this AFTER JWTRequired so the role is already in the context.
//
// Usage in routes:
//
//	admin := r.Group("/admin")
//	admin.Use(middleware.JWTRequired(), middleware.RBACRequired("access_admin"))
//
// LEARN: By separating authentication (JWT) from authorization (RBAC), each concern
// is independent. You can change who can access what by editing policy.csv.
func RBACRequired(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("userRole")
		if role == "" {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// Ask Casbin: "Can role X perform action Y?"
		if !auth.Can(role, action) {
			// Role doesn't have this permission — redirect with an error flash message.
			// We redirect rather than rendering a 403 page to keep middleware simple.
			c.Redirect(http.StatusFound, "/dashboard?error=You+don%27t+have+permission+for+that")
			c.Abort()
			return
		}

		c.Next()
	}
}
