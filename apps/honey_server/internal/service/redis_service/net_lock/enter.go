package net_lock

// File: honey_server/service/redis_service/net_lock/enter.go
// Description: 子网分布式锁模块，基于Redis的RedSync实现子网级别的分布式互斥锁，防止多节点/多进程并发操作同一子网，保障子网部署/删除/更新等操作的原子性

import (
	"fmt"
	"honey_server/internal/global"
	"sync"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/sirupsen/logrus"
)

// netLockStore 子网分布式锁实例缓存
var netLockStore = sync.Map{}

// Lock 获取指定子网的分布式锁
func Lock(netID uint) error {
	// 1. 从缓存加载已存在的锁实例
	_mutex, ok := netLockStore.Load(netID)
	if !ok {
		// 2. 初始化Redis连接池（基于全局Redis客户端）
		pool := goredis.NewPool(global.Redis)
		rs := redsync.New(pool)

		// 3. 构建锁名称（子网ID唯一标识）
		mutexName := fmt.Sprintf("net_action_lock_%d", netID)

		// 4. 创建分布式锁实例，配置锁参数
		_mutex = rs.NewMutex(mutexName,
			redsync.WithExpiry(20*time.Second),           // 锁过期时间20秒
			redsync.WithTries(1),                         // 锁获取重试次数1次
			redsync.WithRetryDelay(500*time.Millisecond), // 重试间隔500毫秒
		)

		// 5. 将锁实例存入缓存
		netLockStore.Store(netID, _mutex)
	}

	// 6. 类型断言转换为RedSync互斥锁实例，尝试获取锁
	mutex := _mutex.(*redsync.Mutex)
	return mutex.Lock()
}

// UnLock 释放指定子网的分布式锁
func UnLock(netID uint) (bool, error) {
	// 从缓存加载锁实例
	_mutex, ok := netLockStore.Load(netID)
	if !ok {
		// 缓存中无锁实例，记录错误日志
		logrus.Errorf("子网%d不存在分布式锁实例，释放操作失败", netID)
		return false, nil
	}

	// 类型断言转换为RedSync互斥锁实例，执行释放操作
	mutex := _mutex.(*redsync.Mutex)
	return mutex.Unlock()
}
