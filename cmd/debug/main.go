package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/osse101/BrandishBot_Go/internal/database"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default/environment variables")
	}

	// Construct connection string
	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	dbPool, err := database.NewPool(connString, 10, 30*time.Minute, time.Hour)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	ctx := context.Background()

	// Dump Platforms
	fmt.Println("--- Platforms ---")
	rows, err := dbPool.Query(ctx, "SELECT platform_id, platform_name FROM platforms")
	if err != nil {
		log.Printf("Failed to query platforms: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var id int
			var name string
			if err := rows.Scan(&id, &name); err != nil {
				log.Printf("Failed to scan platform: %v", err)
			}
			fmt.Printf("ID: %d, Name: %s\n", id, name)
		}
	}

	// Dump Users
	fmt.Println("\n--- Users ---")
	rows, err = dbPool.Query(ctx, "SELECT user_id, username, created_at FROM users")
	if err != nil {
		log.Printf("Failed to query users: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var id string
			var username string
			var createdAt interface{}
			if err := rows.Scan(&id, &username, &createdAt); err != nil {
				log.Printf("Failed to scan user: %v", err)
			}
			fmt.Printf("ID: %s, Username: %s, CreatedAt: %v\n", id, username, createdAt)
		}
	}

	// Dump Links
	fmt.Println("\n--- User Platform Links ---")
	query := `
		SELECT upl.user_platform_link_id, u.username, p.platform_name, upl.external_id
		FROM user_platform_links upl
		JOIN users u ON upl.user_id = u.user_id
		JOIN platforms p ON upl.platform_id = p.platform_id
	`
	rows, err = dbPool.Query(ctx, query)
	if err != nil {
		log.Printf("Failed to query links: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var id int
			var username, platform, externalID string
			if err := rows.Scan(&id, &username, &platform, &externalID); err != nil {
				log.Printf("Failed to scan link: %v", err)
			}
			fmt.Printf("LinkID: %d, User: %s, Platform: %s, ExternalID: %s\n", id, username, platform, externalID)
		}
	}

	// Dump Inventory
	fmt.Println("\n--- User Inventory ---")
	rows, err = dbPool.Query(ctx, "SELECT user_id, inventory_data FROM user_inventory")
	if err != nil {
		log.Printf("Failed to query inventory: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var userID string
			var data interface{}
			if err := rows.Scan(&userID, &data); err != nil {
				log.Printf("Failed to scan inventory: %v", err)
			}
			fmt.Printf("UserID: %s, Data: %v\n", userID, data)
		}
	}

	// Dump Items
	fmt.Println("\n--- Items ---")
	rows, err = dbPool.Query(ctx, "SELECT item_id, item_name FROM items")
	if err != nil {
		log.Printf("Failed to query items: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var id int
			var name string
			if err := rows.Scan(&id, &name); err != nil {
				log.Printf("Failed to scan item: %v", err)
			}
			fmt.Printf("ID: %d, Name: %s\n", id, name)
		}
	}

	// Dump Item Types
	fmt.Println("\n--- Item Types ---")
	rows, err = dbPool.Query(ctx, "SELECT item_type_id, type_name FROM item_types")
	if err != nil {
		log.Printf("Failed to query item types: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var id int
			var name string
			if err := rows.Scan(&id, &name); err != nil {
				log.Printf("Failed to scan item type: %v", err)
			}
			fmt.Printf("ID: %d, Name: %s\n", id, name)
		}
	}

	// Dump Item Assignments
	fmt.Println("\n--- Item Assignments ---")
	query = `
		SELECT i.item_name, t.type_name
		FROM item_type_assignments ita
		JOIN items i ON ita.item_id = i.item_id
		JOIN item_types t ON ita.item_type_id = t.item_type_id
	`
	rows, err = dbPool.Query(ctx, query)
	if err != nil {
		log.Printf("Failed to query assignments: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var itemName, typeName string
			if err := rows.Scan(&itemName, &typeName); err != nil {
				log.Printf("Failed to scan assignment: %v", err)
			}
			fmt.Printf("Item: %s, Type: %s\n", itemName, typeName)
		}
	}
}
