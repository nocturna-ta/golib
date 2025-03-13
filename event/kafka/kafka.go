package kafka

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/nocturna-ta/golib/event"
	"github.com/nocturna-ta/golib/log"

	"os"
)

func init() {
	event.RegisterConsumer("kafka", NewKafkaConsumer)
	event.RegisterPublisher("kafka", NewKafkaPublisher)
}

func createTlsConfig(certFile, keyFile, caFile string) (t *tls.Config) {
	if certFile != "" && keyFile != "" && caFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			log.Fatal(err)
		}

		caCert, err := os.ReadFile(caFile)
		if err != nil {
			log.Fatal(err)
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		t = &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      caCertPool,
		}
	}
	return t
}
