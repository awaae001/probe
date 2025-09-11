package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// LoadConfig 从多个源加载配置：.env 文件、config.yaml、以及 ./config/ 目录下的 JSON 文件。
// 配置加载顺序:
// 1. .env 文件 (用于环境变量)
// 2. config.yaml (基础配置)
// 3. config/thread_config.json (合并到主配置)
// 4. config/scanning_config.json (合并到主配置)
// 环境变量会覆盖配置文件中的同名设置。
func LoadConfig() {
	// 1. 从 .env 文件加载环境变量，如果文件不存在则忽略。
	if err := godotenv.Load(); err != nil {
		log.Printf("未找到 .env 文件，将跳过加载。")
	}

	// 2. 设置并读取基础配置文件 (config.yaml)。
	viper.SetConfigName("config")                          // 配置文件名 (无扩展名)
	viper.SetConfigType("yaml")                            // 配置文件类型
	viper.AddConfigPath(".")                               // 在当前工作目录中查找
	viper.AutomaticEnv()                                   // 自动读取匹配的环境变量
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // 将配置键中的'.'替换为'_'以匹配环境变量

	// 读取基础配置。
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 配置文件未找到是正常情况，可以继续。
			log.Printf("未找到基础配置文件 (config.yaml)，将仅使用环境变量和后续合并的配置。")
		} else {
			// 如果找到配置文件但解析出错，则终止程序。
			panic(fmt.Errorf("解析基础配置文件时发生致命错误: %w", err))
		}
	}

	// 3. 合并线程配置文件 (config/thread_config.json)。
	// MergeInConfig 会将配置合并到现有的 viper 配置中。
	viper.SetConfigName("thread_config")
	viper.SetConfigType("json")
	viper.AddConfigPath("./config")

	if err := viper.MergeInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Printf("未找到线程配置文件 (config/thread_config.json)，将跳过合并。")
		} else {
			panic(fmt.Errorf("合并线程配置文件时发生致命错误: %w", err))
		}
	}

	// 4. 合并扫描配置文件 (config/scanning_config.json)。
	viper.SetConfigName("scanning_config")
	viper.SetConfigType("json")
	viper.AddConfigPath("./config")

	if err := viper.MergeInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Printf("未找到扫描配置文件 (config/scanning_config.json)，将跳过合并。")
		} else {
			panic(fmt.Errorf("合并扫描配置文件时发生致命错误: %w", err))
		}
	}

	viper.SetConfigName("new_scan")
	viper.SetConfigType("json")
	viper.AddConfigPath("./config")

	if err := viper.MergeInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Printf("未找到新人加入扫描配置文件 (config/new_scan.json)，将跳过合并。")
		} else {
			panic(fmt.Errorf("合并新人加入扫描配置文件时发生致命错误: %w", err))
		}
	}

	// 5. 合并消息监听器配置文件 (config/message_listener.json)
	viper.SetConfigName("message_listener")
	viper.SetConfigType("json")
	viper.AddConfigPath("./config")

	if err := viper.MergeInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Printf("未找到消息监听器配置文件 (config/message_listener.json)，将跳过合并。")
		} else {
			panic(fmt.Errorf("合并消息监听器配置文件时发生致命错误: %w", err))
		}
	}
}
