// @title           My API
// @version         1.0
// @description     This is a sample server.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@example.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1
package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/username/myproject/configs"
	"github.com/username/myproject/controllers"

	"github.com/username/myproject/repositories"
	"github.com/username/myproject/routes"
	"github.com/username/myproject/schedulers"
	"github.com/username/myproject/services"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, relying on system env vars")
	}
	log.Println("DEBUG GMAIL_USER:", os.Getenv("GMAIL_USER"))
	appName := "example-service"
	db := configs.SetupDatabase(&appName)
	log.Printf(">>> KAFKA_BROKER='%s'", os.Getenv("KAFKA_BROKER"))
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "dev" || appEnv == "local" || appEnv == "sit" {
		mode := "up"
		steps := 1
		if len(os.Args) >= 2 {
			mode = os.Args[1]
		}
		if len(os.Args) >= 3 {
			n, err := strconv.Atoi(os.Args[2])
			if err != nil {
				log.Fatalf("Invalid step number: %v", err)
			}
			steps = n
		}
		configs.RunMigrations(db, mode, steps)
	} else {
		log.Println("Skipping migrations: APP_ENV is not 'dev', 'local' or 'sit'")
	}

	// setup kafka
	broker := os.Getenv("KAFKA_BROKER")
	configs.CreateKafkaTopics(broker)
	kafkaProducer := configs.NewKafkaProducer(broker)
	defer kafkaProducer.Close()
	signingKey := []byte(os.Getenv("JWT_SIGNING_KEY"))
	issuer := os.Getenv("JWT_ISSUER")

	// setup repos, producers, services
	memberRepo := repositories.NewMemberRepository(db)
	memberProducer := configs.NewMemberProducer(kafkaProducer)
	tokenTTL := 60 * time.Minute
	memberService := services.NewMemberServiceWithAuth(memberRepo, memberProducer, db, signingKey, issuer, tokenTTL)
	mailService := services.NewMailService()
	authService := services.NewAuthService(signingKey, issuer)
	permService := services.NewPermissionService(db)

	// setup scheduler
	memberScheduler := schedulers.NewMemberScheduler(memberService, 5*time.Minute)
	go memberScheduler.Start()
	reportScheduler := schedulers.NewReportScheduler(memberService, *mailService, db, 16, 20)
	go reportScheduler.Start()

	// setup controller and routes
	memberController := controllers.NewMemberController(memberService, *mailService, authService)
	permissionController := controllers.NewPermissionController(authService, permService)
	r := gin.Default()

	// ✅ fix: health route registered on "r" (the actual engine), BEFORE Run()
	// ✅ fix: placed outside any auth-required group, at root level
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	routes.MemberRegisterRoutes(api, memberController)
	routes.PermissionRoutes(api, permissionController)
	// run server in goroutine so we can listen for shutdown signal
	go func() {
		if err := r.Run(":8080"); err != nil {
			log.Fatalf("❌ server error: %v", err)
		}
	}()

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🔴 shutting down...")
	memberScheduler.Stop()
	log.Println("✅ shutdown complete")
}
