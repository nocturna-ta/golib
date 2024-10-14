# Go gRPC

## Installation

using homebrew
```shell
brew install protoc-gen-go
brew install protoc-gen-go-grpc
```

other OS you can googling it first.


## Setting up proto file
example:
```protobuf
syntax = "proto3";

package greet;

option go_package = "github.com/scprimesolution/golib/proto";

message GreetRequest {
  string first_name = 1;
}

message GreetResponse {
  string result = 1;
}

service GreetService {
  rpc Greet(GreetRequest) returns (GreetResponse);
}
```

Learned protobuf syntax by yourself.

### Generating Go and gRPC from proto file
command
```shell
protoc -I ${PROTO_DIR} --go_opt=module=${PACKAGE} --go_out=. --go-grpc_opt=module=${PACKAGE} --go-grpc_out=. $@/${PROTO_DIR}/*.proto
```

You will see it generating 2 files that containg definition of go and go-grpc of the proto file.

# Usage

## gRPC Server
```go
import (
    "context"
    "github.com/scprimesolution/golib/grpc"
    "github.com/scprimesolution/golib/log"
    "github.com/scprimesolution/golib/proto"
)

type Server struct {
    proto.GreetServiceServer
}

func main() {
    log.SetFormatter("json")
    
    // create new server
    srv := grpc.NewServer(&grpc.ServerOptions{
        Port: 50051,
    })
    
    // register service
    srv.Register(proto.RegisterGreetServiceServer, &Server{})
    
    // guarantee that server must start
    srv.MustStart()
}

// service implementation
func (s *Server) Greet(ctx context.Context, req *proto.GreetRequest) (*proto.GreetResponse, error) {
    log.Printf("Greet was invoked with %v\n", req)
    return &proto.GreetResponse{Result: "Hello " + req.FirstName}, nil
}
```

## gRPC Client
```go
import (
	"context"
	"github.com/scprimesolution/golib/grpc"
	"github.com/scprimesolution/golib/log"
	"github.com/scprimesolution/golib/proto"
)

var addr string = "localhost:50051"

func main() {
	log.SetFormatter("json")

	// create new client connect to server
	cl := grpc.NewClient(&grpc.ClientOptions{
		Address: addr,
	})

	// create new service client
	svc := proto.NewGreetServiceClient(cl.GetConn())

	// calling via grpc
	r, err := svc.Greet(context.Background(), &proto.GreetRequest{FirstName: "Test"})
	if err != nil {
		log.Fatal("failed to call grpc")
	}

	// getting result
	log.Printf("Greeting: %s\n", r.Result)
}

```