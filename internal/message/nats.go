package message

import (
	"github.com/nats-io/nats.go"
)

var js nats.JetStreamContext

func Connect(url string) (*nats.Conn, error) {
	return nats.Connect(url)
}

func InitNats(nc *nats.Conn) error {
	var err error

	js, err = nc.JetStream()
	if err != nil {
		return err
	}

	_, err = js.KeyValue("users")
	if err == nats.ErrBucketNotFound {
		_, err = js.CreateKeyValue(&nats.KeyValueConfig{
			Bucket: "users",
		})
		if err != nil {
			return err
		}
	}

	_, err = js.KeyValue("functions")
	if err == nats.ErrBucketNotFound {
		_, err = js.CreateKeyValue(&nats.KeyValueConfig{
			Bucket: "functions",
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func GetJetStream() nats.JetStreamContext {
	return js
}
