package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
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

	fmt.Printf("🔄 Authenticating OCI registry %s as %s/***** ...\n", c.Registry, c.Username)

	fmt.Println("🔄 Validating credentials ...")

	endpointStr := c.Registry

	// in case of docker hub - endpoint set to https://auth.docker.io/token
	// else it is not validating correctly - issue SDP-9313
	if isDockerHubRegistry(c.Registry) {
		endpointStr = "https://auth.docker.io/token"
	} else {
		if !strings.HasPrefix(endpointStr, "http://") && !strings.HasPrefix(endpointStr, "https://") {
			// default is to assume https
			endpointStr = "https://" + endpointStr
		}
		endpointStr = strings.TrimSuffix(endpointStr, "/") + "/v2/"
	}

	pingClient := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, endpointStr, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.Username, c.Password)

	resp, err := pingClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized {
		fmt.Println("❌ Invalid credentials")
		return fmt.Errorf("supplied credentials failed to autenticate to %s", endpointStr)
	} else if resp.StatusCode/100 != 2 {
		fmt.Printf("❌ Unexpected error\nHTTP/%d %s\n", resp.StatusCode, resp.Status)
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to authenticate to remote registry\nHTTP/%d %s\n%s", resp.StatusCode, resp.Status, string(body))
	}

	fmt.Println("✅ Credentials validated")

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
		fmt.Println("🔄 Creating ~/.docker/config.json ...")
		bytes = []byte("{}")
	} else {
		fmt.Println("🔄 Merging with existing ~/.docker/config.json ...")
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

	// additionally add default docker registry url to auths
	// to support dockerhub repo. the following additional
	// entry will fix the issue SDP-9313
	if isDockerHubRegistry(c.Registry) {
		auths["https://index.docker.io/v1/"] = auth
	} else {
		auths[c.Registry] = auth
	}

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

	if err := os.WriteFile(configPath, updated, 0644); err != nil {
		return err
	}

	fmt.Println("✅ ~/.docker/config.json updated")

	return nil
}

func isDockerHubRegistry(registry string) bool {
	if !strings.HasPrefix(registry, "https://") {
		// url.parse will fail without  a schema http /https
		registry = "https://" + registry
	}
	parsed, err := url.Parse(registry)
	if err != nil {
		fmt.Println("Error parsing resistry : " + err.Error())
		return false
	}
	return (parsed.Host == "index.docker.io" || parsed.Host == "docker.io")
}
