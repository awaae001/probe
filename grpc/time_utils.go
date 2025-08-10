package grpc

import (
	"strconv"
	"time"
)

// TimeRange 表示时间范围
type TimeRange struct {
	StartTime int64 // 开始时间戳
	EndTime   int64 // 结束时间戳
}

// GetYesterdayRange 获取昨天的时间范围
func GetYesterdayRange() TimeRange {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	// 昨天的开始时间
	startOfDay := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location())
	// 昨天的结束时间
	endOfDay := startOfDay.Add(24 * time.Hour)

	return TimeRange{
		StartTime: startOfDay.Unix(),
		EndTime:   endOfDay.Unix(),
	}
}

// GetLastDaysRange 获取最近N天的时间范围
func GetLastDaysRange(days int) TimeRange {
	now := time.Now()

	// N天前的开始时间
	startDaysAgo := now.AddDate(0, 0, -days)
	startOfDay := time.Date(startDaysAgo.Year(), startDaysAgo.Month(), startDaysAgo.Day(), 0, 0, 0, 0, startDaysAgo.Location())

	return TimeRange{
		StartTime: startOfDay.Unix(),
		EndTime:   now.Unix(),
	}
}

// GetLastDayRange 获取最近1天的时间范围
func GetLastDayRange() TimeRange {
	return GetLastDaysRange(1)
}

// GetLast3DaysRange 获取最近3天的时间范围
func GetLast3DaysRange() TimeRange {
	return GetLastDaysRange(3)
}

// GetLast7DaysRange 获取最近7天的时间范围
func GetLast7DaysRange() TimeRange {
	return GetLastDaysRange(7)
}

// FormatTimestamp 格式化时间戳为可读字符串
func FormatTimestamp(timestamp int64) string {
	return time.Unix(timestamp, 0).Format("2006-01-02 15:04:05")
}

// GetTimeRangeLabel 获取时间范围的标签描述
func GetTimeRangeLabel(days int) string {
	switch days {
	case 1:
		return "最近1天"
	case 3:
		return "最近3天"
	case 7:
		return "最近7天"
	default:
		return "最近" + strconv.Itoa(days) + "天"
	}
}
