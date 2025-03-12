package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Storage  StorageConfig
}

type ServerConfig struct {
	Port string
	Env  string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type JWTConfig struct {
	Secret     string
	Expiration string
}

type StorageConfig struct {
	Path          string
	MaxUploadSize int64
	Provider      string
	SeaweedFS     SeaweedFSConfig
	S3            S3Config
}

type SeaweedFSConfig struct {
	MasterURL  string
	Container  string
	Volume     string
	MasterPort int
	VolumePort int
	DataDir    string
	VolumeMax  int
	Replicas   int
}

type S3Config struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	PublicURL       string
	Endpoint        string
	ForcePathStyle  bool
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("error loading .env file: %v", err)
	}

	config := &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
			Env:  getEnv("ENV", "development"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "media_center"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", "your-secret-key"),
			Expiration: getEnv("JWT_EXPIRATION", "24h"),
		},
		Storage: StorageConfig{
			Path:          getEnv("STORAGE_PATH", "./storage/media"),
			MaxUploadSize: int64(getEnvAsInt("MAX_UPLOAD_SIZE", 10485760)),
			Provider:      getEnv("STORAGE_PROVIDER", "seaweedfs"),
			SeaweedFS: SeaweedFSConfig{
				MasterURL:  getEnv("SEAWEEDFS_MASTER_URL", "http://localhost:9333"),
				Container:  getEnv("SEAWEED_CONTAINER", "media-center-seaweedfs"),
				Volume:     getEnv("SEAWEED_VOLUME", "media-center-seaweedfs-data"),
				MasterPort: getEnvAsInt("SEAWEED_MASTER_PORT", 9333),
				VolumePort: getEnvAsInt("SEAWEED_VOLUME_PORT", 8080),
				DataDir:    getEnv("SEAWEED_DATA_DIR", "/data"),
				VolumeMax:  getEnvAsInt("SEAWEED_VOLUME_MAX", 30000),
				Replicas:   getEnvAsInt("SEAWEED_REPLICAS", 1),
			},
			S3: S3Config{
				Region:          getEnv("AWS_REGION", "us-east-1"),
				AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
				SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
				BucketName:      getEnv("AWS_BUCKET_NAME", ""),
				PublicURL:       getEnv("AWS_PUBLIC_URL", ""),
				Endpoint:        getEnv("AWS_ENDPOINT", ""),
				ForcePathStyle:  getEnvAsBool("AWS_FORCE_PATH_STYLE", false),
			},
		},
	}

	return config, nil
}

func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode)
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		var intVal int
		if _, err := fmt.Sscanf(value, "%d", &intVal); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}
