package grpc

import (
	"context"
	"log"
	"time"

	"discord-bot/proto"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client 封装 gRPC 客户端连接
type Client struct {
	conn          *grpc.ClientConn
	postClient    proto.PostServiceClient
	serverAddress string
	timeout       time.Duration
}

// NewClient 创建新的 gRPC 客户端
func NewClient(serverAddress string, timeout time.Duration) (*Client, error) {
	conn, err := grpc.Dial(serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:          conn,
		postClient:    proto.NewPostServiceClient(conn),
		serverAddress: serverAddress,
		timeout:       timeout,
	}, nil
}

// Close 关闭 gRPC 连接
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// GetPostClient 获取 PostService 客户端
func (c *Client) GetPostClient() proto.PostServiceClient {
	return c.postClient
}

// GetServerAddress 获取服务器地址
func (c *Client) GetServerAddress() string {
	return c.serverAddress
}

// QueryPosts 查询帖子的通用方法
func (c *Client) QueryPosts(ctx context.Context, req *proto.QueryPostsRequest) (*proto.QueryPostsResponse, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), c.timeout)
		defer cancel()
	}

	log.Printf("Querying posts from gRPC server at %s with params: author_id=%v, channel_id=%v, start_time=%v, end_time=%v",
		c.serverAddress, req.GetAuthorId(), req.GetChannelId(), req.GetStartTime(), req.GetEndTime())

	resp, err := c.postClient.QueryPosts(ctx, req)
	if err != nil {
		log.Printf("Error querying posts from gRPC server: %v", err)
		return nil, err
	}

	log.Printf("Successfully retrieved %d posts from gRPC server", len(resp.GetPosts()))
	return resp, nil
}
