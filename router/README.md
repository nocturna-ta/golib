# Http Router

Based on gofiber.

## Usage
```go
import (
	"context"
	"github.com/scprimesolution/golib/log"
	"github.com/scprimesolution/golib/response/rest"
	"github.com/scprimesolution/golib/router"
	"os"
	"time"
)

func main() {
	// setting up your http server
	myRouter := router.New(&router.Options{
		Prefix:         "/v1",
		Port:           8900,
		RequestTimeout: 5 * time.Second,
	})

	// directly call handler function
	myRouter.GET("/ping", func(ctx context.Context, req *router.Request) (*rest.JSONResponse, error) {
		return rest.NewJSONResponse().SetData("ok"), nil
	}, router.MustAuthorized(false))

	// sample using attachment method to return file
	myRouter.ATTACHMENT("/file", func(ctx context.Context, req *router.Request) (*rest.AttachmentResponse, error) {
		f, err := os.Open("file.txt")
		if err != nil {
			return nil, err
		}
		return &rest.AttachmentResponse{
			File:        f,
			FileName:    f.Name(),
			ContentType: "txt",
		}, nil
	}, router.MustAuthorized(false))

	// using Group to grouping handler
	myRouter.Group("/test", func(r *router.FastRouter) {
		// using showInt handler
		r.GET("/show-int", showInt())

		// using log middleware above showInt handler
		r.GET("/show-int-with-middleware-log", logMiddleware(showInt()))
	})

	// starting up server
	err := myRouter.StartServe()
	if err != nil {
		log.Fatal("failed to start server")
	}
}

func showInt() router.Handler[rest.JSONResponse] {  
	return func(ctx context.Context, req *router.Request) (*rest.JSONResponse, error) {
		return rest.NewJSONResponse().SetData(1), nil
	}
}

// act as middleware
func logMiddleware(handler router.Handler[rest.JSONResponse]) router.Handler[rest.JSONResponse] {
	return func(ctx context.Context, req *router.Request) (*rest.JSONResponse, error) {
		log.Info("this is log")
		return handler(ctx, req)
	}
}

```
Notes:
This is only simple example, you need to make better approach structure for production.