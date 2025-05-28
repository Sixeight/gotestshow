// ABOUTME: コンパイルエラーのあるパッケージのテストです。
// ABOUTME: このテストは実行されることはありません。

package broken

import "testing"

func TestBroken(t *testing.T) {
	// このテストは実行されません
	t.Log("This test will never run")
}