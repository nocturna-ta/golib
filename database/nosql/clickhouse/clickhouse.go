package clickhouse

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/jmoiron/sqlx"
	"github.com/nocturna-ta/golib/log"
	"github.com/nocturna-ta/golib/tracing"
	"time"
)

type Client interface {
	GetDB() *sqlx.DB
	GetConn() driver.Conn
	Close() error
	Ping(ctx context.Context) error
	Exec(ctx context.Context, query string, args ...any) error
	Select(ctx context.Context, dest any, query string, args ...any) error
	Get(ctx context.Context, dest any, query string, args ...any) error
	AsyncInsert(ctx context.Context, query string, wait bool, args ...any) error
	PrepareBatch(ctx context.Context, query string) (*Batch, error)
	GetServerInfo(ctx context.Context) (*ServerInfo, error)
	Stats() driver.Stats
}

type client struct {
	db       *sqlx.DB
	conn     driver.Conn
	cfg      Config
	isClosed bool
}

type Config struct {
	Addrs              []string           `json:"addrs" mapstructure:"addrs" yaml:"Addrs"`
	Auth               Auth               `json:"auth" mapstructure:"auth" yaml:"Auth"`
	Database           string             `json:"database" mapstructure:"database" yaml:"Database"`
	DialTimeout        time.Duration      `json:"dial_timeout" mapstructure:"dial_timeout" yaml:"DialTimeout"`
	MaxOpenConns       int                `json:"max_open_conns" mapstructure:"max_open_conns" yaml:"MaxOpenConns"`
	MaxIdleConns       int                `json:"max_idle_conns" mapstructure:"max_idle_conns" yaml:"MaxIdleConns"`
	ConnMaxLifetime    time.Duration      `json:"conn_max_lifetime" mapstructure:"conn_max_lifetime" yaml:"ConnMaxLifetime"`
	TLS                *TLSConfig         `json:"tls" mapstructure:"tls" yaml:"TLS"`
	BlockBufferSize    uint8              `json:"block_buffer_size" mapstructure:"block_buffer_size" yaml:"BlockBufferSize"`
	MaxCompressionSize uint64             `json:"max_compression_size" mapstructure:"max_compression_size" yaml:"MaxCompressionSize"`
	AsyncInsert        bool               `json:"async_insert" mapstructure:"async_insert" yaml:"AsyncInsert"`
	AsyncInsertOptions AsyncInsertOptions `json:"async_insert_options" mapstructure:"async_insert_options" yaml:"AsyncInsertOptions"`
	Debug              bool               `json:"debug" mapstructure:"debug" yaml:"Debug"`
}

type Auth struct {
	Database string `json:"database" mapstructure:"database" yaml:"Database"`
	Username string `json:"username" mapstructure:"username" yaml:"Username"`
	Password string `json:"password" mapstructure:"password" yaml:"Password"`
}

type TLSConfig struct {
	Enable             bool   `json:"enable" mapstructure:"enable" yaml:"Enable"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify" mapstructure:"insecure_skip_verify" yaml:"InsecureSkipVerify"`
	CertFile           string `json:"cert_file" mapstructure:"cert_file" yaml:"CertFile"`
	KeyFile            string `json:"key_file" mapstructure:"key_file" yaml:"KeyFile"`
	CAFile             string `json:"ca_file" mapstructure:"ca_file" yaml:"CAFile"`
}

type AsyncInsertOptions struct {
	MaxBatchSize int           `json:"max_batch_size" mapstructure:"max_batch_size" yaml:"MaxBatchSize"`
	MaxDelay     time.Duration `json:"max_delay" mapstructure:"max_delay" yaml:"MaxDelay"`
}

func New(cfg *Config) (Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	c := &client{cfg: *cfg}

	if err := c.connect(); err != nil {
		return nil, err
	}

	return c, nil
}

func NewFromDSN(dsn string) (Client, error) {
	db, err := sqlx.Open("clickhouse", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open clickhouse connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping clickhouse: %w", err)
	}

	return &client{
		db: db,
	}, nil
}

func (c *client) connect() error {
	options := &clickhouse.Options{
		Addr: c.cfg.Addrs,
		Auth: clickhouse.Auth{
			Database: c.cfg.Auth.Database,
			Username: c.cfg.Auth.Username,
			Password: c.cfg.Auth.Password,
		},
		Debug: c.cfg.Debug,
		Debugf: func(format string, v ...any) {
			log.Debugf("[ClickHouse]"+format, v...)
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		DialTimeout:     c.cfg.DialTimeout,
		MaxOpenConns:    c.cfg.MaxOpenConns,
		MaxIdleConns:    c.cfg.MaxIdleConns,
		ConnMaxLifetime: c.cfg.ConnMaxLifetime,
		BlockBufferSize: c.cfg.BlockBufferSize,
	}

	if c.cfg.TLS != nil && c.cfg.TLS.Enable {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: c.cfg.TLS.InsecureSkipVerify,
		}
		options.TLS = tlsConfig
	}

	if c.cfg.AsyncInsert {
		options.Settings["async_insert"] = 1
		options.Settings["wait_for_async_insert"] = 1
	}

	if c.cfg.MaxOpenConns > 0 {
		c.db.SetMaxOpenConns(c.cfg.MaxOpenConns)
	}
	if c.cfg.MaxIdleConns > 0 {
		c.db.SetMaxIdleConns(c.cfg.MaxIdleConns)
	}
	if c.cfg.ConnMaxLifetime > 0 {
		c.db.SetConnMaxLifetime(c.cfg.ConnMaxLifetime)
	}

	conn, err := clickhouse.Open(options)
	if err != nil {
		return fmt.Errorf("failed to open clickhouse connection: %w", err)
	}

	if err := conn.Ping(context.Background()); err != nil {
		return fmt.Errorf("failed to ping clickhouse: %w", err)
	}

	c.conn = conn

	c.db = sqlx.NewDb(clickhouse.OpenDB(options), "clickhouse")

	return nil
}

func (c *client) GetDB() *sqlx.DB {
	return c.db
}

func (c *client) GetConn() driver.Conn {
	return c.conn
}

func (c *client) Close() error {
	if c.isClosed {
		return nil
	}

	c.isClosed = true

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return err
		}
	}

	if c.db != nil {
		if err := c.db.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (c *client) Ping(ctx context.Context) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "ClickHouse.Ping")
	defer span.End()

	if c.conn != nil {
		return c.conn.Ping(ctx)
	}

	if c.db != nil {
		return c.db.PingContext(ctx)
	}

	return fmt.Errorf("no connection or database available to ping")
}

func (c *client) Exec(ctx context.Context, query string, args ...any) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "ClickHouse.Exec")
	defer span.End()

	if c.conn != nil {
		return c.conn.Exec(ctx, query, args...)
	}

	_, err := c.db.ExecContext(ctx, query, args...)

	return err
}

func (c *client) Select(ctx context.Context, dest any, query string, args ...any) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "ClickHouse.Select")
	defer span.End()

	if c.conn != nil {
		return c.conn.Select(ctx, dest, query, args...)
	}

	return c.db.SelectContext(ctx, dest, query, args...)
}

func (c *client) Get(ctx context.Context, dest any, query string, args ...any) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "ClickHouse.Get")
	defer span.End()

	return c.db.GetContext(ctx, dest, query, args...)
}

func (c *client) AsyncInsert(ctx context.Context, query string, wait bool, args ...any) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "ClickHouse.AsyncInsert")
	defer span.End()

	if !c.cfg.AsyncInsert {
		return fmt.Errorf("async insert is not enabled")
	}

	options := clickhouse.Settings{
		"async_insert": 1,
	}

	if wait {
		options["wait_for_async_insert"] = 1
	} else {
		options["wait_for_async_insert"] = 0
	}

	ctx = clickhouse.Context(ctx, clickhouse.WithSettings(options))

	return c.conn.Exec(ctx, query, args...)
}

type Batch struct {
	batch driver.Batch
}

func (c *client) PrepareBatch(ctx context.Context, query string) (*Batch, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ClickHouse.PrepareBatch")
	defer span.End()

	batch, err := c.conn.PrepareBatch(ctx, query)
	if err != nil {
		return nil, err
	}

	return &Batch{batch: batch}, nil
}

func (b *Batch) Append(values ...any) error {
	return b.batch.Append(values...)
}

func (b *Batch) Send() error {
	return b.batch.Send()
}

func (b *Batch) Abort() error {
	return b.batch.Abort()
}

func BuildDSN(cfg *Config) string {
	params := make(map[string]string)

	params["username"] = cfg.Auth.Username
	params["password"] = cfg.Auth.Password
	params["database"] = cfg.Database

	if cfg.Debug {
		params["debug"] = "true"
	}

	if cfg.DialTimeout > 0 {
		params["dial_timeout"] = fmt.Sprintf("%d", int(cfg.DialTimeout.Seconds()))
	}

	if cfg.TLS != nil && cfg.TLS.Enable {
		params["secure"] = "true"
		if cfg.TLS.InsecureSkipVerify {
			params["skip_verify"] = "true"
		}
	}

	dsn := "clickhouse://"
	for i, addr := range cfg.Addrs {
		if i > 0 {
			dsn += ","
		}
		dsn += addr
	}

	dsn += "/?"
	first := true
	for k, v := range params {
		if !first {
			dsn += "&"
		}
		dsn += fmt.Sprintf("%s=%s", k, v)
		first = false
	}

	return dsn
}

func (c *client) GetServerInfo(ctx context.Context) (*ServerInfo, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ClickHouse.GetServerInfo")
	defer span.End()

	info := &ServerInfo{}

	query := `
		SELECT 
			version() as version,
			currentDatabase() as current_database,
			hostName() as hostname,
			timezone() as timezone,
			uptime() as uptime
	`

	err := c.Get(ctx, info, query)
	if err != nil {
		return nil, err
	}

	return info, nil
}

type ServerInfo struct {
	Version         string `db:"version"`
	CurrentDatabase string `db:"current_database"`
	Hostname        string `db:"hostname"`
	Timezone        string `db:"timezone"`
	Uptime          uint64 `db:"uptime"`
}

func (c *client) Stats() driver.Stats {
	if c.conn != nil {
		return c.conn.Stats()
	}
	return driver.Stats{}
}

type BatchManager struct {
	client Client
}

func NewBatchManager(client Client) *BatchManager {
	return &BatchManager{client: client}
}

func (bm *BatchManager) ExecuteBatch(ctx context.Context, query string, fn func(batch *Batch) error) error {
	batch, err := bm.client.PrepareBatch(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare batch: %w", err)
	}

	if err := fn(batch); err != nil {
		if abortErr := batch.Abort(); abortErr != nil {
			log.WithFields(log.Fields{
				"error":       err,
				"abort_error": abortErr,
			}).ErrorWithCtx(ctx, "Failed to abort batch after error")
		}
		return err
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("failed to send batch: %w", err)
	}

	return nil
}
