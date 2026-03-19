package config

import (
	"errors"
	"os"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	// MongoURI is the MongoDB connection URI. Required.
	// Source: KFLOW_MONGO_URI
	MongoURI string

	// MongoDB is the database name. Defaults to "kflow".
	// Source: KFLOW_MONGO_DB
	MongoDB string

	// Namespace is the Kubernetes namespace used for all workloads.
	// Defaults to "kflow".
	// Source: KFLOW_NAMESPACE
	Namespace string
}

// LoadConfig reads configuration from environment variables.
// Returns an error if any required variable is missing.
func LoadConfig() (*Config, error) {
	mongoURI := os.Getenv("KFLOW_MONGO_URI")
	if mongoURI == "" {
		return nil, errors.New("config: KFLOW_MONGO_URI is required but not set")
	}

	mongoDB := os.Getenv("KFLOW_MONGO_DB")
	if mongoDB == "" {
		mongoDB = "kflow"
	}

	namespace := os.Getenv("KFLOW_NAMESPACE")
	if namespace == "" {
		namespace = "kflow"
	}

	return &Config{
		MongoURI:  mongoURI,
		MongoDB:   mongoDB,
		Namespace: namespace,
	}, nil
}
