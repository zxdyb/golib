package lockercommon

import (
    "fmt"
    "testing"
    "time"

    "github.com/garyburd/redigo/redis"
)

func TestLockWrapper(t *testing.T){
    rdft := RedisLockFactory{}
    rdft.Init("47.91.178.4", 9379)
    key := "yyy"
    go func() {
        Alock := rdft.CreateRedisLock(key)
        err := Alock.LockEx(5)
        time.Sleep(7 * time.Second)
        fmt.Println("111", err)
        err = Alock.UnlockEx() //想删除的是Alock锁，但是Alock 已经被自动删除 ,Block由于value 不一样，所以也不会删除
        fmt.Println("11111", err) //这里一般情况下不会报错，因为函数内部是通过发送脚本来实现的。
        defer Alock.Free()
    }()

    time.Sleep(6 * time.Second) //此时Alock 已经被删除
    Block := rdft.CreateRedisLock(key)
    err := Block.LockEx(5) //此时 会获取新的lock Block
    fmt.Println("222", err)
    defer Block.Free()

    time.Sleep(2 * time.Second)
    Clock := rdft.CreateRedisLock(key)
    err = Clock.LockEx(5) //想获取新的lock Clock，但由于 Block还存在，返回错误
    fmt.Println("333", err)
    defer Clock.Free()

    time.Sleep(2 * time.Second)

    fmt.Println("***********************************")
}

func TestLock(t *testing.T) {
    rd := Redispool.Get()
    defer rd.Close()

    go func() {
        Alock := RedisLock{lockKey: "xxxxx"}
        err := Alock.Lock(rd, 5) //5 秒后自动删除Alock

        time.Sleep(7 * time.Second) //等待7秒
        fmt.Println("111", err)
        Alock.Unlock(rd) //想删除的是Alock锁，但是Alock 已经被自动删除 ,Block由于value 不一样，所以也不会删除
    }()

    time.Sleep(6 * time.Second) //此时Alock 已经被删除
    Block := RedisLock{lockKey: "xxxxx"}
    err := Block.Lock(rd, 5) //此时 会获取新的lock Block
    fmt.Println("222", err)

    time.Sleep(2 * time.Second)
    Clock := RedisLock{lockKey: "xxxxx"}
    err = Clock.Lock(rd, 5) //想获取新的lock Clock，但由于 Block还存在，返回错误
    fmt.Println("333", err)

    time.Sleep(2 * time.Second)

    fmt.Println("===================================")
}

var Redispool *redis.Pool

func init() {
    Redispool = &redis.Pool{
        MaxIdle:     10,
        IdleTimeout: 300 * time.Second,
        Dial: func() (redis.Conn, error) {
            tcp := fmt.Sprintf("%s:%d", "47.91.178.4", 9379)
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
