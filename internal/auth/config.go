package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	// Registry The registry server such as `docker.example.com`.
	Registry string
	// Username The username to authenticate with.
	Username string
	// Password The password to authenticate with.
	Password string
}

func (c Config) Authenticate(ctx context.Context) error {
	if c.Registry == "" {
		return fmt.Errorf("registry must be specified")
	}

	fmt.Printf("Adding authentication %s/***** for %s to ~/.docker/config.json\n", c.Username, c.Registry)

	homePath := os.Getenv("HOME")
	kubePath := filepath.Join(homePath, ".docker")
	if err := os.MkdirAll(kubePath, os.ModePerm); err != nil {
		return err
	}
	configPath := filepath.Join(kubePath, "config.json")

	bytes, err := os.ReadFile(configPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("could not read %s: %w", configPath, err)
	} else if errors.Is(err, os.ErrNotExist) {
		bytes = []byte("{}")
	}

	var config map[string]interface{}

	if err := json.Unmarshal(bytes, &config); err != nil {
		return fmt.Errorf("could not parse %s: %w", configPath, err)
	}

	auth := make(map[string]string)
	auth["auth"] = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", c.Username, c.Password)))

	var auths map[string]interface{}
	if a, found := config["auths"]; found {
		if aa, ok := a.(map[string]interface{}); ok {
			auths = aa
		}
	}
	if auths == nil {
		auths = make(map[string]interface{})
	}

	auths[c.Registry] = auth

	config["auths"] = auths

	var credHelpers map[string]interface{}
	if a, found := config["credHelpers"]; found {
		if aa, ok := a.(map[string]interface{}); ok {
			credHelpers = aa
		}
	}
	if credHelpers == nil {
		credHelpers = make(map[string]interface{})
	}

	delete(credHelpers, c.Registry)

	config["credHelpers"] = credHelpers

	updated, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, updated, 0644)
}
