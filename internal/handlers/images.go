package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/h2non/bimg"
	"github.com/petermazzocco/go-image-project/models"
	"gorm.io/gorm"
)

var publicURL = os.Getenv("PUBLIC_URL")

func UploadImageHandler(w http.ResponseWriter, r *http.Request, db *gorm.DB, client *s3.Client) {
	// Grab user ID from the middleware context
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	// Parse multipart form
	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create a unique image UUID and R2 key
	imageUUID := uuid.New()
	r2Key := fmt.Sprintf("images/%s/originals/%s_%s", userID, imageUUID.String(), header.Filename)

	// Upload image to R2
	obj, err := client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      aws.String(os.Getenv("BUCKET_NAME")),
		Key:         aws.String(r2Key),
		Body:        file,
		ContentType: aws.String(header.Header.Get("Content-Type")),
	})
	if err != nil {
		log.Println("Failed to upload image to R2:", err)
		http.Error(w, "Failed to upload image", http.StatusInternalServerError)
		return
	}
	log.Printf("Image uploaded to R2: %s, ETag: %s\n", r2Key, *obj.ETag)

	// Convert user ID to int
	userIDint, err := strconv.Atoi(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Create new image struct with returned metadata from R2
	// r2Key, filename, mimetype, etc.
	image := &models.Image{
		UUID:      imageUUID.String(),
		UserID:    uint(userIDint),
		Filename:  header.Filename,
		R2Key:     r2Key,
		MimeType:  header.Header.Get("Content-Type"),
		ImageData: nil,
	}

	// Save the returned image metadata to the database
	result := db.Create(image)
	if result.Error != nil {
		http.Error(w, "Error adding image to database", http.StatusInternalServerError)
		return
	}

	// Success response with message and url
	cleanURL := CleanURL(fmt.Sprintf(publicURL, r2Key))
	response := map[string]any{
		"message": "Image uploaded successfully",
		"url":     cleanURL,
	}
	json.NewEncoder(w).Encode(response)
}

func TransformImage(w http.ResponseWriter, r *http.Request, db *gorm.DB, client *s3.Client) {
	// Grab user ID from the middleware context
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")

	// JSON payload of image ID and options to transform
	var options bimg.Options
	if err := json.NewDecoder(r.Body).Decode(&options); err != nil {
		log.Println("ERROR DECODE: ", err)
		http.Error(w, "Invalid transformation options", http.StatusBadRequest)
		return
	}

	// Grab the image metadata from the DB that matches both the image ID and the user ID
	var image models.Image
	result := db.Where("id = ? AND user_id = ?", id, userID).First(&image)
	if result.Error != nil {
		log.Println("ERROR RESULT: ", result.Error)
		http.Error(w, "Image not found or user ID does not match", http.StatusUnauthorized)
		return
	}

	res, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(os.Getenv("BUCKET_NAME")),
		Key:    aws.String(image.R2Key),
	})
	if err != nil {
		log.Println("ERROR: ", err)
		http.Error(w, "Error getting object", http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		log.Println("ERROR: ", err)
		http.Error(w, "Error reading body data", http.StatusInternalServerError)
		return
	}

	newImage, err := bimg.NewImage(data).Process(options)
	if err != nil {
		log.Println(err)
		http.Error(w, "Error creating new image with options", http.StatusInternalServerError)
		return
	}

	// Generate a unique key for the transformed image
	transformedKey := fmt.Sprintf("images/%s/transformed/%s_%s_%d",
		userID, id, image.Filename, time.Now().Unix())

	// Upload the transformed image back to R2
	contentType := "image/jpeg"
	obj, err := client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(os.Getenv("BUCKET_NAME")),
		Key:         aws.String(transformedKey),
		Body:        bytes.NewReader(newImage),
		ContentType: aws.String(contentType),
	})
	log.Printf("Image uploaded to R2: %s, ETag: %s\n", transformedKey, *obj.ETag)
	if err != nil {
		log.Println(err)
		http.Error(w, "Error adding object to bucket", http.StatusInternalServerError)
		return
	}

	// Create new image record for the transformed image
	userIDUint, _ := strconv.Atoi(userID)
	transformedImage := models.Image{
		UserID:   uint(userIDUint),
		Filename: fmt.Sprintf("transformed_%s", image.Filename),
		R2Key:    transformedKey,
		UUID:     uuid.New().String(),
		MimeType: contentType,
	}

	// Save the transformed image metadata to database
	if err := db.Create(&transformedImage).Error; err != nil {
		log.Printf("Error saving transformed image metadata: %v", err)
		http.Error(w, "Error saving image metadata", http.StatusInternalServerError)
		return
	}

	// Success response with message and new url for transformed image
	cleanURL := CleanURL(fmt.Sprintf(publicURL, transformedKey))
	response := map[string]any{
		"message": "Image transformed successfully",
		"url":     cleanURL,
	}
	json.NewEncoder(w).Encode(response)
}

func CleanURL(urlStr string) string {
	urlStr = strings.ReplaceAll(urlStr, " ", "%20")
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}

	return parsedURL.String()
}

func GetImagesForUserHandler(w http.ResponseWriter, r *http.Request, db *gorm.DB, client *s3.Client) {
	// Grab user ID from the middleware context
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	// Grab all image metadata a user has
	var images []models.Image
	result := db.Where("user_id = ?", userID).Find(&images)
	if result.Error != nil {
		log.Println("ERROR RESULT: ", result.Error)
		http.Error(w, "Invalid user ID for images", http.StatusUnauthorized)
		return
	}

	var imageURLS []string
	for _, image := range images {
		cleanURL := CleanURL(fmt.Sprintf(publicURL, image.R2Key))
		imageURLS = append(imageURLS, cleanURL)
	}
	response := map[string]any{
		"message":    "Fetched images successfully",
		"image_urls": imageURLS,
	}
	json.NewEncoder(w).Encode(response)
}

func GetImageByIDHandler(w http.ResponseWriter, r *http.Request, db *gorm.DB, client *s3.Client) {
	// Grab user ID from the middleware context
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")

	// Grab the image metadata from the DB that matches both the image ID and the user ID
	var image models.Image
	result := db.Where("id = ? AND user_id = ?", id, userID).First(&image)
	if result.Error != nil {
		log.Println("ERROR RESULT: ", result.Error)
		http.Error(w, "Image not found or user ID does not match", http.StatusUnauthorized)
		return
	}

	cleanURL := CleanURL(fmt.Sprintf(publicURL, image.R2Key))
	response := map[string]any{
		"message": "Fetched image successfully",
		"url":     cleanURL,
	}
	json.NewEncoder(w).Encode(response)
}
