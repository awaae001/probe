package client

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	pb "discord-bot/grpc/proto/gen/registry"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// RegistryClient 网关注册客户端
type RegistryClient struct {
	conn            *grpc.ClientConn
	client          pb.RegistryServiceClient
	stream          pb.RegistryService_EstablishConnectionClient
	serverAddr      string
	apiKey          string
	clientName      string
	connectionID    string
	ctx             context.Context
	cancel          context.CancelFunc
	heartbeatTicker *time.Ticker
	reconnectCount  int
	maxReconnects   int
	isConnected     bool
}

// NewRegistryClient 创建新的注册客户端
func NewRegistryClient(serverAddr, apiKey, clientName string) *RegistryClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &RegistryClient{
		serverAddr:    serverAddr,
		apiKey:        apiKey,
		clientName:    clientName,
		ctx:           ctx,
		cancel:        cancel,
		maxReconnects: -1, // -1 表示无限重连
		isConnected:   false,
	}
}

// Connect 连接到网关服务器
func (rc *RegistryClient) Connect() error {
	// 建立 gRPC 连接
	conn, err := grpc.NewClient(
		rc.serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	rc.conn = conn
	rc.client = pb.NewRegistryServiceClient(conn)

	log.Printf("[gRPC] 已连接到网关服务器: %s", rc.serverAddr)

	// 建立双向流连接
	stream, err := rc.client.EstablishConnection(rc.ctx)
	if err != nil {
		return fmt.Errorf("failed to establish connection stream: %w", err)
	}

	rc.stream = stream

	// 发送注册消息
	if err := rc.register(); err != nil {
		return fmt.Errorf("failed to register: %w", err)
	}

	// 启动接收消息的 goroutine
	go rc.receiveMessages()

	// 启动心跳
	rc.startHeartbeat()

	rc.isConnected = true
	rc.reconnectCount = 0 // 重置重连计数
	log.Printf("[gRPC] 客户端 '%s' 注册成功", rc.clientName)

	return nil
}

// register 发送注册消息
func (rc *RegistryClient) register() error {
	registerMsg := &pb.ConnectionMessage{
		MessageType: &pb.ConnectionMessage_Register{
			Register: &pb.ConnectionRegister{
				ApiKey:       rc.apiKey,
				Services:     []string{}, // 空实现，暂不提供服务
				ConnectionId: "",         // 留空，等待服务端分配 UUID
			},
		},
	}

	if err := rc.stream.Send(registerMsg); err != nil {
		return fmt.Errorf("failed to send register message: %w", err)
	}

	log.Printf("[gRPC] 已发送注册消息，等待服务端分配连接ID")

	return nil
}

// startHeartbeat 启动心跳
func (rc *RegistryClient) startHeartbeat() {
	rc.heartbeatTicker = time.NewTicker(30 * time.Second)

	go func() {
		for {
			select {
			case <-rc.ctx.Done():
				return
			case <-rc.heartbeatTicker.C:
				if err := rc.sendHeartbeat(); err != nil {
					log.Printf("[gRPC] 发送心跳失败: %v", err)
				}
			}
		}
	}()

	log.Printf("[gRPC] 心跳已启动 (间隔: 30秒)")
}

// sendHeartbeat 发送心跳消息
func (rc *RegistryClient) sendHeartbeat() error {
	heartbeatMsg := &pb.ConnectionMessage{
		MessageType: &pb.ConnectionMessage_Heartbeat{
			Heartbeat: &pb.Heartbeat{
				Timestamp:    time.Now().Unix(),
				ConnectionId: rc.connectionID,
			},
		},
	}

	if err := rc.stream.Send(heartbeatMsg); err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}

	// log.Printf("[gRPC] 已发送心跳")
	return nil
}

// receiveMessages 接收来自网关的消息
func (rc *RegistryClient) receiveMessages() {
	for {
		msg, err := rc.stream.Recv()
		if err != nil {
			if err == io.EOF {
				log.Printf("[gRPC] 连接已关闭")
			} else {
				log.Printf("[gRPC] 接收消息错误: %v", err)
			}

			// 标记为断开连接
			rc.isConnected = false

			// 触发重连
			go rc.reconnect()
			return
		}

		rc.handleMessage(msg)
	}
}

// handleMessage 处理接收到的消息
func (rc *RegistryClient) handleMessage(msg *pb.ConnectionMessage) {
	switch msg.MessageType.(type) {
	case *pb.ConnectionMessage_Request:
		// 空实现：暂不处理请求转发
		req := msg.GetRequest()
		log.Printf("[gRPC] 收到转发请求 (空实现): request_id=%s, method=%s",
			req.RequestId, req.MethodPath)

	case *pb.ConnectionMessage_Heartbeat:
		// 收到服务器心跳
		hb := msg.GetHeartbeat()
		log.Printf("[gRPC] 收到服务器心跳: timestamp=%d", hb.Timestamp)

	case *pb.ConnectionMessage_Status:
		// 连接状态消息
		status := msg.GetStatus()
		log.Printf("[gRPC] 连接状态: %s - %s", status.Status, status.Message)

		// 如果是 CONNECTED 状态，保存服务端分配的连接ID
		if status.Status == pb.ConnectionStatus_CONNECTED && status.ConnectionId != "" {
			rc.connectionID = status.ConnectionId
			log.Printf("[gRPC] 已收到服务端分配的连接ID: %s", rc.connectionID)
		}

	case *pb.ConnectionMessage_Event:
		// 事件消息
		event := msg.GetEvent()
		log.Printf("[gRPC] 收到事件: type=%s, id=%s", event.EventType, event.EventId)

	default:
		log.Printf("[gRPC] 收到未知消息类型")
	}
}

// reconnect 重连到网关服务器
func (rc *RegistryClient) reconnect() {
	// 如果上下文已取消，不进行重连
	if rc.ctx.Err() != nil {
		log.Printf("[gRPC] 上下文已取消，停止重连")
		return
	}

	// 如果已连接，不重复重连
	if rc.isConnected {
		return
	}

	rc.reconnectCount++

	// 检查是否超过最大重连次数
	if rc.maxReconnects > 0 && rc.reconnectCount > rc.maxReconnects {
		log.Printf("[gRPC] 已达到最大重连次数 (%d)，停止重连", rc.maxReconnects)
		return
	}

	// 计算退避延迟：指数退避，最大 60 秒
	backoff := time.Duration(1<<uint(rc.reconnectCount-1)) * time.Second
	if backoff > 60*time.Second {
		backoff = 60 * time.Second
	}

	log.Printf("[gRPC] 第 %d 次重连尝试，等待 %v 后重连...", rc.reconnectCount, backoff)

	select {
	case <-time.After(backoff):
		// 继续重连
	case <-rc.ctx.Done():
		log.Printf("[gRPC] 上下文已取消，停止重连")
		return
	}

	// 清理旧连接
	if rc.heartbeatTicker != nil {
		rc.heartbeatTicker.Stop()
	}
	if rc.stream != nil {
		rc.stream.CloseSend()
	}
	if rc.conn != nil {
		rc.conn.Close()
	}

	// 重新连接
	log.Printf("[gRPC] 正在尝试重新连接...")
	if err := rc.Connect(); err != nil {
		log.Printf("[gRPC] 重连失败: %v", err)
		// 继续下一次重连尝试
		go rc.reconnect()
	} else {
		log.Printf("[gRPC] 重连成功")
	}
}

// Close 关闭客户端连接
func (rc *RegistryClient) Close() error {
	log.Printf("[gRPC] 正在关闭客户端连接...")

	// 标记为未连接
	rc.isConnected = false

	// 取消上下文（这会停止重连尝试）
	if rc.cancel != nil {
		rc.cancel()
	}

	// 停止心跳
	if rc.heartbeatTicker != nil {
		rc.heartbeatTicker.Stop()
	}

	// 关闭流
	if rc.stream != nil {
		if err := rc.stream.CloseSend(); err != nil {
			log.Printf("[gRPC] 关闭流失败: %v", err)
		}
	}

	// 关闭连接
	if rc.conn != nil {
		if err := rc.conn.Close(); err != nil {
			log.Printf("[gRPC] 关闭连接失败: %v", err)
			return err
		}
	}

	log.Printf("[gRPC] 客户端连接已关闭")
	return nil
}
