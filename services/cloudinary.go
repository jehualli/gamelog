// Package services contains third-party integrations.
//
// LEARN — What is Cloudinary?
// Cloudinary is a cloud service for storing and serving images/videos.
// Instead of storing images on your server's disk (which doesn't scale),
// you upload them to Cloudinary and get back a URL you store in the database.
//
// Benefits:
//   • CDN delivery — images served from servers near the user (fast!)
//   • On-the-fly transforms — resize/crop/convert by changing the URL
//     e.g. .../upload/w_300,h_200,c_fill/... returns a 300×200 crop
//   • Free tier — 25GB storage, 25GB bandwidth/month (plenty for learning)
//
// Flow:
//   User picks file in browser
//     → multipart form POST to your Go server
//     → Go server receives the file bytes
//     → Cloudinary SDK uploads bytes to Cloudinary
//     → Cloudinary returns a JSON response with the secure URL
//     → Go server stores the URL in the database
//     → Browser displays the image via the URL
package services

import (
	"context"
	"fmt"
	"mime/multipart"

	"gamelog/config"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

// CloudinaryEnabled returns true if Cloudinary credentials are configured.
// When false, image upload fields are hidden in the UI.
func CloudinaryEnabled() bool {
	return config.App.CloudinaryCloudName != ""
}

// UploadImage uploads a multipart file to Cloudinary and returns the secure HTTPS URL.
//
// folder: organizes files in your Cloudinary media library (e.g. "covers", "screenshots")
//
// LEARN: The Cloudinary SDK handles:
//   - Multipart chunking for large files
//   - Authentication (signs the request with your API secret)
//   - Retry logic
// You just give it a file and a folder, and it gives back a URL.
func UploadImage(file multipart.File, folder string) (string, error) {
	if !CloudinaryEnabled() {
		return "", fmt.Errorf("cloudinary credentials not set — add them to .env")
	}

	// NewFromParams creates a Cloudinary client using your cloud name + API key + secret.
	// LEARN: The API secret never leaves your server — it's used to sign upload requests.
	cld, err := cloudinary.NewFromParams(
		config.App.CloudinaryCloudName,
		config.App.CloudinaryAPIKey,
		config.App.CloudinaryAPISecret,
	)
	if err != nil {
		return "", fmt.Errorf("cloudinary: init failed: %w", err)
	}

	ctx := context.Background()

	// Upload.Upload sends the file to Cloudinary.
	// UploadParams configures the upload: folder path, resource type, auto-transforms.
	result, err := cld.Upload.Upload(ctx, file, uploader.UploadParams{
		Folder:       "gamelog/" + folder, // e.g. "gamelog/covers" or "gamelog/screenshots"
		ResourceType: "image",
		// LEARN: Transformation runs automatically on upload. This resizes to max 800px wide,
		// auto-selects quality and format (WebP where supported) — smaller files, same quality.
		Transformation: "c_limit,w_800,q_auto,f_auto",
	})
	if err != nil {
		return "", fmt.Errorf("cloudinary: upload failed: %w", err)
	}

	// SecureURL is the HTTPS CDN URL for the uploaded image.
	// Example: https://res.cloudinary.com/your_cloud/image/upload/v1234/gamelog/covers/xyz.webp
	return result.SecureURL, nil
}
