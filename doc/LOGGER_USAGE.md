# 日志记录器使用指南

本指南说明了如何在项目中使用新添加的日志记录器。

## 概述

日志记录器允许将格式化的嵌入式消息发送到在 `config.yaml` 文件中配置的 Discord 管理员频道。它支持三种日志级别：`INFO`、`WARN` 和 `ERROR`。

## 初始化

日志记录器在机器人启动时自动初始化。无需手动初始化。

## 如何使用

要记录消息，请从 `utils` 包中导入并调用相应的函数。

### 导入

首先，在需要记录日志的文件中导入 `utils` 包：

```go
import "discord-bot/utils"
```

### 日志级别和功能

提供了三种日志记录功能，对应不同的日志级别：

*   `utils.Info(module, operation, details string)`
*   `utils.Warn(module, operation, details string)`
*   `utils.Error(module, operation, details string)`

### 参数

每个函数都接受三个字符串参数：

*   `module`: 产生日志的模块或组件的名称 (例如, "Bot", "Scanner", "CommandHandler")。
*   `operation`: 正在执行的操作 (例如, "Startup", "Shutdown", "SendMessage")。
*   `details`: 关于日志事件的详细信息。

### 示例

以下是如何在代码中使用日志记录器的示例：

```go
package main

import (
	"discord-bot/bot"
	"discord-bot/handlers"
	"discord-bot/utils"
)

func main() {
	// 示例：在应用程序启动时记录一条信息
	utils.Info("Main", "Application Start", "Application is starting up.")

	bot.Run(handlers.Register)

	// 示例：在应用程序关闭时记录一条信息
	utils.Info("Main", "Application Shutdown", "Application is shutting down.")
}
```

### 日志输出

当调用日志函数时，将向 `config.yaml` 中 `bot.adminChanneId` 指定的 Discord 频道发送一条嵌入式消息。嵌入消息将包含：

*   **日志级别**: INFO, WARN, 或 ERROR.
*   **颜色**: 绿色代表 INFO, 黄色代表 WARN, 红色代表 ERROR.
*   **模块**: 您指定的模块名称。
*   **操作**: 您指定的操作名称。
*   **附加信息**: 您提供的详细信息。
*   **时间戳**: 日志事件发生的时间。

如果 `bot.adminChanneId` 未在配置中设置，日志消息将作为后备，打印到标准控制台输出。