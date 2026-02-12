package main

import (
	"context"
	"fmt"
	"log"

	"github.com/mwork/mwork-api/internal/config"
	"github.com/mwork/mwork-api/internal/domain/user"
	"github.com/mwork/mwork-api/internal/pkg/database"
	"github.com/mwork/mwork-api/internal/pkg/password"
)

func main() {
	// Initialize config
	cfg := config.Load()

	// Connect to database
	db, err := database.NewPostgres(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.ClosePostgres(db)

	// List all users to debug
	query := `SELECT id, email, role, email_verified, is_banned FROM users`
	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("Failed to query users: %v", err)
	}
	defer rows.Close()

	// List tables to debug schema
	tableQuery := `SELECT table_name FROM information_schema.tables WHERE table_schema = 'public'`
	tRows, err := db.Query(tableQuery)
	if err != nil {
		log.Fatalf("Failed to query tables: %v", err)
	}
	defer tRows.Close()
	fmt.Println("--- Tables in DB ---")
	for tRows.Next() {
		var name string
		if err := tRows.Scan(&name); err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		fmt.Println(name)
	}
	fmt.Println("--------------------")

	// Inspect columns of 'profiles' table
	colQuery := `SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'profiles'`
	cRows, err := db.Query(colQuery)
	if err != nil {
		log.Printf("Failed to query profiles columns: %v", err)
	} else {
		defer cRows.Close()
		fmt.Println("--- Columns in 'profiles' ---")
		for cRows.Next() {
			var cName, cType string
			cRows.Scan(&cName, &cType)
			fmt.Printf("%s (%s)\n", cName, cType)
		}
		fmt.Println("-----------------------------")
	}

	// Inspect columns of 'users' table
	uColQuery := `SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'users'`
	uRows, err := db.Query(uColQuery)
	if err != nil {
		log.Printf("Failed to query users columns: %v", err)
	} else {
		defer uRows.Close()
		fmt.Println("--- Columns in 'users' ---")
		for uRows.Next() {
			var uName, uType string
			uRows.Scan(&uName, &uType)
			fmt.Printf("%s (%s)\n", uName, uType)
		}
		fmt.Println("--------------------------")
	}

	// Use an EXISTING user from the DB list above
	email := "model2@test.com"
	pwd := "password123"

	fmt.Println("--- Testing Login for", email, "---")
	repo := user.NewRepository(db)
	u, err := repo.GetByEmail(context.Background(), email)
	if err != nil {
		log.Printf("GetByEmail error for %s: %v", email, err)
		return
	}
	if u == nil {
		log.Printf("User %s not found (nil return)", email)
		return
	}

	fmt.Printf("User found: %+v\n", u.ID)
	fmt.Printf("Hash from DB: '%s'\n", u.PasswordHash)

	if u.PasswordHash == "" {
		log.Fatal("Password hash is empty!")
	}

	// Hash password
	generatedHash, err := password.Hash(pwd)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}
	fmt.Printf("Generated Hash for '%s': %s\n", pwd, generatedHash)

	match := password.Verify(pwd, u.PasswordHash)
	fmt.Printf("Password '%s' verification result: %v\n", pwd, match)

	if !match {
		fmt.Println("HASH MISMATCH! The hash in DB does not match 'password123'.")
		fmt.Println("Please update the seed script with the Generated Hash above.")
		// Don't fatal here, let it finish inspection
	}

	if u.IsBanned {
		log.Fatal("ERROR: User is BANNED")
	}

	if !u.EmailVerified {
		fmt.Println("WARNING: Email not verified (but usually allows login)")
	} else {
		fmt.Println("Email is verified.")
	}
	userCount := 0
	for rows.Next() {
		var id string
		var email string
		var role string
		var verified bool
		var banned bool
		if err := rows.Scan(&id, &email, &role, &verified, &banned); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		fmt.Printf("User: %s | %s | %s | Verified: %v | Banned: %v\n", id, email, role, verified, banned)
		userCount++
	}
	fmt.Printf("Total users found: %d\n", userCount)
	fmt.Println("----------------------------")

	// Try to get specific user
	repo = user.NewRepository(db)
	u, err = repo.GetByEmail(context.Background(), email)
	if err != nil {
		log.Printf("GetByEmail error for %s: %v", email, err)
		return
	}
	if u == nil {
		log.Printf("User %s not found (nil return)", email)
		return
	}

	fmt.Printf("User found: %+v\n", u.ID)
	fmt.Printf("Hash from DB: '%s'\n", u.PasswordHash)

	if u.PasswordHash == "" {
		fmt.Println("ERROR: Password hash is empty!")
		return
	}
	if u.IsBanned {
		fmt.Println("ERROR: User is banned!")
		return
	}
	if !u.EmailVerified {
		fmt.Println("WARNING: Email not verified (login should still be allowed)")
	}

	// Match password
	match = password.Verify(pwd, u.PasswordHash)
	fmt.Printf("Password '%s' verification result: %v\n", pwd, match)
}
