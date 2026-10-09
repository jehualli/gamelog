# GameLog — Video Game Progress Tracker

A cloud-native Go app for learning **JWT auth**, **RBAC (Casbin)**, and **Cloudinary** by building something you'll actually use.

## Technologies & Where to Find Them

| Technology | Purpose | Key Files |
|-----------|---------|-----------|
| **Gin** | HTTP framework, routing, middleware | `main.go`, `handlers/` |
| **GORM + SQLite** | ORM, database, auto-migration | `database/`, `models/` |
| **JWT** | Stateless authentication | `auth/jwt.go`, `middleware/middleware.go` |
| **Casbin (RBAC)** | Role-based access control | `auth/casbin.go`, `rbac/` |
| **Cloudinary** | Image/video hosting on CDN | `services/cloudinary.go` |
| **Tailwind CSS** | Styling via CDN (no build step) | `templates/` |

---

## Quick Start

### 1. Prerequisites
- Go 1.22+ → [go.dev/dl](https://go.dev/dl)
- (Optional) Cloudinary free account → [cloudinary.com](https://cloudinary.com)

### 2. Set up environment
```bash
cd gamelog
cp .env.example .env
# Edit .env — at minimum, change JWT_SECRET.
# Add Cloudinary credentials if you want image upload.
```

### 3. Install dependencies
```bash
go mod tidy
```

### 4. Run
```bash
go run .
```

Open **http://localhost:8080**

---

## Learning Walkthrough

### JWT — How login works
1. Register at `/register`
2. Look at `auth/jwt.go` — see how the token is created with `GenerateToken()`
3. Open browser DevTools → Application → Cookies → find `jwt_token`
4. Paste the value at [jwt.io](https://jwt.io) to decode the payload (UserID, username, role)
5. Notice: the payload is readable, but the signature can't be faked without JWT_SECRET

### RBAC — Two roles in action
1. Create two accounts. In `gamelog.db`, set one user's `role` to `admin`
2. Log in as admin → you see ⚙ Admin in the nav
3. Log in as player → `/admin` redirects you away
4. Read `rbac/policy.csv` — this is the only place roles are defined
5. Add a new permission: add a line, restart, test

### Cloudinary — Image upload flow
1. Add Cloudinary credentials to `.env`
2. Add a game with a cover image
3. In your Cloudinary dashboard, see the uploaded file under `gamelog/covers/`
4. The URL in the DB ends with `.webp` — Cloudinary auto-converted it

### GORM — Database operations
1. After running the app, open `gamelog.db` with [DB Browser for SQLite](https://sqlitebrowser.org/)
2. See the `users`, `games`, `journal_entries` tables auto-created
3. Look at `models/models.go` — each struct field with a `gorm:` tag maps to a column

---

## Project Structure

```
gamelog/
├── main.go                   ← entry point, routes
├── config/config.go          ← env var loading
├── database/database.go      ← GORM connection + migration
├── models/models.go          ← User, Game, JournalEntry structs
├── auth/
│   ├── jwt.go                ← token generation + validation
│   └── casbin.go             ← RBAC enforcer setup
├── middleware/middleware.go   ← JWTRequired, RBACRequired
├── services/cloudinary.go    ← image upload to Cloudinary
├── handlers/
│   ├── helpers.go            ← template renderer, currentUser()
│   ├── auth.go               ← login, register, logout
│   ├── games.go              ← game CRUD
│   ├── entries.go            ← journal entry creation
│   └── admin.go              ← admin panel
├── templates/                ← HTML templates (base.html + pages)
└── rbac/
    ├── model.conf            ← Casbin model (evaluation logic)
    └── policy.csv            ← Casbin policy (who can do what)
```

---

## Making yourself an Admin

```bash
# Install sqlite3 CLI (macOS: brew install sqlite)
sqlite3 gamelog.db "UPDATE users SET role='admin' WHERE username='your_username';"
```

Then log out and back in — the new JWT will carry the `admin` role.
