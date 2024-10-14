# Redis Distributed Locking

Sometimes, you only want to make sure that request handle concurrency with expected behaviour.
So maybe you need that only 1 request can be fulfilled for 1 time for specific occurent.
This distributed lock will help you a lot.

## Usage

```go
import (
    "context"
    "github.com/scprimesolution/golib/cache/redlock"
    "github.com/scprimesolution/golib/log"
)

func main() {
    // you can also set config max retries for request to acquiring lock
    redLock := redlock.New(&redlock.Config{
        ConnectionUrl:  "redis://:@localhost:6379/your-namespace:",
    })
	
    err := redLock.AcquireLock(context.Background(), "key", 1)
    if err != nil {
        log.Fatal("can not acquire lock")
    }
	
    // Do some code
	
    // you have to release the lock, so other request can access it
    // this option can also be done with defer, so you don't need to worry about releasing the lock 
    // because it already deferred
    err = redLock.ReleaseLock(context.Background(), "key")
    if err != nil {
        log.Fatal("failed release lock")
    }
}
```