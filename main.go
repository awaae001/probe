package main

import (
	"discord-bot/config"
	"discord-bot/scanner"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
)

func main() {
	// Load configuration
	config.LoadConfig()

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
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuilds

	// 打开一个 websocket 连接到 Discord 并开始监听
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// 设置定时任务
	c := cron.New()
	_, err = c.AddFunc("@hourly", func() {
		log.Println("Running hourly scan...")
		scanner.StartScanning(dg)
	})
	if err != nil {
		log.Fatalf("Could not set up cron job: %v", err)
	}
	c.Start()
	log.Println("Cron job scheduled to run hourly.")

	// 第一次启动时立即执行一次扫描
	go func() {
		log.Println("Performing initial scan...")
		scanner.StartScanning(dg)
	}()

	// 等待终止信号
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// 优雅地关闭 discord session 和定时任务
	c.Stop()
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
