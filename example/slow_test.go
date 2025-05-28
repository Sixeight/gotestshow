// ABOUTME: 実行時間が長いテストを含むテストファイルです。
// ABOUTME: gotestshowのリアルタイム表示機能を確認するために使用します。

package example

import (
	"testing"
	"time"
)

func TestSlowOperation1(t *testing.T) {
	t.Log("Starting slow operation 1...")
	time.Sleep(800 * time.Millisecond)
	// このテストは成功します
}

func TestSlowOperation2(t *testing.T) {
	t.Log("Starting slow operation 2...")
	time.Sleep(600 * time.Millisecond)
	// このテストも成功します
}

func TestSlowOperation3(t *testing.T) {
	t.Log("Starting slow operation 3...")
	time.Sleep(1 * time.Second)
	// このテストは失敗します
	t.Error("Intentional failure in slow operation 3")
}

func TestParallelSlow(t *testing.T) {
	t.Run("parallel1", func(t *testing.T) {
		t.Parallel()
		time.Sleep(700 * time.Millisecond)
	})
	t.Run("parallel2", func(t *testing.T) {
		t.Parallel()
		time.Sleep(700 * time.Millisecond)
	})
	t.Run("parallel3", func(t *testing.T) {
		t.Parallel()
		time.Sleep(700 * time.Millisecond)
	})
}