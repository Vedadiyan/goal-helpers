package helpers

import (
	"time"

	"github.com/nats-io/nats.go"
	pb "github.com/vedadiyan/goal-helpers/pkg/helpers/pb"
	codecs "github.com/vedadiyan/goal/pkg/bus/nats"
	"github.com/vedadiyan/goal/pkg/di"
)

var codec codecs.CompressedProtoConn

func GetAuthHeaders(connName string, namespace string) (map[string]string, error) {
	c, err := di.ResolveWithName[*nats.Conn](connName, nil)
	if err != nil {
		return nil, err
	}
	conn := *c
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

func SendAuthHeaders(connName string, reply string, authHeaders map[string]string) error {
	c, err := di.ResolveWithName[*nats.Conn](connName, nil)
	if err != nil {
		return err
	}
	conn := *c
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
