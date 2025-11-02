package services

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type CloudinaryService struct {
	cld *cloudinary.Cloudinary
}

var Cloudinary *CloudinaryService

func InitializeCloudinary(cloudinaryURL string) error {
	if cloudinaryURL == "" {
		return fmt.Errorf("cloudinary URL is required")
	}

	fmt.Printf("Attempting to initialize Cloudinary with URL: %s\n", cloudinaryURL)
	
	cld, err := cloudinary.NewFromURL(cloudinaryURL)
	if err != nil {
		fmt.Printf("Cloudinary initialization failed: %v\n", err)
		return fmt.Errorf("failed to initialize Cloudinary: %w", err)
	}

	Cloudinary = &CloudinaryService{cld: cld}
	fmt.Printf("Cloudinary initialized successfully\n")
	return nil
}

func (cs *CloudinaryService) UploadImage(file multipart.File, folder string) (*uploader.UploadResult, error) {
	ctx := context.Background()
	
	// Generate unique public ID
	publicID := fmt.Sprintf("%s/%d", folder, time.Now().UnixNano())
	
	result, err := cs.cld.Upload.Upload(ctx, file, uploader.UploadParams{
		PublicID: publicID,
		Folder:   folder,
		UseFilename: &[]bool{true}[0],
		UniqueFilename: &[]bool{true}[0],
		Overwrite: &[]bool{false}[0],
		ResourceType: "image",
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}
    // Normalize URLs to HTTPS to avoid production blocking
    if result != nil {
        if result.URL != "" {
            result.URL = forceHTTPS(result.URL)
        }
        if result.SecureURL != "" {
            result.SecureURL = forceHTTPS(result.SecureURL)
        } else if result.URL != "" {
            // Fallback: set SecureURL from URL
            result.SecureURL = forceHTTPS(result.URL)
        }
    }

	return result, nil
}

func (cs *CloudinaryService) UploadImageFromBytes(data []byte, folder, filename string) (*uploader.UploadResult, error) {
	ctx := context.Background()
	
	// Generate unique public ID
	publicID := fmt.Sprintf("%s/%s_%d", folder, strings.TrimSuffix(filename, filepath.Ext(filename)), time.Now().UnixNano())
	
	// Convert bytes to reader
	reader := bytes.NewReader(data)
	
	result, err := cs.cld.Upload.Upload(ctx, reader, uploader.UploadParams{
		PublicID: publicID,
		Folder:   folder,
		UseFilename: &[]bool{true}[0],
		UniqueFilename: &[]bool{true}[0],
		Overwrite: &[]bool{false}[0],
		ResourceType: "image",
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}
    // Normalize URLs to HTTPS to avoid production blocking
    if result != nil {
        if result.URL != "" {
            result.URL = forceHTTPS(result.URL)
        }
        if result.SecureURL != "" {
            result.SecureURL = forceHTTPS(result.SecureURL)
        } else if result.URL != "" {
            result.SecureURL = forceHTTPS(result.URL)
        }
    }

	return result, nil
}

func (cs *CloudinaryService) DeleteImage(publicID string) error {
	ctx := context.Background()
	
	_, err := cs.cld.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID: publicID,
		ResourceType: "image",
	})
	
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}
	
	return nil
}

func (cs *CloudinaryService) GetImageURL(publicID string, transformations ...string) string {
	// For Cloudinary v2, we need to construct the URL manually
	// This is a simplified version - in production you might want to use a proper URL builder
    baseURL := "https://res.cloudinary.com"
	cloudName := cs.cld.Config.Cloud.CloudName
	
	if cloudName == "" {
		return ""
	}
	
	url := fmt.Sprintf("%s/%s/image/upload", baseURL, cloudName)
	
	// Add transformations if provided
	if len(transformations) > 0 {
		url += "/" + strings.Join(transformations, ",")
	}
	
	url += "/" + publicID
	
    return forceHTTPS(url)
}

func (cs *CloudinaryService) GenerateTransformationURL(publicID string, width, height int, crop string) string {
	transformations := []string{
		fmt.Sprintf("w_%d", width),
		fmt.Sprintf("h_%d", height),
		fmt.Sprintf("c_%s", crop),
		"q_auto",
		"f_auto",
	}
	
    return forceHTTPS(cs.GetImageURL(publicID, transformations...))
}

// Helper function to extract public ID from Cloudinary URL
func ExtractPublicID(url string) string {
	// Cloudinary URLs typically look like: https://res.cloudinary.com/account/image/upload/v1234567890/folder/filename.jpg
	parts := strings.Split(url, "/")
	if len(parts) < 4 {
		return ""
	}
    
	// Find the "upload" part and take everything after it
	for i, part := range parts {
		if part == "upload" && i+1 < len(parts) {
			// Join everything after "upload" and remove the version prefix if present
			path := strings.Join(parts[i+1:], "/")
			// Remove version prefix (v1234567890/)
			if strings.Contains(path, "/") {
				pathParts := strings.Split(path, "/")
				if len(pathParts) > 1 && strings.HasPrefix(pathParts[0], "v") {
					path = strings.Join(pathParts[1:], "/")
				}
			}
			// Remove file extension
			return strings.TrimSuffix(path, filepath.Ext(path))
		}
	}
	
	return ""
}

// UploadVideo uploads a video file to Cloudinary
func (cs *CloudinaryService) UploadVideo(file multipart.File, folder string) (*uploader.UploadResult, error) {
	ctx := context.Background()
	
	// Generate unique public ID
	publicID := fmt.Sprintf("%s/%d", folder, time.Now().UnixNano())
	
	result, err := cs.cld.Upload.Upload(ctx, file, uploader.UploadParams{
		PublicID: publicID,
		Folder:   folder,
		UseFilename: &[]bool{true}[0],
		UniqueFilename: &[]bool{true}[0],
		Overwrite: &[]bool{false}[0],
		ResourceType: "video",
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to upload video: %w", err)
	}
	
	// Normalize URLs to HTTPS
	if result != nil {
		if result.URL != "" {
			result.URL = forceHTTPS(result.URL)
		}
		if result.SecureURL != "" {
			result.SecureURL = forceHTTPS(result.SecureURL)
		} else if result.URL != "" {
			result.SecureURL = forceHTTPS(result.URL)
		}
	}

	return result, nil
}

// Helper functions for handlers to use directly with multipart.FileHeader
func UploadImageToCloudinary(fileHeader *multipart.FileHeader) (string, error) {
	if Cloudinary == nil {
		return "", fmt.Errorf("Cloudinary not initialized")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	result, err := Cloudinary.UploadImage(file, "melhaf_images")
	if err != nil {
		return "", err
	}

	if result.SecureURL != "" {
		return result.SecureURL, nil
	}
	return result.URL, nil
}

func UploadVideoToCloudinary(fileHeader *multipart.FileHeader) (string, error) {
	if Cloudinary == nil {
		return "", fmt.Errorf("Cloudinary not initialized")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	result, err := Cloudinary.UploadVideo(file, "melhaf_videos")
	if err != nil {
		return "", err
	}

	if result.SecureURL != "" {
		return result.SecureURL, nil
	}
	return result.URL, nil
}

// UploadMaisonAdrarPerfumeImage uploads an image to Cloudinary for Maison Adrar perfumes
func UploadMaisonAdrarPerfumeImage(fileHeader *multipart.FileHeader) (string, error) {
	if Cloudinary == nil {
		return "", fmt.Errorf("Cloudinary not initialized")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	result, err := Cloudinary.UploadImage(file, "maison_adrar_perfumes")
	if err != nil {
		return "", err
	}

	if result.SecureURL != "" {
		return result.SecureURL, nil
	}
	return result.URL, nil
}

// UploadMaisonAdrarBackground uploads a background image to Cloudinary for Maison Adrar collections
func UploadMaisonAdrarBackground(fileHeader *multipart.FileHeader) (string, error) {
	if Cloudinary == nil {
		return "", fmt.Errorf("Cloudinary not initialized")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	result, err := Cloudinary.UploadImage(file, "maison_adrar_backgrounds")
	if err != nil {
		return "", err
	}

	if result.SecureURL != "" {
		return result.SecureURL, nil
	}
	return result.URL, nil
}

// UploadMaisonAdrarBanner uploads a banner image to Cloudinary for Maison Adrar
func UploadMaisonAdrarBanner(fileHeader *multipart.FileHeader) (string, error) {
	if Cloudinary == nil {
		return "", fmt.Errorf("Cloudinary not initialized")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	result, err := Cloudinary.UploadImage(file, "maison_adrar_banners")
	if err != nil {
		return "", err
	}

	if result.SecureURL != "" {
		return result.SecureURL, nil
	}
	return result.URL, nil
}

// forceHTTPS ensures Cloudinary URLs use https scheme
func forceHTTPS(in string) string {
    if in == "" {
        return in
    }
    // Trim whitespace and force https
    out := strings.TrimSpace(in)
    out = strings.Replace(out, "http://", "https://", 1)
    return out
}
