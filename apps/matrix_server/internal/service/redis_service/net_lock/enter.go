package net_lock

// File: matrix_server/service/redis_service/net_lock/enter.go
// Description: 子网分布式锁管理模块，基于redsync实现子网级别的分布式锁管控，通过锁会话与自动续租机制保障长耗时异步操作的并发安全性

import (
	"context"
	"fmt"
	"matrix_server/internal/global"
	"sync"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/sirupsen/logrus"
)

const (
	lockExpiry        = 60 * time.Second // 锁过期时间60秒，覆盖长耗时异步场景
	lockRenewInterval = 20 * time.Second // 锁自动续租间隔20秒
	lockExtendTimeout = 5 * time.Second  // 单次续租操作的超时时间5秒
)

// lockSession 子网分布式锁会话，持有锁实例及续租生命周期控制对象
type lockSession struct {
	mutex  *redsync.Mutex     // redsync分布式锁实例
	cancel context.CancelFunc // 续租协程取消函数
	done   chan struct{}      // 续租协程退出通知通道
}

// netLockStore 子网分布式锁会话的缓存容器
var netLockStore sync.Map

// newMutex 创建指定子网对应的redsync互斥锁实例
func newMutex(netID uint) *redsync.Mutex {
	// 创建Redis连接池
	pool := goredis.NewPool(global.Redis)
	// 创建redsync实例
	rs := redsync.New(pool)

	// 构建子网分布式锁的Key（格式：net_action_lock_子网ID）
	return rs.NewMutex(
		fmt.Sprintf("net_action_lock_%d", netID),
		redsync.WithExpiry(lockExpiry),
		redsync.WithTries(1),
		redsync.WithRetryDelay(500*time.Millisecond),
	)
}

// Lock 为指定子网加分布式锁
func Lock(netID uint) error {
	// 若当前进程中已存在该子网的活跃锁会话，则直接返回失败，避免重复加锁和重复续租
	if _, ok := netLockStore.Load(netID); ok {
		return redsync.ErrFailed
	}

	// 创建新的分布式锁实例并尝试获取锁
	mutex := newMutex(netID)
	if err := mutex.Lock(); err != nil {
		return err
	}

	// 加锁成功后创建锁会话及续租生命周期控制对象
	ctx, cancel := context.WithCancel(context.Background())
	session := &lockSession{
		mutex:  mutex,
		cancel: cancel,
		done:   make(chan struct{}),
	}

	// 将锁会话写入缓存；若并发情况下已有其他会话写入，则回滚本次加锁结果
	if _, loaded := netLockStore.LoadOrStore(netID, session); loaded {
		cancel()
		if _, err := mutex.Unlock(); err != nil {
			logrus.WithError(err).Warnf("子网%d本地锁会话冲突后回滚解锁失败", netID)
		}
		return redsync.ErrFailed
	}

	// 启动后台续租协程，保障异步批处理期间锁不会因TTL到期而提前失效
	go renewLock(netID, session, ctx)
	return nil
}

// renewLock 后台定时续租指定子网的分布式锁
func renewLock(netID uint, session *lockSession, ctx context.Context) {
	ticker := time.NewTicker(lockRenewInterval)
	defer ticker.Stop()
	defer close(session.done)

	for {
		select {
		case <-ctx.Done():
			// 收到取消信号后退出续租协程
			return
		case <-ticker.C:
			// 定时为锁续期，避免长耗时异步操作期间锁提前过期
			extendCtx, cancel := context.WithTimeout(context.Background(), lockExtendTimeout)
			ok, err := session.mutex.ExtendContext(extendCtx)
			cancel()
			if err != nil {
				logrus.WithError(err).Errorf("子网%d分布式锁续租失败", netID)
				clearSession(netID, session)
				return
			}
			if !ok {
				logrus.Warnf("子网%d分布式锁续租未生效，锁会话已失效", netID)
				clearSession(netID, session)
				return
			}
		}
	}
}

// clearSession 清理指定子网的锁会话缓存（仅当当前缓存仍指向该会话时）
func clearSession(netID uint, session *lockSession) {
	if current, ok := netLockStore.Load(netID); ok && current == session {
		netLockStore.Delete(netID)
	}
}

// UnLock 释放指定子网的分布式锁
func UnLock(netID uint) (bool, error) {
	// 从缓存获取该子网对应的锁会话
	sessionValue, ok := netLockStore.Load(netID)
	if !ok {
		logrus.Errorf("不存在的子网分布式锁，子网ID：%d", netID)
		return false, nil
	}

	// 先停止后台续租，再等待续租协程退出，避免解锁与续租并发冲突
	session := sessionValue.(*lockSession)
	session.cancel()
	<-session.done

	// 执行解锁操作并清理本地锁会话缓存
	ok, err := session.mutex.Unlock()
	netLockStore.Delete(netID)

	return ok, err
}
