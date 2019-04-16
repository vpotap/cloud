package util

import (
	"encoding/json"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/cache"
	_ "github.com/astaxie/beego/cache/redis"
	"github.com/astaxie/beego/logs"
	"github.com/garyburd/redigo/redis"
)

// 2019-01-20 redis写入数据
func RedisCacheClient(key string) (cache.Cache, error) {
	return cache.NewCache("redis", `{"conn":"`+beego.AppConfig.String("redis")+`", "key":"`+key+`_", "dbNum":"`+beego.AppConfig.String("redis.dbNum")+`"}`)
}

// 2019-01-19 08:51
// redis数据转换到对象中
func RedisObj2Obj(r interface{}, o interface{}) bool {
	if r != nil {
		redisStr, err := redis.String(r, nil)
		if err == nil {
			json.Unmarshal([]byte(redisStr), &o)
			return true
		}
		logs.Error("转换redis数据出错", err)
	}
	return false
}
