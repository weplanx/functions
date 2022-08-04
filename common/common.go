package common

import (
	"go.mongodb.org/mongo-driver/mongo"
)

type Inject struct {
	Values *Values
	Mongo  *mongo.Client
	Db     *mongo.Database
}

type Values struct {
	// 监听地址
	Address string `env:"address" envDefault:":9000"`
	// 数据库
	Database Database `envPrefix:"DATABASE_"`
}

type Database struct {
	Uri    string `env:"URI"`
	DbName string `env:"DBNAME"`
}
