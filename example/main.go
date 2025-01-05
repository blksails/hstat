package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pkg.blksails.net/x/hstat"
)

// 模拟用户活动
func simulateUserActivity(window *hstat.TimeWindow, done chan struct{}) {
	ticker := time.NewTicker(200 * time.Millisecond) // 每200ms模拟一次活动
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 随机暂停，模拟无活动期间
			if rand.Float64() < 0.1 { // 10%的概率暂停
				time.Sleep(2 * time.Second)
				continue
			}

			// 随机模拟用户上线或下线
			if rand.Float64() > 0.5 {
				// 模拟1-3人上线
				delta := rand.Float64()*2 + 1
				window.Inc(delta)
			} else {
				// 模拟1-2人下线
				delta := rand.Float64() + 1
				window.Dec(delta)
			}
		case <-done:
			return
		}
	}
}

// 显示统计图表
func displayStats(window *hstat.TimeWindow, done chan struct{}) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	opt := &hstat.HistogramOption{
		Height: 15,
	}

	clearScreen := "\033[H\033[2J"

	for {
		select {
		case <-ticker.C:
			fmt.Print(clearScreen)

			now := time.Now()
			lastUpdate := window.LastUpdateTime()
			total := window.Sum()

			// 计算最近更新距现在的时间
			var timeSinceUpdate string
			if !lastUpdate.IsZero() {
				duration := now.Sub(lastUpdate)
				if duration < time.Second {
					timeSinceUpdate = "刚刚"
				} else {
					timeSinceUpdate = fmt.Sprintf("%.1f秒前", duration.Seconds())
				}
			} else {
				timeSinceUpdate = "暂无数据"
			}

			// 显示标题和统计信息
			fmt.Printf("实时在线人数监控 [%s]\n", now.Format("15:04:05"))
			fmt.Printf("当前在线总人数: %.0f    最近更新: %s\n", total, timeSinceUpdate)

			// 显示直方图
			fmt.Print(window.PrintHistogram(opt))
		case <-done:
			return
		}
	}
}

func main() {
	// 设置随机数种子
	rand.Seed(time.Now().UnixNano())

	// 创建一个60秒的时间窗口
	window := hstat.NewTimeWindow(60, time.Second)

	// 创建用于优雅退出的通道
	done := make(chan struct{})
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动模拟器和显示器
	go simulateUserActivity(window, done)
	go displayStats(window, done)

	// 等待中断信号
	<-sigChan
	close(done) // 通知所有goroutine退出

	// 给goroutines一点时间来清理
	time.Sleep(100 * time.Millisecond)
	fmt.Println("\n程序已退出")
}
