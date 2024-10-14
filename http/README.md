# Http Client

Standardized http client call.
It's already use go generic, so you can get response struct as you wish.

## Usage

```go
import (
	"context"
	"github.com/scprimesolution/golib/http"
	"github.com/scprimesolution/golib/log"
	"time"
)

type UserGithub struct {
	Id   int    `json:"id"`
	Url  string `json:"url"`
	Name string `json:"name"`
}

func main() {
	// defining your client
	client := http.NewHttpClient(&http.Options{
		BaseUrl: "https://api.github.com",
		Timeout: 10 * time.Second,
	})

	// build your request, set UserGithub as expected response
	req := http.NewGETRequest[*UserGithub]("/users/gofiber", &UserGithub{}, client)

resp, err := req.WithContext(context.WithValue(context.Background(), libCtx.RequestContextKey, libCtx.RequestContext{})).Execute()
	if err != nil {
		log.Fatal("failed to get response")
	}
	
	// you will get result as type of UserGithub
	result := resp.ResponseData
}
```

Notes:
This is only simple example, you need to make better approach structure for production.