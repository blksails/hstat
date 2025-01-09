package hstat

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// TimeWindow 表示一个基于时间的滑动窗口
type TimeWindow struct {
	mu         sync.RWMutex
	buckets    []float64     // 改为单个float64值的切片
	size       int           // 窗口大小(桶的数量)
	duration   time.Duration // 每个桶的时间跨度
	lastTime   time.Time     // 上次更新时间
	cursor     int           // 当前桶的位置
	lastUpdate time.Time     // 最近一次数据更新时间
}

// NewTimeWindow 创建一个新的时间窗口
// size: 窗口中桶的数量
// duration: 每个桶的时间跨度
// chartHeight: 图表最大高度（如果 <= 0，则使用默认值20）
func NewTimeWindow(size int, duration time.Duration) *TimeWindow {
	return &TimeWindow{
		buckets:  make([]float64, size),
		size:     size,
		duration: duration,
		lastTime: time.Now(),
	}
}

// Append 添加一个值到当前时间窗口
func (w *TimeWindow) Append(value float64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	w.rotate(now)

	// 直接设置当前桶的值
	w.buckets[w.cursor] = value
}

// rotate 根据时间推移调整窗口
func (w *TimeWindow) rotate(now time.Time) {
	passed := int(now.Sub(w.lastTime) / w.duration)
	if passed <= 0 {
		return
	}

	// 如果经过的时间超过窗口大小，清空所有桶
	if passed >= w.size {
		for i := range w.buckets {
			w.buckets[i] = 0
		}
		w.cursor = 0
	} else {
		// 清空过期的桶
		for i := 0; i < passed; i++ {
			w.cursor = (w.cursor + 1) % w.size
			w.buckets[w.cursor] = 0
		}
	}

	w.lastTime = now
}

// Sum 计算窗口内所有值的和
func (w *TimeWindow) Sum() float64 {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var sum float64
	for _, v := range w.buckets {
		sum += v
	}
	return sum
}

// Count 返回窗口内的非零值的数量
func (w *TimeWindow) Count() int {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var count int
	for _, v := range w.buckets {
		if v != 0 {
			count++
		}
	}
	return count
}

// Avg 计算窗口内值的平均值
func (w *TimeWindow) Avg() float64 {
	w.mu.RLock()
	defer w.mu.RUnlock()

	count := w.Count()
	if count == 0 {
		return 0
	}
	return w.Sum() / float64(count)
}

// Inc 在当前时间窗口中累加值
func (w *TimeWindow) Inc(delta float64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	w.rotate(now)
	w.lastUpdate = now

	w.buckets[w.cursor] += delta
}

// Dec 在当前时间窗口中递减值
func (w *TimeWindow) Dec(delta float64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	w.rotate(now)
	w.lastUpdate = now

	w.buckets[w.cursor] -= delta
}

// Reset 重置当前桶的值为指定值
func (w *TimeWindow) Reset(value float64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	w.rotate(now)

	w.buckets[w.cursor] = value
}

// HistogramOption 用于配置直方图显示选项
type HistogramOption struct {
	Height int // 图表高度
}

// DefaultHistogramOption 返回默认的直方图配置
func DefaultHistogramOption() *HistogramOption {
	return &HistogramOption{
		Height: 20, // 默认高度
	}
}

// PrintHistogram 返回时间窗口内的数据分布情况（垂直柱状图）
func (w *TimeWindow) PrintHistogram(opt *HistogramOption) string {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 在显示之前先更新窗口状态
	w.rotate(time.Now())

	if opt == nil {
		opt = DefaultHistogramOption()
	}

	var result strings.Builder
	result.WriteString("\nTime Window Histogram:\n\n")

	// 获取所有值和时间，注意顺序要从最新到最旧
	values := make([]float64, w.size)
	times := make([]int, w.size)
	maxValue := 0.0

	// 从当前游标位置向前收集数据
	for i := 0; i < w.size; i++ {
		// 计算实际索引，从当前游标向前遍历
		idx := (w.cursor - i + w.size) % w.size
		times[i] = -i * int(w.duration.Seconds())

		if w.buckets[idx] > 0 {
			value := w.buckets[idx]
			values[i] = value
			if value > maxValue {
				maxValue = value
			}
		}
	}

	height := opt.Height
	if maxValue == 0 {
		return "No data available\n"
	}

	// 打印柱状图（从上到下）
	for h := height; h > 0; h-- {
		threshold := maxValue * float64(h) / float64(height)
		for i := 0; i < w.size; i++ {
			if values[i] >= threshold {
				result.WriteString("▇ ")
			} else {
				result.WriteString("  ")
			}
		}
		result.WriteString("\n")
	}

	// 打印底部分隔线
	for i := 0; i < w.size; i++ {
		result.WriteString("──")
	}
	result.WriteString("\n")

	// 打印数值
	for i := 0; i < w.size; i++ {
		if values[i] > 0 {
			fmt.Fprintf(&result, "%-2.0f", values[i])
		} else {
			result.WriteString("  ")
		}
	}
	result.WriteString("\n")

	// 打印相对时间刻度
	interval := 1
	if w.size > 20 {
		interval = w.size / 10
	}

	// 打印时间刻度
	for i := 0; i < w.size; i++ {
		if i%interval == 0 {
			fmt.Fprintf(&result, "%-2d", times[i])
		} else {
			result.WriteString("  ")
		}
	}
	result.WriteString("s\n")

	return result.String()
}

// LastUpdateTime 返回最近一次数据更新时间
func (w *TimeWindow) LastUpdateTime() time.Time {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.lastUpdate
}

// Value 实现 sql.Valuer 接口
func (w *TimeWindow) Value() (driver.Value, error) {
	if w == nil {
		return nil, nil
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	data := struct {
		Buckets    []float64     `json:"buckets"`
		Size       int           `json:"size"`
		Duration   time.Duration `json:"duration"`
		LastTime   time.Time     `json:"last_time"`
		Cursor     int           `json:"cursor"`
		LastUpdate time.Time     `json:"last_update"`
	}{
		Buckets:    w.buckets,
		Size:       w.size,
		Duration:   w.duration,
		LastTime:   w.lastTime,
		Cursor:     w.cursor,
		LastUpdate: w.lastUpdate,
	}

	return json.Marshal(data)
}

// Scan 实现 sql.Scanner 接口
func (w *TimeWindow) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	var data struct {
		Buckets    []float64     `json:"buckets"`
		Size       int           `json:"size"`
		Duration   time.Duration `json:"duration"`
		LastTime   time.Time     `json:"last_time"`
		Cursor     int           `json:"cursor"`
		LastUpdate time.Time     `json:"last_update"`
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("expected []byte, got %T", value)
	}

	if err := json.Unmarshal(bytes, &data); err != nil {
		data.Duration = 5 * time.Minute
		return nil
	}

	w.buckets = data.Buckets
	w.size = data.Size
	w.duration = data.Duration
	w.lastTime = data.LastTime
	w.cursor = data.Cursor
	w.lastUpdate = data.LastUpdate

	return nil
}

// TimeWindowData 表示时间窗口中的数据点
type TimeWindowData struct {
	Time   time.Time `json:"time"`   // 数据时间点
	Values []float64 `json:"values"` // 该时间点的所有值
}

// GetData 返回时间窗口中的所有数据
func (w *TimeWindow) GetData() []TimeWindowData {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.rotate(time.Now())

	now := time.Now()
	result := make([]TimeWindowData, w.size)

	for i := 0; i < w.size; i++ {
		idx := (w.cursor - i + w.size) % w.size
		bucketTime := now.Add(-time.Duration(i) * w.duration)

		// 将单个值包装在切片中保持兼容性
		values := []float64{w.buckets[idx]}

		result[i] = TimeWindowData{
			Time:   bucketTime,
			Values: values,
		}
	}

	return result
}

// GetLatestValue 返回最新的值
func (w *TimeWindow) GetLatestValue() (float64, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.buckets[w.cursor], true
}
