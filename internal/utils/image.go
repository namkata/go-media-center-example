package utils

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/disintegration/imaging"
)

// TransformationOptions defines the available image transformation options
type TransformationOptions struct {
	Width   int    // Width in pixels
	Height  int    // Height in pixels
	Fit     string // Fit mode: "contain", "cover", "fill"
	Crop    string // Crop position: "center", "top", "bottom", "left", "right"
	Quality int    // JPEG quality (1-100)
	Format  string // Output format: "jpeg", "png", "webp"
	Preset  string // Predefined transformation preset
	Fresh   bool   // Force fresh transformation
}

// IsEmpty checks if any transformation options are set
func (t *TransformationOptions) IsEmpty() bool {
	return t.Width == 0 && t.Height == 0 && t.Fit == "" && t.Crop == "" &&
		t.Quality == 0 && t.Format == "" && t.Preset == "" && !t.Fresh
}

// Validate checks if the transformation options are valid
func (t *TransformationOptions) Validate() error {
	// Check dimensions
	if t.Width < 0 || t.Height < 0 {
		return fmt.Errorf("width and height must be non-negative")
	}

	// Maximum dimension increased to 16384 (16K resolution)
	maxDimension := 16384
	if t.Width > maxDimension || t.Height > maxDimension {
		return fmt.Errorf("maximum allowed dimension is %d pixels", maxDimension)
	}

	// Check fit mode
	if t.Fit != "" && t.Fit != "contain" && t.Fit != "cover" && t.Fit != "fill" {
		return fmt.Errorf("invalid fit mode: %s", t.Fit)
	}

	// Check crop position
	if t.Crop != "" && t.Crop != "center" && t.Crop != "top" && t.Crop != "bottom" && t.Crop != "left" && t.Crop != "right" {
		return fmt.Errorf("invalid crop position: %s", t.Crop)
	}

	// Check quality
	if t.Quality < 0 || t.Quality > 100 {
		return fmt.Errorf("quality must be between 0 and 100")
	}

	// Check format
	if t.Format != "" && t.Format != "jpeg" && t.Format != "jpg" && t.Format != "png" && t.Format != "webp" {
		return fmt.Errorf("unsupported format: %s", t.Format)
	}

	return nil
}

// TransformImage applies the specified transformations to an image
func TransformImage(input io.Reader, options TransformationOptions) ([]byte, error) {

	// If no parameter header
	if options.Width == 0 && options.Height == 0 && options.Fit == "" && options.Crop == "" && options.Format == "" {
		originalBytes, err := io.ReadAll(input)
		if err != nil {
			return nil, fmt.Errorf("failed to read original image: %v", err)
		}
		return originalBytes, nil
	}

	// Decode the input image
	src, format, err := image.Decode(input)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %v", err)
	}

	// Get original dimensions
	bounds := src.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()

	// Convert to NRGBA to ensure consistent color space
	img := imaging.Clone(src)

	// Apply transformations
	var transformed *image.NRGBA

	// Handle resizing based on fit mode
	if options.Width > 0 || options.Height > 0 {
		// Calculate target dimensions while maintaining aspect ratio
		targetWidth := options.Width
		targetHeight := options.Height

		// If only one dimension is specified, calculate the other
		if targetWidth == 0 {
			// Calculate width based on height while maintaining aspect ratio
			targetWidth = int(float64(origWidth) * float64(targetHeight) / float64(origHeight))
			fmt.Printf("Calculated width: %d (based on height: %d)\n", targetWidth, targetHeight)
		} else if targetHeight == 0 {
			// Calculate height based on width while maintaining aspect ratio
			targetHeight = int(float64(origHeight) * float64(targetWidth) / float64(origWidth))
			fmt.Printf("Calculated height: %d (based on width: %d)\n", targetHeight, targetWidth)
		}

		fmt.Printf("Target dimensions: %dx%d\n", targetWidth, targetHeight)

		// Apply resize based on fit mode
		switch options.Fit {
		case "contain":
			transformed = imaging.Fit(img, targetWidth, targetHeight, imaging.Lanczos)
		case "cover":
			transformed = imaging.Fill(img, targetWidth, targetHeight, imaging.Center, imaging.Lanczos)
		case "fill":
			transformed = imaging.Resize(img, targetWidth, targetHeight, imaging.Lanczos)
		default:
			// Default to contain if no fit specified
			transformed = imaging.Fit(img, targetWidth, targetHeight, imaging.Lanczos)
		}

		// Log final dimensions after resize
		finalBounds := transformed.Bounds()
		fmt.Printf("Final dimensions after resize: %dx%d\n", finalBounds.Dx(), finalBounds.Dy())
	}

	fmt.Println("Crop:", options.Crop)

	// Handle cropping if specified
	if options.Crop != "" {
		currentBounds := transformed.Bounds()
		currentWidth := currentBounds.Dx()
		currentHeight := currentBounds.Dy()

		fmt.Printf("Pre-crop dimensions: %dx%d\n", currentWidth, currentHeight)

		// Determine crop demesions
		cropWidth := options.Width
		cropHeight := options.Height

		if cropWidth == 0 || cropWidth > currentWidth {
			cropWidth = currentWidth
		}

		if cropHeight == 0 || cropHeight > currentHeight {
			cropHeight = currentHeight
		}

		fmt.Printf("Crop dimensions: %dx%d\n", cropWidth, cropHeight)

		// Calculate crop anchor
		var anchor imaging.Anchor
		switch options.Crop {
		case "top":
			anchor = imaging.Top
		case "bottom":
			anchor = imaging.Bottom
		case "left":
			anchor = imaging.Left
		case "right":
			anchor = imaging.Right
		default:
			anchor = imaging.Center
		}

		// Apply crop
		transformed = imaging.CropAnchor(transformed, cropWidth, cropHeight, anchor)

		// Log final dimensions after crop
		finalBounds := transformed.Bounds()
		fmt.Printf("Final dimensions after crop: %dx%d\n", finalBounds.Dx(), finalBounds.Dy())
	}

	// Encode the transformed image
	var buf bytes.Buffer
	outputFormat := options.Format
	if outputFormat == "" {
		outputFormat = format
	}

	fmt.Printf("Encoding to format: %s\n", outputFormat)

	switch outputFormat {
	case "jpeg", "jpg":
		quality := options.Quality
		if quality == 0 {
			quality = 85 // Default quality
		}
		fmt.Printf("JPEG quality: %d\n", quality)
		err = jpeg.Encode(&buf, transformed, &jpeg.Options{Quality: quality})
	case "png":
		err = png.Encode(&buf, transformed)
	case "webp":
		// If webp is needed, you'll need to add the webp package and implement webp encoding
		return nil, fmt.Errorf("webp format not yet supported")
	default:
		// Default to JPEG if format is not specified or unknown
		err = jpeg.Encode(&buf, transformed, &jpeg.Options{Quality: 85})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode transformed image: %v", err)
	}

	finalSize := buf.Len()
	fmt.Printf("Final image size: %d bytes\n", finalSize)

	return buf.Bytes(), nil
}

// ApplyPreset applies a predefined transformation preset
func ApplyPreset(options *TransformationOptions, preset string) error {
	switch preset {
	case "thumbnail":
		options.Width = 150
		options.Height = 150
		options.Fit = "cover"
		options.Quality = 80
	case "social":
		options.Width = 1200
		options.Height = 630
		options.Fit = "contain"
		options.Quality = 85
	case "avatar":
		options.Width = 300
		options.Height = 300
		options.Fit = "cover"
		options.Quality = 85
	case "banner":
		options.Width = 1920
		options.Height = 400
		options.Fit = "cover"
		options.Quality = 90
	default:
		return fmt.Errorf("unknown preset: %s", preset)
	}
	return nil
}
