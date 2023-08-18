package helpers

import (
	"github.com/nats-io/nats.go"
	auto "github.com/vedadiyan/goal/pkg/config/auto"
	"github.com/vedadiyan/goal/pkg/db/postgres"
	"github.com/vedadiyan/goal/pkg/di"
	"github.com/vedadiyan/goal/pkg/insight"
)

func AddNats(configName string) {
	init := false
	nats := auto.New(configName, true, func(value string) {
		if !init {
			init = true
			di.AddSinletonWithName(configName, func() (*nats.Conn, error) {
				return nats.Connect(value)
			})
			return
		}
		di.RefreshSinletonWithName(configName, func(current *nats.Conn) (*nats.Conn, error) {
			current.Drain()
			return nats.Connect(value)
		})
	})
	auto.Register(nats)
}

func AddPostgres(configName string) {
	init := false
	nats := auto.New(configName, true, func(value string) {
		if !init {
			init = true
			di.AddSinletonWithName(configName, func() (*postgres.Pool, error) {
				return postgres.New(value, 100, 10)
			})
			return
		}
		di.RefreshSinletonWithName(configName, func(current *postgres.Pool) (*postgres.Pool, error) {
			current.Close()
			return postgres.New(value, 100, 10)
		})
	})
	auto.Register(nats)
}

func AddInfluxDb(configName string, bucket string) {
	influxDb := auto.New(configName, false, func(value auto.KeyValue) {
		dsn, err := value.GetStringValue("dsn")
		if err != nil {
			panic(err)
		}
		authToken, err := value.GetStringValue("auth_token")
		if err != nil {
			panic(err)
		}
		org, err := value.GetStringValue("org")
		if err != nil {
			panic(err)
		}
		insight.UseInfluxDbWithFailover(dsn, authToken, org, bucket, "logs.txt")
	})
	auto.Register(influxDb)
}
