# Image Processing Service

A Go-based image processing API service built for the [roadmap.sh Image Processing Service project](https://roadmap.sh/projects/image-processing-service).

## Features

- **Image Transformations**: Upload and apply various transformations to images using the bimg library
- **Google OAuth Authentication**: Secure user authentication via Google OAuth 2.0
- **Rate Limiting**: API endpoints protected with rate limiting (20 requests per minute)
- **Database Integration**: PostgreSQL database with GORM for user management
- **Session Management**: Secure session handling with cookie-based storage

## API Endpoints

### Authentication
- `POST /auth/google` - Initiate Google OAuth authentication
- `GET /auth/google/callback` - Handle OAuth callback
- `POST /logout/google` - Logout user

### Image Processing (Authenticated)
- `POST /api/transform` - Transform uploaded images with specified options

## Setup

1. Create a `.env` file with the following variables:
```
GOOGLE_KEY=your_google_oauth_client_id
GOOGLE_SECRET=your_google_oauth_client_secret
SECRET_KEY=your_session_secret_key
DSN=your_postgresql_connection_string
```

2. Install dependencies:
```bash
go mod tidy
```

3. Run the server:
```bash
go run cmd/api/main.go
```

The server will start on port 3000.

## Usage

1. Authenticate via Google OAuth at `/auth/google`
2. Upload images to `/api/transform` with transformation options
3. Receive processed images in response

## Dependencies

- Chi router for HTTP routing
- bimg for image processing
- Goth for OAuth authentication
- GORM for database operations
- PostgreSQL driver