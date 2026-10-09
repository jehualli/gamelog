package handlers

import (
	"net/http"
	"strings"

	"gamelog/auth"
	"gamelog/database"
	"gamelog/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Index renders the landing page, or redirects to dashboard if already logged in.
func Index(c *gin.Context) {
	if token, err := c.Cookie("jwt_token"); err == nil {
		if _, err := auth.ValidateToken(token); err == nil {
			c.Redirect(http.StatusFound, "/dashboard")
			return
		}
	}
	render(c, http.StatusOK, "index.html", gin.H{
		"Title": "Track Your Gaming Journey",
	})
}

// LoginPage shows the login form.
func LoginPage(c *gin.Context) {
	// If already logged in, go to dashboard
	if _, err := c.Cookie("jwt_token"); err == nil {
		c.Redirect(http.StatusFound, "/dashboard")
		return
	}
	render(c, http.StatusOK, "login.html", gin.H{
		"Title": "Login",
		"Error": c.Query("error"),
	})
}

// Login handles the login form submission.
// LEARN: The login flow is:
//  1. Look up user by username
//  2. Compare submitted password against stored bcrypt hash
//  3. If valid, generate a JWT token
//  4. Store the token in an HTTP-only cookie
//  5. Redirect to dashboard
func Login(c *gin.Context) {
	username := strings.TrimSpace(c.PostForm("username"))
	password := c.PostForm("password")

	if username == "" || password == "" {
		render(c, http.StatusUnprocessableEntity, "login.html", gin.H{
			"Title": "Login",
			"Error": "Username and password are required.",
		})
		return
	}

	// Find user by username
	var user models.User
	if err := database.DB.Where("username = ?", username).First(&user).Error; err != nil {
		render(c, http.StatusUnauthorized, "login.html", gin.H{
			"Title": "Login",
			"Error": "Invalid username or password.",
		})
		return
	}

	// LEARN: bcrypt.CompareHashAndPassword is timing-safe — it takes the same amount
	// of time whether the password is right or wrong, preventing timing attacks.
	// We store hashes, never plain-text passwords.
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		render(c, http.StatusUnauthorized, "login.html", gin.H{
			"Title": "Login",
			"Error": "Invalid username or password.",
		})
		return
	}

	// Generate JWT token
	token, err := auth.GenerateToken(&user)
	if err != nil {
		render(c, http.StatusInternalServerError, "login.html", gin.H{
			"Title": "Login",
			"Error": "Could not create session. Please try again.",
		})
		return
	}

	// LEARN: SetCookie parameters: name, value, maxAge(seconds), path, domain, secure, httpOnly
	// httpOnly=true  → JavaScript can't access this cookie (XSS protection)
	// secure=false   → Allow HTTP for local dev; set true in production (HTTPS only)
	// maxAge=86400   → Cookie expires in 24 hours (same as token lifetime)
	c.SetCookie("jwt_token", token, 86400, "/", "", false, true)
	c.Redirect(http.StatusFound, "/dashboard")
}

// RegisterPage shows the registration form.
func RegisterPage(c *gin.Context) {
	if _, err := c.Cookie("jwt_token"); err == nil {
		c.Redirect(http.StatusFound, "/dashboard")
		return
	}
	render(c, http.StatusOK, "register.html", gin.H{
		"Title": "Create Account",
	})
}

// Register handles account creation.
// LEARN: bcrypt.GenerateFromPassword adds a random salt automatically.
// Even if two users have the same password, their hashes will be different.
// The cost factor (12) controls how slow the hash is — slow = harder to brute-force.
func Register(c *gin.Context) {
	username := strings.TrimSpace(c.PostForm("username"))
	email := strings.TrimSpace(c.PostForm("email"))
	password := c.PostForm("password")
	confirm := c.PostForm("confirm_password")

	// Validate inputs
	if username == "" || email == "" || password == "" {
		render(c, http.StatusUnprocessableEntity, "register.html", gin.H{
			"Title": "Create Account",
			"Error": "All fields are required.",
		})
		return
	}
	if len(password) < 6 {
		render(c, http.StatusUnprocessableEntity, "register.html", gin.H{
			"Title": "Create Account",
			"Error": "Password must be at least 6 characters.",
		})
		return
	}
	if password != confirm {
		render(c, http.StatusUnprocessableEntity, "register.html", gin.H{
			"Title": "Create Account",
			"Error": "Passwords do not match.",
		})
		return
	}

	// Hash password with bcrypt (cost 12 is the recommended minimum for 2024)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		render(c, http.StatusInternalServerError, "register.html", gin.H{
			"Title": "Create Account",
			"Error": "Failed to secure password. Please try again.",
		})
		return
	}

	user := models.User{
		Username: username,
		Email:    email,
		Password: string(hashedPassword),
		Role:     "player", // all new users start as player
	}

	// LEARN: GORM returns a specific error type for unique constraint violations.
	// We check the error message because different databases use different error codes.
	if err := database.DB.Create(&user).Error; err != nil {
		msg := "Registration failed. Please try again."
		if strings.Contains(err.Error(), "UNIQUE") {
			msg = "Username or email is already taken."
		}
		render(c, http.StatusConflict, "register.html", gin.H{
			"Title": "Create Account",
			"Error": msg,
		})
		return
	}

	// Auto-login after registration
	token, _ := auth.GenerateToken(&user)
	c.SetCookie("jwt_token", token, 86400, "/", "", false, true)
	c.Redirect(http.StatusFound, "/dashboard")
}

// Logout clears the JWT cookie and redirects home.
func Logout(c *gin.Context) {
	// Set cookie with maxAge=-1 to delete it
	c.SetCookie("jwt_token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/")
}
