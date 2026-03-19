package config_test

import (
	"testing"

	"github.com/pastorenue/kflow/internal/config"
)

func TestLoadConfig_MissingMongoURI(t *testing.T) {
	t.Setenv("KFLOW_MONGO_URI", "")
	t.Setenv("KFLOW_MONGO_DB", "")
	t.Setenv("KFLOW_NAMESPACE", "")

	_, err := config.LoadConfig()
	if err == nil {
		t.Fatal("expected error when KFLOW_MONGO_URI is unset")
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	t.Setenv("KFLOW_MONGO_URI", "mongodb://localhost:27017")
	t.Setenv("KFLOW_MONGO_DB", "")
	t.Setenv("KFLOW_NAMESPACE", "")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MongoDB != "kflow" {
		t.Errorf("expected MongoDB default 'kflow', got %q", cfg.MongoDB)
	}
	if cfg.Namespace != "kflow" {
		t.Errorf("expected Namespace default 'kflow', got %q", cfg.Namespace)
	}
	if cfg.MongoURI != "mongodb://localhost:27017" {
		t.Errorf("unexpected MongoURI: %q", cfg.MongoURI)
	}
}

func TestLoadConfig_ExplicitValues(t *testing.T) {
	t.Setenv("KFLOW_MONGO_URI", "mongodb://prod:27017")
	t.Setenv("KFLOW_MONGO_DB", "mydb")
	t.Setenv("KFLOW_NAMESPACE", "production")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MongoDB != "mydb" {
		t.Errorf("expected MongoDB 'mydb', got %q", cfg.MongoDB)
	}
	if cfg.Namespace != "production" {
		t.Errorf("expected Namespace 'production', got %q", cfg.Namespace)
	}
}
