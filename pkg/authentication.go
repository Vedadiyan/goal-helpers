package helpers

import (
	"time"

	"github.com/nats-io/nats.go"
	pb "github.com/vedadiyan/goal-helpers/pkg/helpers/pb"
	codecs "github.com/vedadiyan/goal/pkg/bus/nats"
)

var codec codecs.CompressedProtoConn

func GetAuthHeaders(conn *nats.Conn, namespace string) (map[string]string, error) {
	msg, err := conn.Request(namespace, nil, time.Second*30)
	if err != nil {
		return nil, err
	}
	var webHeaders pb.WebHeaders
	err = codec.Decode(namespace, msg.Data, &webHeaders)
	if err != nil {
		return nil, err
	}
	return webHeaders.WebHeaders, nil
}

func SendAuthHeaders(conn *nats.Conn, reply string, authHeaders map[string]string) error {
	webHeaders := pb.WebHeaders{}
	webHeaders.WebHeaders = authHeaders
	bytes, err := codec.Encode(reply, &webHeaders)
	if err != nil {
		return err
	}
	err = conn.Publish(reply, bytes)
	if err != nil {
		return err
	}
	return nil
}
