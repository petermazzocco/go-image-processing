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

## Transformation JSON
Pass in a JSON object into the body with transformation options on the `/api/transform` POST call, like so:
```
{
  "height": 800,
  "width": 600,
  "areaHeight": 400,
  "areaWidth": 300,
  "top": 50,
  "left": 100,
  "quality": 85,
  "compression": 6,
  "zoom": 1,
  "crop": true,
  "smartCrop": false,
  "enlarge": false,
  "embed": false,
  "flip": false,
  "flop": false,
  "force": false,
  "noAutoRotate": false,
  "noProfile": false,
  "interlace": true,
  "stripMetadata": true,
  "trim": false,
  "lossless": false,
  "extend": {
    "top": 10,
    "bottom": 10,
    "left": 5,
    "right": 5,
    "background": "#ffffff"
  },
  "rotate": 90,
  "background": {
    "r": 255,
    "g": 255,
    "b": 255
  },
  "gravity": "center",
  "watermark": {
    "text": "© Your Company",
    "opacity": 0.5,
    "width": 200,
    "dpi": 150,
    "margin": 20,
    "font": "Arial",
    "background": {
      "r": 0,
      "g": 0,
      "b": 0
    }
  },
  "watermarkImage": {
    "left": 10,
    "top": 10,
    "buf": "base64_encoded_image_data_here"
  },
  "type": "jpeg",
  "interpolator": "bicubic",
  "interpretation": "srgb",
  "gaussianBlur": {
    "sigma": 1.5,
    "minAmpl": 0.2
  },
  "sharpen": {
    "radius": 1,
    "x1": 2,
    "y2": 10,
    "y3": 20,
    "m1": 1,
    "m2": 2
  },
  "threshold": 0.5,
  "gamma": 2.2,
  "brightness": 1.0,
  "contrast": 1.2,
  "outputICC": "sRGB",
  "inputICC": "Adobe RGB",
  "palette": false,
  "speed": 4
}
```

## Dependencies

- Chi router for HTTP routing
- bimg for image processing
- Goth for OAuth authentication
- GORM for database operations
- PostgreSQL driver
