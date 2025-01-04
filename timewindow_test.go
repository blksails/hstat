package hstat

import (
	"testing"
	"time"
)

func TestTimeWindow_Basic(t *testing.T) {
	w := NewTimeWindow(5, time.Second)

	// Test Append and Count
	w.Append(1.0)
	w.Append(2.0)
	if count := w.Count(); count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}

	// Test Sum
	if sum := w.Sum(); sum != 3.0 {
		t.Errorf("Expected sum 3.0, got %f", sum)
	}

	// Test Avg
	if avg := w.Avg(); avg != 1.5 {
		t.Errorf("Expected average 1.5, got %f", avg)
	}
}

func TestTimeWindow_Inc(t *testing.T) {
	w := NewTimeWindow(5, time.Second)

	w.Inc(1.0)
	w.Inc(2.0)
	if val, ok := w.GetLatestValue(); !ok || val != 3.0 {
		t.Errorf("Expected value 3.0, got %f", val)
	}
}

func TestTimeWindow_Dec(t *testing.T) {
	w := NewTimeWindow(5, time.Second)

	w.Inc(5.0)
	w.Dec(2.0)
	if val, ok := w.GetLatestValue(); !ok || val != 3.0 {
		t.Errorf("Expected value 3.0, got %f", val)
	}
}

func TestTimeWindow_Reset(t *testing.T) {
	w := NewTimeWindow(5, time.Second)

	w.Inc(5.0)
	w.Reset(2.0)
	if val, ok := w.GetLatestValue(); !ok || val != 2.0 {
		t.Errorf("Expected value 2.0, got %f", val)
	}
}

func TestTimeWindow_GetData(t *testing.T) {
	w := NewTimeWindow(3, time.Second)
	now := time.Now()

	w.Append(1.0)
	w.Append(2.0)

	data := w.GetData()
	if len(data) != 3 {
		t.Errorf("Expected 3 data points, got %d", len(data))
	}

	if len(data[0].Values) != 2 {
		t.Errorf("Expected 2 values in current bucket, got %d", len(data[0].Values))
	}

	if data[0].Time.Before(now) {
		t.Error("Expected first data point time to be after or equal to start time")
	}
}

func TestTimeWindow_Rotation(t *testing.T) {
	w := NewTimeWindow(2, time.Second)
	w.Append(1.0)

	// Wait for rotation
	time.Sleep(time.Second * 2)

	w.Append(2.0)
	if sum := w.Sum(); sum != 2.0 {
		t.Errorf("Expected sum 2.0 after rotation, got %f", sum)
	}
}

// Benchmarks

func BenchmarkTimeWindow_Append(b *testing.B) {
	w := NewTimeWindow(60, time.Second)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Append(float64(i))
	}
}

func BenchmarkTimeWindow_Inc(b *testing.B) {
	w := NewTimeWindow(60, time.Second)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Inc(1.0)
	}
}

func BenchmarkTimeWindow_GetData(b *testing.B) {
	w := NewTimeWindow(60, time.Second)
	for i := 0; i < 100; i++ {
		w.Append(float64(i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.GetData()
	}
}

func BenchmarkTimeWindow_Sum(b *testing.B) {
	w := NewTimeWindow(60, time.Second)
	for i := 0; i < 100; i++ {
		w.Append(float64(i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Sum()
	}
}

func BenchmarkTimeWindow_Avg(b *testing.B) {
	w := NewTimeWindow(60, time.Second)
	for i := 0; i < 100; i++ {
		w.Append(float64(i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Avg()
	}
}

func BenchmarkTimeWindow_PrintHistogram(b *testing.B) {
	w := NewTimeWindow(60, time.Second)
	for i := 0; i < 100; i++ {
		w.Append(float64(i))
	}
	opt := DefaultHistogramOption()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.PrintHistogram(opt)
	}
}
