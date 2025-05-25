# ClickHouse Integration

This package provides ClickHouse database integration for the golib.

## Features

- Connection pooling and management
- Support for batch inserts
- Async insert capabilities
- TLS/SSL support
- Multiple node support (cluster)
- Compression support
- Tracing integration

## Usage

### Basic Connection

```go
import (
    "context"
    "github.com/nocturna-ta/golib/database/nosql/clickhouse"
    "github.com/nocturna-ta/golib/log"
)

func main() {
    // Option 1: Using configuration struct
    cfg := &clickhouse.Config{
        Addrs:    []string{"localhost:9000"},
        Auth: clickhouse.Auth{
            Database: "default",
            Username: "default",
            Password: "",
        },
        Debug: true,
    }
    
    client, err := clickhouse.New(cfg)
    if err != nil {
        log.Fatal("Failed to connect to ClickHouse:", err)
    }
    defer client.Close()
    
    // Option 2: Using DSN
    client, err := clickhouse.NewFromDSN("clickhouse://localhost:9000/default?username=default&password=")
    if err != nil {
        log.Fatal("Failed to connect to ClickHouse:", err)
    }
}
```

### Using with database/sql package

```go
import (
    "github.com/nocturna-ta/golib/database/sql"
    "github.com/nocturna-ta/golib/log"
)

func main() {
    // ClickHouse DSN format for database/sql
    cfg := sql.DBConfig{
        MasterDSN: "clickhouse://default:password@localhost:9000/database?dial_timeout=10s&compress=true",
        SlaveDSN:  "clickhouse://default:password@localhost:9000/database?dial_timeout=10s&compress=true",
        MaxIdleConn: 5,
        MaxConn: 10,
    }
    
    store := sql.New(cfg, sql.DriverClickHouse)
    
    // Use the connection
    db := store.GetMaster()
    
    // Execute query
    _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS events (
            timestamp DateTime,
            event_type String,
            user_id UInt64
        ) ENGINE = MergeTree()
        ORDER BY timestamp
    `)
    if err != nil {
        log.Fatal("Failed to create table:", err)
    }
}
```

### Batch Inserts

```go
func insertBatch(client clickhouse.Client) error {
    ctx := context.Background()
    
    // Prepare batch
    batch, err := client.PrepareBatch(ctx, "INSERT INTO events (timestamp, event_type, user_id)")
    if err != nil {
        return err
    }
    
    // Add data to batch
    for i := 0; i < 1000; i++ {
        err := batch.Append(
            time.Now(),
            "click",
            uint64(i),
        )
        if err != nil {
            batch.Abort()
            return err
        }
    }
    
    // Send batch
    return batch.Send()
}
```

### Async Inserts

```go
func asyncInsert(client clickhouse.Client) error {
    cfg := &clickhouse.Config{
        Addrs:    []string{"localhost:9000"},
        Auth: clickhouse.Auth{
            Database: "default",
            Username: "default",
            Password: "",
        },
        AsyncInsert: true,
        AsyncInsertOptions: &clickhouse.AsyncInsertOptions{
            MaxBatchSize: 1000,
            MaxDelay:     5 * time.Second,
        },
    }
    
    client, err := clickhouse.New(cfg)
    if err != nil {
        return err
    }
    
    // Insert with async mode
    query := "INSERT INTO events (timestamp, event_type, user_id) VALUES (?, ?, ?)"
    
    // wait=true means wait for insert to complete
    err = client.AsyncInsert(context.Background(), query, true, time.Now(), "async_click", uint64(123))
    if err != nil {
        return err
    }
    
    return nil
}
```

### Querying Data

```go
func queryData(client clickhouse.Client) error {
    ctx := context.Background()
    
    // Single row
    var count uint64
    err := client.Get(ctx, &count, "SELECT COUNT(*) FROM events")
    if err != nil {
        return err
    }
    
    // Multiple rows
    type Event struct {
        Timestamp time.Time `db:"timestamp"`
        EventType string    `db:"event_type"`
        UserID    uint64    `db:"user_id"`
    }
    
    var events []Event
    err = client.Select(ctx, &events, "SELECT * FROM events WHERE event_type = ? LIMIT 100", "click")
    if err != nil {
        return err
    }
    
    return nil
}
```

### Cluster Configuration

```go
func clusterConfig() *clickhouse.Config {
    return &clickhouse.Config{
        Addrs: []string{
            "clickhouse-node1:9000",
            "clickhouse-node2:9000",
            "clickhouse-node3:9000",
        },
        Auth: clickhouse.Auth{
            Database: "default",
            Username: "default",
            Password: "password",
        },
        DialTimeout:     10 * time.Second,
        MaxOpenConns:    10,
        MaxIdleConns:    5,
        ConnMaxLifetime: time.Hour,
    }
}
```

### TLS Configuration

```go
func tlsConfig() *clickhouse.Config {
    return &clickhouse.Config{
        Addrs: []string{"secure-clickhouse:9440"},
        Auth: clickhouse.Auth{
            Database: "default",
            Username: "default",
            Password: "password",
        },
        TLS: &clickhouse.TLSConfig{
            Enable:             true,
            InsecureSkipVerify: false,
            CertFile:          "/path/to/cert.pem",
            KeyFile:           "/path/to/key.pem",
            CAFile:            "/path/to/ca.pem",
        },
    }
}
```

### Getting Server Information

```go
func getServerInfo(client clickhouse.Client) error {
    info, err := client.GetServerInfo(context.Background())
    if err != nil {
        return err
    }
    
    log.Printf("ClickHouse Version: %s", info.Version)
    log.Printf("Current Database: %s", info.CurrentDatabase)
    log.Printf("Hostname: %s", info.Hostname)
    log.Printf("Timezone: %s", info.Timezone)
    log.Printf("Uptime: %d seconds", info.Uptime)
    
    return nil
}
```

## DSN Format

The DSN format for ClickHouse is:

```
clickhouse://[username[:password]@][host1[:port1],...[hostN[:portN]]]/[database][?param1=value1&...&paramN=valueN]
```

### Common DSN Parameters:

- `username`, `password` - auth credentials
- `database` - default database
- `dial_timeout` - timeout for establishing connection
- `max_execution_time` - query execution timeout
- `compress` - enable compression
- `secure` - use TLS
- `skip_verify` - skip TLS verification
- `debug` - enable debug logging

## Best Practices

1. **Use Batch Inserts**: For bulk data insertion, always use batch inserts for better performance.

2. **Connection Pooling**: Configure appropriate connection pool settings based on your workload.

3. **Async Inserts**: For high-throughput scenarios where you can tolerate some delay, use async inserts.

4. **Partitioning**: Design your tables with proper partitioning for better query performance.

5. **Compression**: Enable compression for network traffic to reduce bandwidth usage.

6. **Error Handling**: Always handle errors appropriately, especially for batch operations.

## Example Service Integration

```go
type EventService struct {
    clickhouse clickhouse.Client
}

func NewEventService(cfg *clickhouse.Config) (*EventService, error) {
    client, err := clickhouse.New(cfg)
    if err != nil {
        return nil, err
    }
    
    return &EventService{
        clickhouse: client,
    }, nil
}

func (s *EventService) RecordEvent(ctx context.Context, eventType string, userID uint64) error {
    query := `
        INSERT INTO events (timestamp, event_type, user_id) 
        VALUES (?, ?, ?)
    `
    
    return s.clickhouse.Exec(ctx, query, time.Now(), eventType, userID)
}

func (s *EventService) GetEventCount(ctx context.Context, eventType string) (uint64, error) {
    var count uint64
    
    query := `
        SELECT COUNT(*) 
        FROM events 
        WHERE event_type = ?
    `
    
    err := s.clickhouse.Get(ctx, &count, query, eventType)
    return count, err
}

func (s *EventService) Close() error {
    return s.clickhouse.Close()
}
```