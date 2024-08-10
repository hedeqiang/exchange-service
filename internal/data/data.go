package data

import (
	"context"
	"exchange-service/internal/conf"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/rabbitmq/amqp091-go"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewGreeterRepo)

// Data .
type Data struct {
	db       *gorm.DB
	rdb      *redis.Client
	rabbitmq *amqp091.Connection
}

// NewData .
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	log := log.NewHelper(logger)

	// 初始化 MySQL
	db, err := gorm.Open(mysql.Open(c.Database.Source), &gorm.Config{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 初始化 Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:         c.Redis.Addr,
		Password:     c.Redis.Password, // 设置 Redis 密码
		DB:           0,                // 使用默认DB
		ReadTimeout:  c.Redis.ReadTimeout.AsDuration(),
		WriteTimeout: c.Redis.WriteTimeout.AsDuration(),
	})

	// 测试 Redis 连接
	_, err = rdb.Ping(context.Background()).Result()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	// 初始化 RabbitMQ
	rabbitmqConnStr := fmt.Sprintf("amqp://%s:%s@%s/%s",
		c.Rabbitmq.Username,
		c.Rabbitmq.Password,
		c.Rabbitmq.Addr,
		c.Rabbitmq.VirtualHost,
	)
	rabbitmqConn, err := amqp091.Dial(rabbitmqConnStr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	d := &Data{
		db:       db,
		rdb:      rdb,
		rabbitmq: rabbitmqConn,
	}

	cleanup := func() {
		log.Info("closing the data resources")
		sqlDB, _ := db.DB()
		if err := sqlDB.Close(); err != nil {
			log.Error(err)
		}
		if err := rdb.Close(); err != nil {
			log.Error(err)
		}
		if err := rabbitmqConn.Close(); err != nil {
			log.Error(err)
		}
	}

	return d, cleanup, nil
}

// DB 返回 GORM DB 实例
func (d *Data) DB() *gorm.DB {
	return d.db
}

// RDB 返回 Redis 客户端实例
func (d *Data) RDB() *redis.Client {
	return d.rdb
}

// RabbitMQ 返回 RabbitMQ 连接实例
func (d *Data) RabbitMQ() *amqp091.Connection {
	return d.rabbitmq
}
