package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
	"github.com/petermazzocco/go-image-project/internal/auth"
	"github.com/petermazzocco/go-image-project/internal/handlers"
	"github.com/petermazzocco/go-image-project/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Initialize environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file", err)
	}
	accountID := os.Getenv("ACCOUNT_ID")
	accessKeyID := os.Getenv("ACCESS_KEY_ID")
	accessKeySecret := os.Getenv("ACCESS_KEY_SECRET")

	// Chi
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// OAUTH
	goth.UseProviders(google.New(os.Getenv("GOOGLE_KEY"), os.Getenv("GOOGLE_SECRET"), "http://localhost:3000/auth/google/callback"))
	m := map[string]string{
		"google": "Google",
	}
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Session store
	secretKey := os.Getenv("JWT_SECRET_KEY")
	key := secretKey
	maxAge := 86400 * 30
	isProd := false
	store := sessions.NewCookieStore([]byte(key))
	store.MaxAge(maxAge)
	store.Options.Path = "/"
	store.Options.HttpOnly = true
	store.Options.Secure = isProd
	gothic.Store = store

	// Database connection
	dbURL := os.Getenv("DSN")
	dsn := dbURL
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto migrate models
	if err := db.AutoMigrate(models.User{}, models.Image{}); err != nil {
		log.Fatalf("Failed to auto migrate models: %v", err)
	}

	// Create custom HTTP client with TLS config
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			MaxVersion: tls.VersionTLS13,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
		},
	}
	httpClient := &http.Client{Transport: tr}

	// AWS S3 configuration
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithHTTPClient(httpClient),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, accessKeySecret, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		log.Fatal("ERR CONFIG:", err)
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID))
	})

	// User auth
	r.Get("/auth/{provider}/callback", func(w http.ResponseWriter, r *http.Request) {
		handlers.UserLoginHandler(w, r, db)
	})
	r.Post("/logout/{provider}", func(w http.ResponseWriter, r *http.Request) {
		gothic.Logout(w, r)
	})
	r.Post("/auth/{provider}", func(w http.ResponseWriter, r *http.Request) {
		if gothUser, err := gothic.CompleteUserAuth(w, r); err == nil {
			fmt.Fprintf(w, "User already authenticated: %s\n", gothUser.Name)
		} else {
			gothic.BeginAuthHandler(w, r)
		}
	})

	// Available API routes for authenticated users
	r.Route("/api", func(r chi.Router) {
		r.Use(auth.UserMiddleware)
		r.Use(httprate.Limit(
			20,
			1*time.Minute,
			httprate.WithKeyFuncs(httprate.KeyByIP, httprate.KeyByEndpoint),
		))
		r.Post("/upload", func(w http.ResponseWriter, r *http.Request) {
			handlers.UploadImageHandler(w, r, db, client)
		})
		r.Post("/transform", func(w http.ResponseWriter, r *http.Request) {
			handlers.TransformImage(w, r, db, client)
		})
		r.Get("/user", func(w http.ResponseWriter, r *http.Request) {
			handlers.GetUserHandler(w, r, db)
		})
	})

	log.Println("Starting API server on :3000")
	log.Fatal(http.ListenAndServe(":3000", r))
}
