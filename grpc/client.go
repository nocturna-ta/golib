package grpc

import (
	"github.com/newrelic/go-agent/v3/integrations/nrgrpc"
	"github.com/nocturna-ta/golib/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type ClientOptions struct {
	Address                  string
	UnaryInterceptors        []grpc.UnaryClientInterceptor
	StreamClientInterceptors []grpc.StreamClientInterceptor
	CertFile                 string
}

type Client struct {
	conn    *grpc.ClientConn
	address string
}

func NewClient(clientOptions *ClientOptions) *Client {
	unaryClientInterceptors := append([]grpc.UnaryClientInterceptor{
		RequestContextClientInterceptor(),
		nrgrpc.UnaryClientInterceptor,
	}, clientOptions.UnaryInterceptors...)

	streamClientInterceptors := append([]grpc.StreamClientInterceptor{
		nrgrpc.StreamClientInterceptor,
	}, clientOptions.StreamClientInterceptors...)

	var opts []grpc.DialOption

	if clientOptions.CertFile != "" {
		creds, err := credentials.NewClientTLSFromFile(clientOptions.CertFile, "")

		if err != nil {
			log.Fatalf("Error while loading CA trust certificate: %v\n", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		creds := grpc.WithTransportCredentials(insecure.NewCredentials())
		opts = append(opts, creds)
	}

	opts = append(opts, grpc.WithChainUnaryInterceptor(unaryClientInterceptors...), grpc.WithChainStreamInterceptor(streamClientInterceptors...))

	conn, err := grpc.Dial(clientOptions.Address, opts...)
	if err != nil {
		log.Fatalf("Did not connect: %v", err)
	}

	return &Client{
		conn:    conn,
		address: clientOptions.Address,
	}
}

func (c *Client) GetConn() *grpc.ClientConn {
	return c.conn
}

func (c *Client) Close() error {
	return c.conn.Close()
}
