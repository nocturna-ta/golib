package elastic

import (
	elasticV7 "github.com/elastic/go-elasticsearch/v7"
	elasticV8 "github.com/elastic/go-elasticsearch/v8"
	"golib/log"
)

type Client interface {
	GetClientV7() *elasticV7.Client
	GetClientV8() *elasticV8.Client
}

type (
	client struct {
		elasticV7Client *elasticV7.Client
		elasticV8Client *elasticV8.Client
	}

	Config struct {
		Addresses []string // A list of Elasticsearch nodes to use.
		Username  string   // Username for HTTP Basic Authentication.
		Password  string   // Password for HTTP Basic Authentication.

		// PEM-encoded certificate authorities.
		// When set, an empty certificate pool will be created, and the certificates will be appended to it.
		// The option is only valid when the transport is not specified, or when it's http.Transport.
		CACert []byte

		CloudID                string // Endpoint for the Elastic Service (https://elastic.co/cloud).
		APIKey                 string // Base64-encoded token for authorization; if set, overrides username/password and service token.
		ServiceToken           string // Service token for authorization; if set, overrides username/password.
		CertificateFingerprint string // SHA256 hex fingerprint given by Elasticsearch on first launch.

		DisableRetry bool // Default: false.
		MaxRetries   int  // Default: 3.
	}

	Version string
)

var (
	Version7 Version = "v7"
	Version8 Version = "v8"
)

func New(cfg *Config, version Version) Client {
	c := &client{}
	switch version {
	case Version7:
		c.elasticV7Client = newV7(cfg)
	case Version8:
		c.elasticV8Client = newV8(cfg)
	default:
		log.Fatal("Unsupported elasticsearch version")
	}
	return c
}

func newV7(cfg *Config) *elasticV7.Client {
	opt := elasticV7.Config{
		Addresses:              cfg.Addresses,
		Username:               cfg.Username,
		Password:               cfg.Password,
		CACert:                 cfg.CACert,
		CloudID:                cfg.CloudID,
		APIKey:                 cfg.APIKey,
		ServiceToken:           cfg.ServiceToken,
		CertificateFingerprint: cfg.CertificateFingerprint,
		DisableRetry:           cfg.DisableRetry,
		MaxRetries:             cfg.MaxRetries,
		DiscoverNodesOnStart:   true,
	}

	c, err := elasticV7.NewClient(opt)
	if err != nil {
		log.Fatalf("[elastic] Failed to connect client-v7: %+v", err)
	}

	res, err := c.Info()
	if err != nil {
		log.Fatalf("[elastic] Failed to ping client: %+v", err)
	}
	if res.IsError() {
		log.Fatalf("[elastic] Failed response from elastic server: %+v", err)
	}

	return c
}

func newV8(cfg *Config) *elasticV8.Client {
	opt := elasticV8.Config{
		Addresses:              cfg.Addresses,
		Username:               cfg.Username,
		Password:               cfg.Password,
		CACert:                 cfg.CACert,
		CloudID:                cfg.CloudID,
		APIKey:                 cfg.APIKey,
		ServiceToken:           cfg.ServiceToken,
		CertificateFingerprint: cfg.CertificateFingerprint,
		DisableRetry:           cfg.DisableRetry,
		MaxRetries:             cfg.MaxRetries,
		DiscoverNodesOnStart:   true,
	}

	c, err := elasticV8.NewClient(opt)
	if err != nil {
		log.Fatalf("[elastic] Failed to connect client-v8: %+v", err)
	}

	res, err := c.Info()
	if err != nil {
		log.Fatalf("[elastic] Failed to ping client: %+v", err)
	}
	if res.IsError() {
		log.Fatalf("[elastic] Failed response from elastic server: %+v", err)
	}
	return c
}

func (c *client) GetClientV7() *elasticV7.Client {
	return c.elasticV7Client
}

func (c *client) GetClientV8() *elasticV8.Client {
	return c.elasticV8Client
}
