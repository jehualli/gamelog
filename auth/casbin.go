package auth

import (
	"log"

	"github.com/casbin/casbin/v2"
)

// LEARN — What is RBAC?
// RBAC (Role-Based Access Control) answers: "Can role X do action Y?"
// Instead of checking "is the user ID 42 allowed?", you check "does their role allow it?"
//
// Example:
//   player  → can access dashboard, create/edit their own games
//   admin   → can do everything a player can + access the admin panel
//
// Casbin separates the RULES from the ENFORCEMENT:
//   rbac/model.conf  → HOW to evaluate (the logic: request format, matching)
//   rbac/policy.csv  → WHAT is allowed (the data: role X can do action Y)
//
// This means you can change who can do what by editing policy.csv — no code changes.

var enforcer *casbin.Enforcer

// InitCasbin loads the model and policy files and creates the global enforcer.
func InitCasbin(modelPath, policyPath string) {
	var err error
	enforcer, err = casbin.NewEnforcer(modelPath, policyPath)
	if err != nil {
		log.Fatalf("casbin: failed to create enforcer: %v", err)
	}

	// Log active policies so you can see the rules at startup
	policies := enforcer.GetPolicy()
	log.Println("casbin: RBAC enforcer initialized with policies:")
	for _, p := range policies {
		log.Printf("  allow  subject=%-10s  action=%s", p[0], p[1])
	}
	roles := enforcer.GetGroupingPolicy()
	for _, g := range roles {
		log.Printf("  role   %s inherits %s", g[0], g[1])
	}
}

// GetEnforcer returns the global Casbin enforcer.
func GetEnforcer() *casbin.Enforcer {
	return enforcer
}

// Can checks if a role has permission to perform the given action.
// LEARN: This single function call encapsulates all RBAC logic.
// Casbin evaluates: does any policy row match (role, action)?
// Role inheritance is applied automatically — admin inherits player's permissions.
func Can(role, action string) bool {
	ok, err := enforcer.Enforce(role, action)
	if err != nil {
		log.Printf("casbin: enforce error for role=%s action=%s: %v", role, action, err)
		return false
	}
	return ok
}
