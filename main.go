package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

func main() {
	// 从 .env 文件加载环境变量
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	// 初始化 Viper
	viper.SetConfigName("config") // 配置文件名 (不带扩展名)
	viper.SetConfigType("yml")    // 配置文件类型
	viper.AddConfigPath(".")      // 查找配置文件的路径
	viper.AutomaticEnv()          // 自动从环境变量读取
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 配置文件未找到；可以忽略
			fmt.Println("Config file not found, using environment variables and defaults.")
		} else {
			// 找到配置文件但解析时发生错误
			panic(fmt.Errorf("fatal error config file: %w", err))
		}
	}

	// 从 Viper 获取 Bot Token
	token := viper.GetString("BOT_TOKEN")
	if token == "" {
		fmt.Println("No bot token provided. Please set the BOT_TOKEN in your .env or config file.")
		return
	}

	// 创建一个新的 Discord session
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// 注册 messageCreate 函数作为一个回调
	dg.AddHandler(messageCreate)

	// 设置 intents
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// 打开一个 websocket 连接到 Discord 并开始监听
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// 等待终止信号
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// 优雅地关闭 discord session
	dg.Close()
}

// 当 bot 收到 "message create" 事件时，这个函数将被调用
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// 忽略所有由 bot 自己发送的消息
	if m.Author.ID == s.State.User.ID {
		return
	}

	// 从 Viper 获取命令前缀
	prefix := viper.GetString("bot.prefix")
	if prefix == "" {
		prefix = "!" // 默认前缀
	}

	// 检查消息是否以命令前缀开头
	if strings.HasPrefix(m.Content, prefix) {
		// 去掉前缀来获取命令
		command := strings.TrimPrefix(m.Content, prefix)

		switch command {
		case "ping":
			s.ChannelMessageSend(m.ChannelID, "Pong!")
		case "pong":
			s.ChannelMessageSend(m.ChannelID, "Ping!")
		}
	}
}
