// ABOUTME: math.goの関数をテストするユニットテストです。
// ABOUTME: 成功・失敗・スキップの各種テストケースを含んでいます。

package example

import (
	"runtime"
	"testing"
	"time"
)

func TestAdd(t *testing.T) {
	tests := []struct {
		name string
		a, b int
		want int
	}{
		{"positive numbers", 2, 3, 5},
		{"negative numbers", -2, -3, -5},
		{"mixed numbers", 10, -5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			time.Sleep(300 * time.Millisecond) // テスト実行をシミュレート
			got := Add(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("Add(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestSubtract(t *testing.T) {
	time.Sleep(500 * time.Millisecond) // 長めのテスト
	result := Subtract(10, 3)
	if result != 7 {
		t.Errorf("Subtract(10, 3) = %d, want 7", result)
	}
}

func TestMultiply(t *testing.T) {
	// このテストは失敗します（意図的なバグ）
	time.Sleep(400 * time.Millisecond)
	result := Multiply(4, 5)
	if result != 20 {
		t.Errorf("Multiply(4, 5) = %d, want 20", result)
	}
}

func TestDivide(t *testing.T) {
	t.Run("normal division", func(t *testing.T) {
		time.Sleep(200 * time.Millisecond)
		result, err := Divide(10, 2)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != 5 {
			t.Errorf("Divide(10, 2) = %d, want 5", result)
		}
	})

	t.Run("divide by zero", func(t *testing.T) {
		// このテストも失敗します（エラーハンドリングのバグ）
		time.Sleep(300 * time.Millisecond)
		_, err := Divide(10, 0)
		if err == nil {
			t.Error("expected error for divide by zero, got nil")
		}
	})
}

func TestSkipped(t *testing.T) {
	time.Sleep(100 * time.Millisecond)
	if runtime.GOOS == "darwin" {
		t.Skip("Skipping this test on macOS")
	}
	t.Log("This test runs on non-macOS systems")
}