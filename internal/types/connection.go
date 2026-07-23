package types

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"
)

// Flavor identifies the search engine backend.
type Flavor string

const (
	FlavorElasticsearch Flavor = "elasticsearch"
	FlavorOpenSearch    Flavor = "opensearch"
	FlavorAuto          Flavor = "auto"
)

// DisplayName returns a human-readable engine name.
func (f Flavor) DisplayName() string {
	switch f {
	case FlavorOpenSearch:
		return "OpenSearch"
	case FlavorElasticsearch:
		return "Elasticsearch"
	case FlavorAuto:
		return "Auto"
	default:
		if f == "" {
			return "Auto"
		}
		return string(f)
	}
}

// Short returns a compact badge label (ES / OS / AUTO).
func (f Flavor) Short() string {
	switch f {
	case FlavorOpenSearch:
		return "OS"
	case FlavorElasticsearch:
		return "ES"
	default:
		return "AUTO"
	}
}

// IsKnown reports whether the flavor is a concrete engine (not auto/empty).
func (f Flavor) IsKnown() bool {
	return f == FlavorElasticsearch || f == FlavorOpenSearch
}

// Connection stores Elasticsearch/OpenSearch connection details.
type Connection struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Host        string     `json:"host"`
	Port        int        `json:"port"`
	Username    string     `json:"username"`
	Password    string     `json:"password,omitempty"`     // #nosec G117 -- stored in local user config.
	APIKey      string     `json:"api_key,omitempty"`      // #nosec G117 -- stored in local user config.
	BearerToken string     `json:"bearer_token,omitempty"` // #nosec G117 -- stored in local user config.
	Flavor      Flavor     `json:"flavor,omitempty"`
	Group       string     `json:"group,omitempty"`
	Color       string     `json:"color,omitempty"`
	UseTLS      bool       `json:"use_tls,omitempty"`
	ReadOnly    bool       `json:"read_only,omitempty"`
	TLSConfig   *TLSConfig `json:"tls_config,omitempty"`
	Created     time.Time  `json:"created_at"`
	Updated     time.Time  `json:"updated_at"`
}

// Address returns host:port.
func (c Connection) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// BaseURL returns the HTTP(S) base URL for the connection.
func (c Connection) BaseURL() string {
	scheme := "http"
	if c.UseTLS {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, c.Host, c.Port)
}

// TLSConfig stores TLS/SSL configuration.
type TLSConfig struct {
	CertFile           string `json:"cert_file,omitempty"`
	KeyFile            string `json:"key_file,omitempty"`
	CAFile             string `json:"ca_file,omitempty"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify,omitempty"`
	ServerName         string `json:"server_name,omitempty"`
}

// BuildTLSConfig creates a *tls.Config from the stored TLS parameters.
func (t *TLSConfig) BuildTLSConfig() (*tls.Config, error) {
	cfg := &tls.Config{
		InsecureSkipVerify: t.InsecureSkipVerify, // #nosec G402 -- user-configured
		ServerName:         t.ServerName,
	}

	if t.CertFile != "" && t.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(t.CertFile, t.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS key pair: %w", err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}

	if t.CAFile != "" {
		caCert, err := os.ReadFile(t.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		cfg.RootCAs = pool
	}

	return cfg, nil
}

// ConnectionGroup organizes connections.
type ConnectionGroup struct {
	Name        string  `json:"name"`
	Color       string  `json:"color,omitempty"`
	Connections []int64 `json:"connections"`
	Collapsed   bool    `json:"collapsed,omitempty"`
}

// Favorite stores a favorited index.
type Favorite struct {
	ConnectionID int64     `json:"connection_id"`
	Connection   string    `json:"connection"`
	Index        string    `json:"index"`
	Label        string    `json:"label,omitempty"`
	AddedAt      time.Time `json:"added_at"`
}

// RecentIndex tracks recently accessed indices.
type RecentIndex struct {
	ConnectionID int64     `json:"connection_id"`
	Index        string    `json:"index"`
	AccessedAt   time.Time `json:"accessed_at"`
}
