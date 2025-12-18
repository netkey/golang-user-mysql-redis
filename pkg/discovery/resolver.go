package discovery

import (
	"fmt"
	"go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/resolver"
	"google.golang.org/grpc"
)

// GetGRPCClient 通过服务名获取一个具备自动发现能力的连接
func GetGRPCClient(etcdEndpoints []string, serviceName string) (*grpc.ClientConn, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints: etcdEndpoints,
	})
	if err != nil {
		return nil, err
	}

	// 使用 etcd 提供的官方命名解析器
	etcdResolver, err := resolver.NewBuilder(cli)
	if err != nil {
		return nil, err
	}

	// 目标地址格式：etcd:///services/user-service
	target := fmt.Sprintf("etcd:///services/%s", serviceName)

	return grpc.Dial(
		target,
		grpc.WithResolvers(etcdResolver),
		grpc.WithInsecure(),                                                    // 演示使用，生产应使用 TLS
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`), // 轮询负载均衡
	)
}
