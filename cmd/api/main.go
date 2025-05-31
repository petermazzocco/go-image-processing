package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/gorilla/sessions"
	"github.com/h2non/bimg"
	"github.com/joho/godotenv"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
	"github.com/petermazzocco/go-image-project/internal/auth"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Name      string         `gorm:"size:255;not null"`
	Email     string         `gorm:"size:255;not null;unique"`
}

func main() {
	// Initialize environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

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
	secretKey := os.Getenv("SECRET_KEY")
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
	if err := db.AutoMigrate(User{}); err != nil {
		log.Fatalf("Failed to auto migrate models: %v", err)
	}

	// User auth
	r.Get("/auth/{provider}/callback", func(w http.ResponseWriter, r *http.Request) {
		user, err := gothic.CompleteUserAuth(w, r)
		if err != nil {
			log.Println(w, err)
			return
		}

		var dbUser User
		if err := db.Where("email = ?", user.Email).First(&dbUser).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				dbUser = User{
					Name:  user.Name,
					Email: user.Email,
				}
				if err := db.Create(&dbUser).Error; err != nil {
					log.Println("Failed to create user:", err)
					http.Error(w, "Failed to create user", http.StatusInternalServerError)
					return
				}
			} else {
				log.Println("Database error:", err)
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
		}

		session, err := gothic.Store.Get(r, "gothic_session")
		if err != nil {
			log.Println("Failed to get session:", err)
			http.Error(w, "Failed to get session", http.StatusInternalServerError)
			return
		}
		session.Values["user_id"] = dbUser.ID

		if err := session.Save(r, w); err != nil {
			log.Println("Failed to save session:", err)
			http.Error(w, "Failed to save session", http.StatusInternalServerError)
			return
		}
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
		r.Post("/transform", func(w http.ResponseWriter, r *http.Request) {
			var o bimg.Options
			if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
				http.Error(w, "Invalid transformation options", http.StatusBadRequest)
				return
			}

			err := r.ParseMultipartForm(10 << 20)
			if err != nil {
				log.Println(err)
				http.Error(w, "Error parsing form", http.StatusBadRequest)
				return
			}

			file, header, err := r.FormFile("image")
			if err != nil {
				log.Println(err)
				http.Error(w, "Error getting file", http.StatusBadRequest)
				return
			}
			defer file.Close()

			fileData, err := io.ReadAll(file)
			if err != nil {
				log.Println(err)
				http.Error(w, "Error reading file data", http.StatusBadRequest)
				return
			}

			processedImage, err := bimg.NewImage(fileData).Process(o)
			bimg.Write(header.Filename, processedImage)
			w.WriteHeader(http.StatusOK)
			w.Write(processedImage)
		})
	})

	log.Println("Starting API server on :3000")
	log.Fatal(http.ListenAndServe(":3000", r))
}
