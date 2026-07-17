package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/username/myproject/services"
)

func main() {
	userID := flag.String("user", "user-123", "user ID to embed in the token")
	role := flag.String("role", "admin", "role to embed in the token")
	ttl := flag.Duration("ttl", time.Hour, "token validity duration, e.g. 1h, 30m")
	flag.Parse()

	signingKey := []byte(os.Getenv("JWT_SIGNING_KEY"))
	issuer := os.Getenv("JWT_ISSUER")

	if len(signingKey) == 0 || issuer == "" {
		log.Fatal("JWT_SIGNING_KEY and JWT_ISSUER must be set in your environment")
	}

	token, err := services.NewToken(signingKey, issuer, *userID, *role, *ttl)
	if err != nil {
		log.Fatal("failed to mint token: ", err)
	}

	fmt.Println(token)
}
