package discovery

import (
	"context"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

type Register struct {
	etcdClient *clientv3.Client
	leaseID    clientv3.LeaseID
	closeChan  chan struct{}
}

func NewRegister(endpoints []string) (*Register, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	return &Register{etcdClient: cli, closeChan: make(chan struct{})}, err
}

// RegisterService 注册服务并绑定租约（心跳）
func (r *Register) RegisterService(ctx context.Context, serviceName, addr string, ttl int64) error {
	// 1. 创建租约
	lease, err := r.etcdClient.Grant(ctx, ttl)
	if err != nil {
		return err
	}
	r.leaseID = lease.ID

	// 2. 注册服务节点
	key := fmt.Sprintf("/services/%s/%d", serviceName, r.leaseID)
	if _, err := r.etcdClient.Put(ctx, key, addr, clientv3.WithLease(r.leaseID)); err != nil {
		return err
	}

	// 3. 自动续租
	kaChan, err := r.etcdClient.KeepAlive(ctx, r.leaseID)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-r.closeChan:
				return
			case <-kaChan:
				// 心跳维持
			}
		}
	}()
	return nil
}

func (r *Register) Stop() {
	close(r.closeChan)
	if r.leaseID != 0 {
		r.etcdClient.Revoke(context.Background(), r.leaseID)
	}
	r.etcdClient.Close()
}
