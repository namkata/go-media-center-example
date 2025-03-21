// Package docs Code generated by swaggo/swag. DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "termsOfService": "http://example.com/terms/",
        "contact": {
            "name": "API Support",
            "url": "http://example.com/support",
            "email": "support@example.com"
        },
        "license": {
            "name": "MIT",
            "url": "https://opensource.org/licenses/MIT"
        },
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/media": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Get paginated list of media files with optional filters",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "media"
                ],
                "summary": "List media files",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "Page number (default 1)",
                        "name": "page",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "Items per page (default 10)",
                        "name": "limit",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "File type filter",
                        "name": "type",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Search term",
                        "name": "search",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Folder ID",
                        "name": "folder_id",
                        "in": "query"
                    },
                    {
                        "type": "array",
                        "items": {
                            "type": "string"
                        },
                        "collectionFormat": "csv",
                        "description": "Tags filter",
                        "name": "tags",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "media": {
                                    "type": "array",
                                    "items": {
                                        "$ref": "#/definitions/go-media-center-example_internal_models.Media"
                                    }
                                },
                                "pagination": {
                                    "type": "object",
                                    "properties": {
                                        "current_page": {
                                            "type": "integer"
                                        },
                                        "per_page": {
                                            "type": "integer"
                                        },
                                        "total_items": {
                                            "type": "integer"
                                        },
                                        "total_pages": {
                                            "type": "integer"
                                        }
                                    }
                                }
                            }
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "error": {
                                    "type": "string"
                                }
                            }
                        }
                    }
                }
            }
        },
        "/media/bulk-upload": {
            "post": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Upload multiple files at once with shared folder and tags",
                "consumes": [
                    "multipart/form-data"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "media"
                ],
                "summary": "Upload multiple media files",
                "parameters": [
                    {
                        "type": "file",
                        "description": "Media files",
                        "name": "files",
                        "in": "formData",
                        "required": true
                    },
                    {
                        "type": "string",
                        "description": "Folder ID",
                        "name": "folder_id",
                        "in": "formData"
                    },
                    {
                        "type": "array",
                        "items": {
                            "type": "string"
                        },
                        "collectionFormat": "csv",
                        "description": "Tags",
                        "name": "tags",
                        "in": "formData"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "message": {
                                    "type": "string"
                                },
                                "results": {
                                    "type": "array",
                                    "items": {
                                        "type": "object"
                                    }
                                },
                                "success_count": {
                                    "type": "integer"
                                },
                                "total": {
                                    "type": "integer"
                                }
                            }
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "error": {
                                    "type": "string"
                                }
                            }
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "error": {
                                    "type": "string"
                                }
                            }
                        }
                    }
                }
            }
        },
        "/media/files/{filename}": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Serve media file with optional transformations",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "*/*"
                ],
                "tags": [
                    "media"
                ],
                "summary": "Serve media file",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Filename",
                        "name": "filename",
                        "in": "path",
                        "required": true
                    },
                    {
                        "type": "integer",
                        "description": "Width in pixels",
                        "name": "width",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "Height in pixels",
                        "name": "height",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Fit method (contain, cover, fill)",
                        "name": "fit",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Crop position",
                        "name": "crop",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "JPEG/WebP quality (1-100)",
                        "name": "quality",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Output format (jpeg, png, webp)",
                        "name": "format",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Transformation preset",
                        "name": "preset",
                        "in": "query"
                    },
                    {
                        "type": "boolean",
                        "description": "Bypass cache",
                        "name": "fresh",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "file"
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "error": {
                                    "type": "string"
                                }
                            }
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "error": {
                                    "type": "string"
                                }
                            }
                        }
                    }
                }
            }
        },
        "/media/upload": {
            "post": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Upload a new media file with optional folder and tags",
                "consumes": [
                    "multipart/form-data"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "media"
                ],
                "summary": "Upload media file",
                "parameters": [
                    {
                        "type": "file",
                        "description": "Media file",
                        "name": "file",
                        "in": "formData",
                        "required": true
                    },
                    {
                        "type": "string",
                        "description": "Folder ID",
                        "name": "folder_id",
                        "in": "formData"
                    },
                    {
                        "type": "array",
                        "items": {
                            "type": "string"
                        },
                        "collectionFormat": "csv",
                        "description": "Tags",
                        "name": "tags",
                        "in": "formData"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "media": {
                                    "$ref": "#/definitions/go-media-center-example_internal_models.Media"
                                },
                                "message": {
                                    "type": "string"
                                }
                            }
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "error": {
                                    "type": "string"
                                }
                            }
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "error": {
                                    "type": "string"
                                }
                            }
                        }
                    }
                }
            }
        },
        "/media/upload-url": {
            "post": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Download and upload a file from a remote URL",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "media"
                ],
                "summary": "Upload media from URL",
                "parameters": [
                    {
                        "description": "URL upload data",
                        "name": "input",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "type": "object",
                            "properties": {
                                "filename": {
                                    "type": "string"
                                },
                                "folder_id": {
                                    "type": "string"
                                },
                                "tags": {
                                    "type": "array",
                                    "items": {
                                        "type": "string"
                                    }
                                },
                                "url": {
                                    "type": "string"
                                }
                            }
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "media": {
                                    "$ref": "#/definitions/go-media-center-example_internal_models.Media"
                                },
                                "message": {
                                    "type": "string"
                                }
                            }
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "error": {
                                    "type": "string"
                                }
                            }
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "error": {
                                    "type": "string"
                                }
                            }
                        }
                    }
                }
            }
        },
        "/media/{id}": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Get media by ID with optional URL expiration time",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "media"
                ],
                "summary": "Get media details with presigned URL",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Media ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    },
                    {
                        "type": "integer",
                        "description": "URL expiration time in seconds (default 86400)",
                        "name": "expires",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "folder": {
                                    "type": "object",
                                    "properties": {
                                        "id": {
                                            "type": "string"
                                        },
                                        "name": {
                                            "type": "string"
                                        }
                                    }
                                },
                                "media": {
                                    "$ref": "#/definitions/go-media-center-example_internal_models.SwaggerMedia"
                                }
                            }
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "error": {
                                    "type": "string"
                                }
                            }
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "error": {
                                    "type": "string"
                                }
                            }
                        }
                    }
                }
            },
            "put": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Update filename, folder, metadata or tags for a media item",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "media"
                ],
                "summary": "Update media details",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Media ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    },
                    {
                        "description": "Media update data",
                        "name": "input",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "type": "object",
                            "properties": {
                                "filename": {
                                    "type": "string"
                                },
                                "folder_id": {
                                    "type": "string"
                                },
                                "metadata": {
                                    "type": "object"
                                },
                                "tags": {
                                    "type": "array",
                                    "items": {
                                        "type": "string"
                                    }
                                }
                            }
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/go-media-center-example_internal_models.Media"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "error": {
                                    "type": "string"
                                }
                            }
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "error": {
                                    "type": "string"
                                }
                            }
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "error": {
                                    "type": "string"
                                }
                            }
                        }
                    }
                }
            },
            "delete": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Delete media file and its metadata",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "media"
                ],
                "summary": "Delete media",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Media ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "message": {
                                    "type": "string"
                                }
                            }
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "error": {
                                    "type": "string"
                                }
                            }
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "error": {
                                    "type": "string"
                                }
                            }
                        }
                    }
                }
            }
        },
        "/media/{id}/transform": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Apply transformations to an image (resize, crop, format conversion)",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "image/jpeg",
                    "image/png",
                    "image/webp"
                ],
                "tags": [
                    "media"
                ],
                "summary": "Transform image",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Media ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    },
                    {
                        "type": "integer",
                        "description": "Width in pixels",
                        "name": "width",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "Height in pixels",
                        "name": "height",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Fit method (contain, cover, fill)",
                        "name": "fit",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Crop position (center, top, bottom, left, right)",
                        "name": "crop",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "JPEG/WebP quality (1-100)",
                        "name": "quality",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Output format (jpeg, png, webp)",
                        "name": "format",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Transformation preset",
                        "name": "preset",
                        "in": "query"
                    },
                    {
                        "type": "boolean",
                        "description": "Bypass cache",
                        "name": "fresh",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "file"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "details": {
                                    "type": "string"
                                },
                                "error": {
                                    "type": "string"
                                }
                            }
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "error": {
                                    "type": "string"
                                }
                            }
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "details": {
                                    "type": "string"
                                },
                                "error": {
                                    "type": "string"
                                }
                            }
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "go-media-center-example_internal_models.Media": {
            "type": "object",
            "properties": {
                "createdAt": {
                    "type": "string"
                },
                "deletedAt": {
                    "$ref": "#/definitions/gorm.DeletedAt"
                },
                "filename": {
                    "type": "string"
                },
                "folderID": {
                    "type": "string"
                },
                "id": {
                    "type": "string"
                },
                "metadata": {
                    "type": "array",
                    "items": {
                        "type": "integer"
                    }
                },
                "mimeType": {
                    "type": "string"
                },
                "path": {
                    "type": "string"
                },
                "size": {
                    "type": "integer"
                },
                "tags": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/go-media-center-example_internal_models.Tag"
                    }
                },
                "updatedAt": {
                    "type": "string"
                },
                "userID": {
                    "type": "integer"
                }
            }
        },
        "go-media-center-example_internal_models.SwaggerMedia": {
            "description": "Media file information",
            "type": "object",
            "properties": {
                "created_at": {
                    "type": "string",
                    "example": "2023-01-01T12:00:00Z"
                },
                "filename": {
                    "type": "string",
                    "example": "vacation.jpg"
                },
                "folder_id": {
                    "type": "string",
                    "example": "folder123"
                },
                "id": {
                    "type": "string",
                    "example": "3f8d9a7c-5e4b-4b3a-8e1d-7f6b5c4d3a2b"
                },
                "metadata": {
                    "type": "object"
                },
                "mime_type": {
                    "type": "string",
                    "example": "image/jpeg"
                },
                "path": {
                    "type": "string",
                    "example": "uploads/vacation.jpg"
                },
                "size": {
                    "type": "integer",
                    "example": 1024000
                },
                "tags": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/go-media-center-example_internal_models.Tag"
                    }
                },
                "updated_at": {
                    "type": "string",
                    "example": "2023-01-01T12:00:00Z"
                },
                "user_id": {
                    "type": "integer",
                    "example": 1
                }
            }
        },
        "go-media-center-example_internal_models.Tag": {
            "type": "object",
            "properties": {
                "createdAt": {
                    "type": "string"
                },
                "deletedAt": {
                    "$ref": "#/definitions/gorm.DeletedAt"
                },
                "id": {
                    "type": "integer"
                },
                "media": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/go-media-center-example_internal_models.Media"
                    }
                },
                "name": {
                    "type": "string"
                },
                "updatedAt": {
                    "type": "string"
                }
            }
        },
        "gorm.DeletedAt": {
            "type": "object",
            "properties": {
                "time": {
                    "type": "string"
                },
                "valid": {
                    "description": "Valid is true if Time is not NULL",
                    "type": "boolean"
                }
            }
        }
    },
    "securityDefinitions": {
        "BearerAuth": {
            "description": "Type \"Bearer\" followed by a space and JWT token",
            "type": "apiKey",
            "name": "Authorization",
            "in": "header"
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "localhost:8080",
	BasePath:         "/api",
	Schemes:          []string{},
	Title:            "Media Center API",
	Description:      "A media management system with support for images, videos, and documents",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
