package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/markbates/goth/gothic"
	"github.com/petermazzocco/go-image-project/models"
	"gorm.io/gorm"
)

func UserLoginHandler(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	user, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		log.Println(w, err)
		return
	}

	var dbUser models.User
	if err := db.Where("email = ?", user.Email).First(&dbUser).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			dbUser = models.User{
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

	session, err := gothic.Store.Get(r, "_gothic_session")
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

	http.Redirect(w, r, "/upload", http.StatusTemporaryRedirect)
}

func GetUserHandler(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	var user models.User

	id, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	db.Where("id = ?", id).First(&user)

	result := db.Preload("Images").First(&user, id)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
