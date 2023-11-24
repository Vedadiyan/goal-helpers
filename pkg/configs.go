package helpers

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	auto "github.com/vedadiyan/goal/pkg/config/auto"
	"github.com/vedadiyan/goal/pkg/db/postgres"
	"github.com/vedadiyan/goal/pkg/di"
	"github.com/vedadiyan/goal/pkg/insight"
	mng "github.com/vedadiyan/gql/pkg/functions/mongo"
	rds "github.com/vedadiyan/gql/pkg/functions/redis"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func init() {
	mng.RegisterConManager(func(connKey string) (*mongo.Client, error) {
		client, err := di.ResolveWithName[*mongo.Client](connKey, nil)
		return *client, err
	})
	rds.RegisterConManager(func(connKey string) (*redis.Client, error) {
		client, err := di.ResolveWithName[*redis.Client](connKey, nil)
		return *client, err
	})

}

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
	postgres := auto.New(configName, true, func(value string) {
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
	auto.Register(postgres)
}

func AddRedis(configName string) {
	init := false
	mongodb := auto.New(configName, true, func(value string) {
		if !init {
			init = true
			di.AddSinletonWithName(configName, func() (*redis.Client, error) {
				return redis.NewClient(&redis.Options{
					Addr: value,
				}), nil
			})
			return
		}
		di.RefreshSinletonWithName(configName, func(current *redis.Client) (*redis.Client, error) {
			current.Close()
			return redis.NewClient(&redis.Options{
				Addr: value,
			}), nil
		})
	})
	auto.Register(mongodb)
}

func AddMongo(configName string) {
	init := false
	mongodb := auto.New(configName, true, func(value string) {
		if !init {
			init = true
			di.AddSinletonWithName(configName, func() (*mongo.Client, error) {
				return mongo.Connect(context.TODO(), options.Client().ApplyURI(value))
			})
			return
		}
		di.RefreshSinletonWithName(configName, func(current *mongo.Client) (*mongo.Client, error) {
			current.Disconnect(context.TODO())
			return mongo.Connect(context.TODO(), options.Client().ApplyURI(value))
		})
	})
	auto.Register(mongodb)
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
