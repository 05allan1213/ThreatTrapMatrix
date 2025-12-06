package net_lock

// File: matrix_server/service/redis_service/net_lock/enter.go
// Description: 子网分布式锁管理模块，基于redsync实现子网级别的分布式锁管控，通过sync.Map缓存锁实例避免重复创建，提供加锁/解锁核心方法，保障子网操作的并发安全性

import (
	"fmt"
	"matrix_server/internal/global"
	"sync"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/sirupsen/logrus"
)

// netLockStore 子网分布式锁实例的缓存容器
var netLockStore = sync.Map{}

// Lock 为指定子网加分布式锁
func Lock(netID uint) error {
	// 从缓存获取该子网对应的锁实例
	_mutex, ok := netLockStore.Load(netID)
	if !ok {
		// 缓存未命中时，创建Redis连接池
		pool := goredis.NewPool(global.Redis)
		// 创建redsync实例
		rs := redsync.New(pool)
		// 构建子网分布式锁的Key（格式：net_action_lock_子网ID）
		mutexName := fmt.Sprintf("net_action_lock_%d", netID)
		// 创建基于子网ID的分布式互斥锁
		_mutex = rs.NewMutex(mutexName,
			redsync.WithExpiry(20*time.Second),           // 锁过期时间20秒（自动释放兜底）
			redsync.WithTries(1),                         // 锁获取重试次数1次（不重复尝试）
			redsync.WithRetryDelay(500*time.Millisecond), // 锁获取重试间隔500毫秒
		)
		// 将新创建的锁实例存入缓存
		netLockStore.Store(netID, _mutex)
	}
	// 类型断言转换为redsync.Mutex实例
	mutex := _mutex.(*redsync.Mutex)
	// 尝试获取分布式锁
	return mutex.Lock()
}

// UnLock 释放指定子网的分布式锁
func UnLock(netID uint) (bool, error) {
	// 从缓存获取该子网对应的锁实例
	_mutex, ok := netLockStore.Load(netID)
	if !ok {
		// 缓存未命中时，记录错误日志
		logrus.Errorf("不存在的子网分布式锁，子网ID：%d", netID)
		return false, nil
	}
	// 类型断言转换为redsync.Mutex实例
	mutex := _mutex.(*redsync.Mutex)
	// 执行解锁操作并返回结果
	return mutex.Unlock()
}
