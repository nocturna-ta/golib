package sql

import (
	"fmt"
	_ "github.com/ClickHouse/clickhouse-go/v2"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/nocturna-ta/golib/log"
	"time"
)

type DBDriver string

const (
	DriverMySQL      DBDriver = "mysql"
	DriverPostgres   DBDriver = "postgres"
	DriverClickHouse DBDriver = "clickhouse"
)

type (
	DBConfig struct {
		SlaveDSN        string `json:"slave_dsn" mapstructure:"slave_dsn"`
		MasterDSN       string `json:"master_dsn" mapstructure:"master_dsn"`
		RetryInterval   int    `json:"retry_interval" mapstructure:"retry_interval"`
		MaxIdleConn     int    `json:"max_idle" mapstructure:"max_idle"`
		MaxConn         int    `json:"max_con" mapstructure:"max_con"`
		ConnMaxLifetime string `json:"conn_max_lifetime" mapstructure:"conn_max_lifetime"`
		// ClickHouse specific settings
		Debug           bool   `json:"debug" mapstructure:"debug"`
		CompressionType string `json:"compression_type" mapstructure:"compression_type"`
	}

	DB struct {
		DBDriver        DBDriver
		DBConnection    *sqlx.DB
		DBString        string
		RetryInterval   int
		MaxIdleConn     int
		MaxConn         int
		ConnMaxLifetime time.Duration
		doneChannel     chan bool
	}

	Store struct {
		Master *DB
		Slave  *DB
	}
)

func New(cfg DBConfig, driver DBDriver) *Store {
	masterDSN := cfg.MasterDSN
	slaveDSN := cfg.SlaveDSN

	var conMaxLifetime time.Duration
	if cfg.ConnMaxLifetime != "" {
		duration, err := time.ParseDuration(cfg.ConnMaxLifetime)
		if err != nil {
			log.Fatal("Invalid ConnMaxLifetime value: " + err.Error())
			return nil
		}

		conMaxLifetime = duration
	}

	Master := &DB{
		DBDriver:        driver,
		DBString:        masterDSN,
		RetryInterval:   cfg.RetryInterval,
		MaxIdleConn:     cfg.MaxIdleConn,
		MaxConn:         cfg.MaxConn,
		ConnMaxLifetime: conMaxLifetime,
		doneChannel:     make(chan bool),
	}
	err := Master.ConnectAndMonitor()
	if err != nil {
		log.Fatal("Could not initiate Master DB connection: " + err.Error())
		return nil
	}

	Slave := &DB{
		DBDriver:        driver,
		DBString:        slaveDSN,
		RetryInterval:   cfg.RetryInterval,
		MaxIdleConn:     cfg.MaxIdleConn,
		MaxConn:         cfg.MaxConn,
		ConnMaxLifetime: conMaxLifetime,
		doneChannel:     make(chan bool),
	}
	err = Slave.ConnectAndMonitor()
	if err != nil {
		log.Fatal("Could not initiate Slave DB connection: " + err.Error())
		return nil
	}

	return &Store{
		Master: Master,
		Slave:  Slave,
	}
}

func (s *Store) GetMaster() *sqlx.DB {
	return s.Master.DBConnection
}

func (s *Store) GetSlave() *sqlx.DB {
	return s.Slave.DBConnection
}

func (d *DB) ConnectAndMonitor() error {
	err := d.Connect()

	if err != nil {
		log.Errorf("Can not connected to database %s", d.DBString)
		return err
	}

	ticker := time.NewTicker(time.Duration(d.RetryInterval) * time.Second)
	go func() error {
		for {
			select {
			case <-ticker.C:
				if d.DBConnection == nil {
					return d.Connect()
				} else {
					err := d.DBConnection.Ping()
					if err != nil {
						log.Error("[Error]: DB reconnect error", err.Error())
						return err
					}
				}
			case <-d.doneChannel:
				return nil
			}
		}
	}()
	return nil
}

func (d *DB) Connect() error {
	db, err := sqlx.Open(string(d.DBDriver), d.DBString)
	if err != nil {
		return fmt.Errorf("failed to open DB connection: %w", err)
	}

	db.SetMaxOpenConns(d.MaxConn)
	db.SetMaxIdleConns(d.MaxIdleConn)

	if d.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(d.ConnMaxLifetime)
	}

	d.DBConnection = db

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping DB: %w", err)
	}

	return nil
}
