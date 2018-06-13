package scripts

import (
	"github.com/go-redis/redis"
)

var EnqueueScript *redis.Script
var DequeueScript *redis.Script
var FinishScript *redis.Script
var RequeueScript *redis.Script

func init() {
	EnqueueScript = redis.NewScript(string(MustAsset("lua/enqueue.lua")))
	DequeueScript = redis.NewScript(string(MustAsset("lua/dequeue.lua")))
	FinishScript = redis.NewScript(string(MustAsset("lua/finish.lua")))
	RequeueScript = redis.NewScript(string(MustAsset("lua/requeue.lua")))
}
