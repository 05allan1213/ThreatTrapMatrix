package net_lock

// File: honey_server/service/redis_service/net_lock/enter.go
// Description: 子网分布式锁模块，基于Redis的RedSync实现子网级别的分布式互斥锁，通过锁会话与自动续租机制防止长耗时异步操作期间锁提前失效

import (
	"context"
	"fmt"
	"honey_server/internal/global"
	"sync"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/sirupsen/logrus"
)

const (
	lockExpiry        = 60 * time.Second // 锁过期时间60秒
	lockRenewInterval = 20 * time.Second // 锁自动续租间隔20秒
	lockExtendTimeout = 5 * time.Second  // 单次续租超时时间5秒
)

// lockSession 子网分布式锁会话，封装锁实例及续租控制对象
type lockSession struct {
	mutex  *redsync.Mutex     // RedSync互斥锁实例
	cancel context.CancelFunc // 续租协程取消函数
	done   chan struct{}      // 续租协程退出通知通道
}

// netLockStore 子网分布式锁会话缓存
var netLockStore sync.Map

// newMutex 创建指定子网对应的分布式锁实例
func newMutex(netID uint) *redsync.Mutex {
	// 初始化Redis连接池（基于全局Redis客户端）
	pool := goredis.NewPool(global.Redis)
	rs := redsync.New(pool)

	// 构建锁名称（子网ID唯一标识）
	return rs.NewMutex(
		fmt.Sprintf("net_action_lock_%d", netID),
		redsync.WithExpiry(lockExpiry),
		redsync.WithTries(1),
		redsync.WithRetryDelay(500*time.Millisecond),
	)
}

// Lock 获取指定子网的分布式锁
func Lock(netID uint) error {
	// 若当前进程已持有该子网锁会话，则直接返回失败，避免重复启动续租协程
	if _, ok := netLockStore.Load(netID); ok {
		return redsync.ErrFailed
	}

	// 创建新的分布式锁实例并尝试获取锁
	mutex := newMutex(netID)
	if err := mutex.Lock(); err != nil {
		return err
	}

	// 加锁成功后创建锁会话，用于管理续租协程生命周期
	ctx, cancel := context.WithCancel(context.Background())
	session := &lockSession{
		mutex:  mutex,
		cancel: cancel,
		done:   make(chan struct{}),
	}

	// 将锁会话写入缓存；若并发情况下已有其他会话落库，则回滚本次加锁结果
	if _, loaded := netLockStore.LoadOrStore(netID, session); loaded {
		cancel()
		if _, err := mutex.Unlock(); err != nil {
			logrus.WithError(err).Warnf("子网%d本地锁会话冲突后回滚解锁失败", netID)
		}
		return redsync.ErrFailed
	}

	// 启动后台续租协程，保障异步回调落库期间锁持续有效
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
			// 定时续租，避免锁在异步回调处理完成前过期
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

// clearSession 清理指定子网的锁会话缓存（仅当缓存仍指向当前会话时）
func clearSession(netID uint, session *lockSession) {
	if current, ok := netLockStore.Load(netID); ok && current == session {
		netLockStore.Delete(netID)
	}
}

// UnLock 释放指定子网的分布式锁
func UnLock(netID uint) (bool, error) {
	// 从缓存加载锁会话
	sessionValue, ok := netLockStore.Load(netID)
	if !ok {
		// 缓存中无锁实例，记录错误日志
		logrus.Errorf("子网%d不存在分布式锁实例，释放操作失败", netID)
		return false, nil
	}

	// 先停止后台续租，再等待续租协程退出，最后执行释放操作
	session := sessionValue.(*lockSession)
	session.cancel()
	<-session.done

	ok, err := session.mutex.Unlock()
	netLockStore.Delete(netID)

	return ok, err
}
