package helpers

import (
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	pb "github.com/vedadiyan/goal-helpers/pkg/helpers/pb"
	codecs "github.com/vedadiyan/goal/pkg/bus/nats"
	"github.com/vedadiyan/goal/pkg/cache"
	"github.com/vedadiyan/goal/pkg/di"
)

var codec codecs.CompressedProtoConn

func GetAuthHeaders(connName string, namespace string, cacheInterval int) (map[string]string, error) {
	var webHeadersMap map[string]string
	var err error
	if cacheInterval != -1 {
		webHeadersMap, err = cache.Get[map[string]string](fmt.Sprintf("%s.%s", connName, namespace))
		if err != nil && !errors.Is(err, cache.KEY_NOT_FOUND) {
			return nil, err
		}
	}
	if webHeadersMap == nil {
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
		webHeadersMap = webHeaders.WebHeaders
		if cacheInterval != -1 {
			cache.SetWithTTL(fmt.Sprintf("%s.%s", connName, namespace), webHeadersMap, time.Second*time.Duration(cacheInterval))
		}
	}
	return webHeadersMap, nil
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
