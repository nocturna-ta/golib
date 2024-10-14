package mongodb

import (
	"context"
	"github.com/newrelic/go-agent/v3/integrations/nrmongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"golib/log"
	"time"
)

var (
	defaultConnectTimeout = 10 * time.Second
	defaultPingTimeout    = 2 * time.Second
)

type Client interface {
	GetDatabase() (*mongo.Database, error)
}

type (
	client struct {
		mongoClient *mongo.Client
		cfg         Config
	}

	Config struct {
		URI            string `json:"uri" mapstructure:"uri"`
		DB             string `json:"database" mapstructure:"database"`
		AppName        string `json:"name" mapstructure:"name"`
		ConnectTimeout time.Duration
		PingTimeout    time.Duration
	}
)

func New(cfg *Config) Client {
	c := &client{cfg: *cfg}
	c.connectClient()
	return c
}

func (c *client) GetDatabase() (*mongo.Database, error) {
	return c.mongoClient.Database(c.cfg.DB), nil
}

func (c *client) connectClient() {
	mongoClient, err := c.connect()
	if err != nil {
		log.Fatalf("[Mongo.ConnectClient] Failed to connect: %v", err)
	}
	c.mongoClient = mongoClient
}

func (c *client) connect() (*mongo.Client, error) {
	if c.cfg.ConnectTimeout == 0 {
		c.cfg.ConnectTimeout = defaultConnectTimeout
	}

	if c.cfg.PingTimeout == 0 {
		c.cfg.PingTimeout = defaultPingTimeout
	}

	connectCtx, cancelConnectCtx := context.WithTimeout(context.Background(), c.cfg.ConnectTimeout)
	defer cancelConnectCtx()

	mongoRegistry := bson.NewRegistry()
	mongoRegistry.RegisterTypeEncoder(tUUID, bsoncodec.ValueEncoderFunc(uuidEncodeValue))
	mongoRegistry.RegisterTypeDecoder(tUUID, bsoncodec.ValueDecoderFunc(uuidDecodeValue))
	mongoRegistry.RegisterTypeEncoder(tJSON, bsoncodec.ValueEncoderFunc(jsonEncodeValue))
	mongoRegistry.RegisterTypeDecoder(tJSON, bsoncodec.ValueDecoderFunc(jsonDecodeValue))

	opts := []*options.ClientOptions{
		options.Client().
			SetConnectTimeout(c.cfg.ConnectTimeout).
			ApplyURI(c.cfg.URI).
			SetAppName(c.cfg.AppName).
			SetMonitor(nrmongo.NewCommandMonitor(nil)).
			SetRegistry(mongoRegistry),
	}

	mc, err := mongo.Connect(connectCtx, opts...)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"msg":   "Failed to connect to mongo",
		}).Error("Failed to create mongodb client")
		return nil, err
	}

	pingCtx, cancelPingCtx := context.WithTimeout(context.Background(), c.cfg.PingTimeout)
	defer cancelPingCtx()

	if err = mc.Ping(pingCtx, readpref.Primary()); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"msg":   "Failed to ping mongo",
		}).Error("Failed to establish connection to mongodb")
		return nil, err
	}

	return mc, nil
}
