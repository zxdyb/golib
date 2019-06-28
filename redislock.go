/**
***基于单节点redis 分布式锁
**/
package lockercommon

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"time"
)

type RedisLock struct {
	lockKey string
	value   string
}

//保证原子性（redis是单线程），避免del删除了，其他client获得的lock
var delScript = redis.NewScript(1, `
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
else
	return 0
end`)

func (rdl *RedisLock) Lock(rd redis.Conn, timeout int) error {

	{ //随机数
		b := make([]byte, 16)
		_, err := rand.Read(b)
		if err != nil {
			return err
		}
		rdl.value = base64.StdEncoding.EncodeToString(b)
	}
	lockReply, err := rd.Do("SET", rdl.lockKey, rdl.value, "ex", timeout, "nx")
	if err != nil {
		return errors.New("redis fail")
	}
	if lockReply == "OK" {
		return nil
	} else {
		return errors.New("lock fail")
	}
}

func (rdl *RedisLock) Unlock(rd redis.Conn) error{
	_, err := delScript.Do(rd, rdl.lockKey, rdl.value)
	return err
}

//定义工厂类，转为创建锁对象，工厂类中持有连接池，创建的锁对象中包含有连接对象
//注意这里每个连接对象包含了不同的连接对象，使用完成后要记得调用包装对象的Free方法释放连接
type RedisLockFactory struct {
	redispool *redis.Pool
}

func (rdlft *RedisLockFactory) Init(ip string, port int){
	rdlft.redispool = &redis.Pool{
		MaxIdle:     10,
		IdleTimeout: 300 * time.Second,
		Dial: func() (redis.Conn, error) {
			tcp := fmt.Sprintf("%s:%d", ip, port)
			c, err := redis.Dial("tcp", tcp)
			if err != nil {
				return nil, err
			}
			//fmt.Println("connect redis success!")
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

func (rdlft *RedisLockFactory) CreateRedisLock(key string) *RedisLockWrapper{
	rdd := rdlft.redispool.Get()
	return &RedisLockWrapper{rdd, RedisLock{lockKey: key}}
}

//锁对象包装器，将具体连接对象封装在其中，便于外部直接使用锁对象，而无需关系具体连接
type RedisLockWrapper struct{
	rd redis.Conn
	RedisLock
}
func (rdlwp *RedisLockWrapper) Free() error{
	return rdlwp.rd.Close()
}

func (rdlwp *RedisLockWrapper) LockEx(timeout int) error {
	return rdlwp.Lock(rdlwp.rd, timeout)
}

func (rdlwp *RedisLockWrapper) UnlockEx() error {
	return rdlwp.Unlock(rdlwp.rd)
}


