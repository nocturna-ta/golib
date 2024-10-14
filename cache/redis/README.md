# Cache - Redis

## Usage

```go
import (
    "context"
    "github.com/scprimesolution/golib/cache"
    _ "github.com/scprimesolution/golib/cache/redis"
    "github.com/scprimesolution/golib/log"
)

func main() {
    redisCache, err := cache.New("redis://:@localhost:6379/your-namespace:")
    if err != nil {
        log.Fatal("failed to connect")
    }
    err = redisCache.Set(context.Background(), "test", "your value", 0)
    if err != nil {
        log.Fatal("failed set redis")
    }
}
```

Notes:
This is only simple example, you need to make better approach structure for production.