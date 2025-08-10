package grpc

import (
	"context"
	"time"

	"discord-bot/proto"
)

// PostClient 扩展 gRPC 客户端，添加便捷的帖子查询方法
type PostClient struct {
	*Client
}

// NewPostClient 创建新的帖子客户端
func NewPostClient(serverAddress string, timeout time.Duration) (*PostClient, error) {
	client, err := NewClient(serverAddress, timeout)
	if err != nil {
		return nil, err
	}
	
	return &PostClient{Client: client}, nil
}

// QueryRecentPosts 查询最近N天的帖子
func (pc *PostClient) QueryRecentPosts(ctx context.Context, days int, opts ...QueryOption) (*proto.QueryPostsResponse, error) {
	timeRange := GetLastDaysRange(days)
	return pc.QueryPostsByTimeRange(ctx, timeRange.StartTime, timeRange.EndTime, opts...)
}

// QueryYesterdayPosts 查询昨天的帖子
func (pc *PostClient) QueryYesterdayPosts(ctx context.Context, opts ...QueryOption) (*proto.QueryPostsResponse, error) {
	timeRange := GetYesterdayRange()
	return pc.QueryPostsByTimeRange(ctx, timeRange.StartTime, timeRange.EndTime, opts...)
}

// QueryLastDayPosts 查询最近1天的帖子
func (pc *PostClient) QueryLastDayPosts(ctx context.Context, opts ...QueryOption) (*proto.QueryPostsResponse, error) {
	timeRange := GetLastDayRange()
	return pc.QueryPostsByTimeRange(ctx, timeRange.StartTime, timeRange.EndTime, opts...)
}

// QueryLast3DaysPosts 查询最近3天的帖子
func (pc *PostClient) QueryLast3DaysPosts(ctx context.Context, opts ...QueryOption) (*proto.QueryPostsResponse, error) {
	timeRange := GetLast3DaysRange()
	return pc.QueryPostsByTimeRange(ctx, timeRange.StartTime, timeRange.EndTime, opts...)
}

// QueryLast7DaysPosts 查询最近7天的帖子
func (pc *PostClient) QueryLast7DaysPosts(ctx context.Context, opts ...QueryOption) (*proto.QueryPostsResponse, error) {
	timeRange := GetLast7DaysRange()
	return pc.QueryPostsByTimeRange(ctx, timeRange.StartTime, timeRange.EndTime, opts...)
}

// QueryPostsByTimeRange 根据时间范围查询帖子
func (pc *PostClient) QueryPostsByTimeRange(ctx context.Context, startTime, endTime int64, opts ...QueryOption) (*proto.QueryPostsResponse, error) {
	req := &proto.QueryPostsRequest{
		StartTime: &startTime,
		EndTime:   &endTime,
	}
	
	// 应用查询选项
	for _, opt := range opts {
		opt(req)
	}
	
	return pc.Client.QueryPosts(ctx, req)
}

// QueryOption 查询选项函数类型
type QueryOption func(*proto.QueryPostsRequest)

// WithAuthorId 设置作者ID查询选项
func WithAuthorId(authorId string) QueryOption {
	return func(req *proto.QueryPostsRequest) {
		req.AuthorId = &authorId
	}
}

// WithChannelId 设置频道ID查询选项
func WithChannelId(channelId string) QueryOption {
	return func(req *proto.QueryPostsRequest) {
		req.ChannelId = &channelId
	}
}

// WithTags 设置标签查询选项
func WithTags(tags []string) QueryOption {
	return func(req *proto.QueryPostsRequest) {
		req.Tags = tags
	}
}

// WithPagination 设置分页查询选项
func WithPagination(pageSize, pageNumber int32) QueryOption {
	return func(req *proto.QueryPostsRequest) {
		req.PageSize = pageSize
		req.PageNumber = pageNumber
	}
}