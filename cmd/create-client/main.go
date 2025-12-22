package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/aouiniamine/aoui-drive/internal/config"
	"github.com/aouiniamine/aoui-drive/internal/database"
	"github.com/aouiniamine/aoui-drive/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	name := flag.String("name", "", "Client name (required)")
	role := flag.String("role", "USER", "Client role: ADMIN, MANAGER, or USER")
	flag.Parse()

	if *name == "" {
		fmt.Println("Usage: create-client -name <name> [-role <ADMIN|MANAGER|USER>]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *role != "ADMIN" && *role != "MANAGER" && *role != "USER" {
		log.Fatal("Role must be ADMIN, MANAGER, or USER")
	}

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := config.Load()

	db, err := database.New(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	accessKey := generateAccessKey()
	secretKey := generateSecretKey()

	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(secretKey), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash secret: %v", err)
	}

	client, err := db.Queries.CreateClient(context.Background(), sqlc.CreateClientParams{
		ID:        uuid.New().String(),
		Name:      *name,
		AccessKey: accessKey,
		SecretKey: string(hashedSecret),
		Role:      *role,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	fmt.Println("Client created successfully!")
	fmt.Println("----------------------------------------")
	fmt.Printf("ID:         %s\n", client.ID)
	fmt.Printf("Name:       %s\n", client.Name)
	fmt.Printf("Role:       %s\n", client.Role)
	fmt.Printf("Access Key: %s\n", client.AccessKey)
	fmt.Printf("Secret Key: %s\n", secretKey)
	fmt.Println("----------------------------------------")
	fmt.Println("Save the secret key now. It cannot be retrieved later.")
}

func generateAccessKey() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return "AK" + hex.EncodeToString(bytes)
}

func generateSecretKey() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
