# Media Transform API Documentation

The Media Transform API allows you to perform various image transformations on your media files. All transformations are performed on-demand and cached for subsequent requests.

## Authentication

All transform API endpoints require authentication. Include your JWT token in the Authorization header:

```bash
curl -H "Authorization: Bearer your_jwt_token" ...
```

## Basic Operations

### 1. Basic Resize

Resize an image to specific dimensions while maintaining aspect ratio:

```bash
curl -X POST \
  "http://localhost:8080/api/v1/media/123/transform?width=800&height=600" \
  -H "Authorization: Bearer your_jwt_token"
```

### 2. Resize with Fit Mode

Control how the image fits within the specified dimensions:

```bash
# Contain mode - fits entire image within dimensions
curl -X POST \
  "http://localhost:8080/api/v1/media/123/transform?width=800&height=600&fit=contain" \
  -H "Authorization: Bearer your_jwt_token"

# Cover mode - covers entire dimensions, may crop
curl -X POST \
  "http://localhost:8080/api/v1/media/123/transform?width=800&height=600&fit=cover" \
  -H "Authorization: Bearer your_jwt_token"

# Fill mode - stretches to exact dimensions
curl -X POST \
  "http://localhost:8080/api/v1/media/123/transform?width=800&height=600&fit=fill" \
  -H "Authorization: Bearer your_jwt_token"
```

### 3. Format Conversion

Convert images between formats and control quality:

```bash
# Convert to WebP with 80% quality
curl -X POST \
  "http://localhost:8080/api/v1/media/123/transform?format=webp&quality=80" \
  -H "Authorization: Bearer your_jwt_token"

# Convert to PNG (lossless)
curl -X POST \
  "http://localhost:8080/api/v1/media/123/transform?format=png" \
  -H "Authorization: Bearer your_jwt_token"

# Convert to JPEG with maximum quality
curl -X POST \
  "http://localhost:8080/api/v1/media/123/transform?format=jpeg&quality=100" \
  -H "Authorization: Bearer your_jwt_token"
```

### 4. Using Presets

Use predefined transformation settings:

```bash
# Thumbnail preset (150x150 cover)
curl -X POST \
  "http://localhost:8080/api/v1/media/123/transform?preset=thumbnail" \
  -H "Authorization: Bearer your_jwt_token"

# Social media preset (1200x630 contain)
curl -X POST \
  "http://localhost:8080/api/v1/media/123/transform?preset=social" \
  -H "Authorization: Bearer your_jwt_token"

# Avatar preset (300x300 cover)
curl -X POST \
  "http://localhost:8080/api/v1/media/123/transform?preset=avatar" \
  -H "Authorization: Bearer your_jwt_token"

# Banner preset (1920x400 cover)
curl -X POST \
  "http://localhost:8080/api/v1/media/123/transform?preset=banner" \
  -H "Authorization: Bearer your_jwt_token"
```

### 5. Crop Operations

Crop a specific region of the image:

```bash
# Crop starting at (100,100) with size 500x300
curl -X POST \
  "http://localhost:8080/api/v1/media/123/transform?crop=100,100,500,300" \
  -H "Authorization: Bearer your_jwt_token"
```

### 6. Combined Operations

Combine multiple transformations in a single request:

```bash
# Resize, convert format, adjust quality, and set fit mode
curl -X POST \
  "http://localhost:8080/api/v1/media/123/transform?width=800&height=600&format=webp&quality=80&fit=cover" \
  -H "Authorization: Bearer your_jwt_token"

# Use preset with format conversion
curl -X POST \
  "http://localhost:8080/api/v1/media/123/transform?preset=social&format=webp&quality=90" \
  -H "Authorization: Bearer your_jwt_token"
```

### 7. Cache Control

Force a fresh transformation (bypass cache):

```bash
curl -X POST \
  "http://localhost:8080/api/v1/media/123/transform?width=800&fresh=true" \
  -H "Authorization: Bearer your_jwt_token"
```

## Response Format

Successful transformations return the transformed image directly with appropriate content type headers:

```
HTTP/1.1 200 OK
Content-Type: image/webp
Cache-Control: public, max-age=31536000
X-Cache: HIT/MISS
```

Error responses return JSON:

```json
{
  "error": "Failed to transform image",
  "details": "Invalid transformation parameters"
}
```

## Error Codes

- `400 Bad Request`: Invalid parameters
- `401 Unauthorized`: Missing or invalid authentication
- `403 Forbidden`: Insufficient permissions
- `404 Not Found`: Media not found
- `422 Unprocessable Entity`: Invalid transformation request
- `500 Internal Server Error`: Server-side error

## Caching

Transformed images are cached automatically. The cache key includes all transformation parameters. To force a fresh transformation, use the `fresh=true` parameter.

Cache headers are set to:
- `Cache-Control: public, max-age=31536000` (1 year) for normal requests
- `Cache-Control: no-cache, no-store, must-revalidate` for fresh transformations

## Limitations

- Maximum output dimensions: 8192x8192 pixels
- Maximum input file size: 100MB
- Supported input formats: JPEG, PNG, GIF, WebP
- Supported output formats: JPEG, PNG, WebP 