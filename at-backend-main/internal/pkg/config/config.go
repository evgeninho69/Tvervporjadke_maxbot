package config

import (
	"activeTver/internal/pkg/logs"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
	"unicode"

	"github.com/go-playground/validator/v10"
)

type Config struct {
	Server struct {
		Host              string
		Port              string `validate:"number,gte=0,lte=65535"`
		Domain            string
		ReadTimeout       time.Duration
		ReadHeaderTimeout time.Duration
		WriteTimeout      time.Duration
		IdleTimeout       time.Duration
	}
	Storage struct {
		Host     string
		Port     string `validate:"number,gte=0,lte=65535"`
		Name     string
		User     string
		Password string
	}
	TLS struct {
		Certificate string `validate:"filepath"` // Path to tls certificate
		Key         string `validate:"filepath"` // Path to tls key
	}
	Tokens struct {
		JWTSecretKey         string
		JWTDuration          time.Duration
		RefreshTokenLength   uint64
		RefreshTokenDuration time.Duration
		SecretKeyToEncrypt   string
	}
	Email struct {
		Host           string
		Port           string `validate:"number,gte=0,lte=65535"`
		SenderEmail    string
		SenderPassword string
		SupportEmail   string
	}
	ObjectStorage struct {
		AccessKeyID     string
		AccessKeySecret string
		Endpoint        string
		SessionToken    string
		Bucket          string
		AccessHost      string
	}
}

const opCfg = "config.GetConfig"

func defError(key string, def any) {
	logs.Warn(
		fmt.Sprintf("An error occurred or the %v environment variable was not found. The default value is %v", key, def),
		"nil",
		opCfg,
	)
}

func getEnvAsString(key, def string) string {
	envVal, ok := os.LookupEnv(key)
	if !ok {
		defError(key, def)
		return def
	}

	return envVal
}

func getEnvAsUint64(key string, def uint64) uint64 {
	envVal, ok := os.LookupEnv(key)
	if !ok {
		defError(key, def)
		return def
	}

	val, err := strconv.ParseUint(envVal, 10, 64)
	if err != nil {
		defError(key, def)
		return def
	}

	return val
}

func getEnvAsTime(key string, def time.Duration) time.Duration {
	envVal, ok := os.LookupEnv(key)
	if !ok {
		defError(key, def)
		return def
	}

	pos := 0
	for i, c := range envVal {
		if !unicode.IsDigit(c) {
			pos = i
			break
		}

		if i == len(envVal)-1 {
			defError(key, def)
			return def
		}
	}

	num, err := strconv.Atoi(envVal[:pos])
	if err != nil {
		defError(key, def)
		return def
	}

	val := time.Duration(num)
	switch envVal[pos:] {
	case "ms":
		val *= time.Millisecond
	case "s":
		val *= time.Second
	case "m":
		val *= time.Minute
	case "h":
		val *= time.Hour
	default:
		defError(key, def)
		return def
	}

	return val
}

var instance *Config
var once sync.Once

func GetConfig() (*Config, error) {
	var configErr error
	once.Do(func() {
		instance = &Config{}

		serverCfg := &instance.Server
		serverCfg.Host = getEnvAsString("BACKEND_HOST", "127.0.0.1")
		serverCfg.Port = getEnvAsString("BACKEND_PORT", "8080")
		serverCfg.Domain = getEnvAsString("BACKEND_DOMAIN", "127.0.0.1.com") // need to be added to hosts
		serverCfg.ReadTimeout = getEnvAsTime("BACKEND_READ_TIMEOUT", 250*time.Millisecond)
		serverCfg.ReadHeaderTimeout = getEnvAsTime("BACKEND_READ_HEADER_TIMEOUT", 250*time.Millisecond)
		serverCfg.WriteTimeout = getEnvAsTime("BACKEND_WRITE_TIMEOUT", 1*time.Second)
		serverCfg.IdleTimeout = getEnvAsTime("BACKEND_IDLE_TIMEOUT", 5*time.Minute)

		storageCfg := &instance.Storage
		storageCfg.Host = getEnvAsString("STORAGE_HOST", "127.0.0.1")
		storageCfg.Port = getEnvAsString("STORAGE_PORT", "5432")
		storageCfg.Name = getEnvAsString("STORAGE_NAME", "storage")
		storageCfg.User = getEnvAsString("STORAGE_USER", "user")
		storageCfg.Password = getEnvAsString("STORAGE_PASSWORD", "pwd")

		tlsCfg := &instance.TLS
		tlsCfg.Certificate = getEnvAsString("TLS_CERTIFICATE", "tls.crt")
		tlsCfg.Key = getEnvAsString("TLS_KEY", "tls.key")

		tokenCfg := &instance.Tokens
		tokenCfg.JWTSecretKey = getEnvAsString("TOKENS_JWT_SECRET_KEY", "default")
		tokenCfg.JWTDuration = getEnvAsTime("TOKENS_JWT_DURATION", 15*time.Minute)
		tokenCfg.RefreshTokenLength = getEnvAsUint64("TOKENS_REFRESH_LENGTH", 64)
		tokenCfg.RefreshTokenDuration = getEnvAsTime("TOKENS_REFRESH_DURATION", 24*31*time.Hour) // month
		tokenCfg.SecretKeyToEncrypt = getEnvAsString("TOKENS_SECRET_KEY", "default")

		emailCfg := &instance.Email
		emailCfg.Host = getEnvAsString("EMAIL_HOST", "None")
		emailCfg.Port = getEnvAsString("EMAIL_PORT", "465")
		emailCfg.SenderEmail = getEnvAsString("EMAIL_SENDER", "default@default.com")
		emailCfg.SenderPassword = getEnvAsString("EMAIL_PASSWORD", "default")
		emailCfg.SupportEmail = getEnvAsString("EMAIL_SUPPORT", "support@default.com")

		objectStorageCfg := &instance.ObjectStorage
		objectStorageCfg.AccessKeyID = getEnvAsString("OBJECT_STORAGE_ACCESS_KEY_ID", "id")
		objectStorageCfg.AccessKeySecret = getEnvAsString("OBJECT_STORAGE_SECRET_ACCESS_KEY", "secret")
		objectStorageCfg.Endpoint = getEnvAsString("OBJECT_STORAGE_ENDPOINT", "127.0.0.1:9000")
		objectStorageCfg.SessionToken = getEnvAsString("OBJECT_STORAGE_SESSION_TOKEN", "")
		objectStorageCfg.Bucket = getEnvAsString("OBJECT_STORAGE_BUCKET", "files")
		objectStorageCfg.AccessHost = getEnvAsString("OBJECT_STORAGE_ACCESS_HOST", "127.0.0.1:9000")

		// Validate config
		validate := validator.New()
		err := validate.Struct(instance)
		if err != nil {
			configErr = err
		}
	})
	return instance, configErr
}
