// ABOUTME: テスト用のサンプル数学関数を提供するパッケージです。
// ABOUTME: gotestshowの動作確認用に、成功・失敗・スキップするテストケースを含みます。

package example

// Add returns the sum of two integers
func Add(a, b int) int {
	return a + b
}

// Subtract returns the difference of two integers  
func Subtract(a, b int) int {
	return a - b
}

// Multiply returns the product of two integers
func Multiply(a, b int) int {
	// バグ: 意図的に間違った実装
	return a + b 
}

// Divide returns the division of two integers
func Divide(a, b int) (int, error) {
	if b == 0 {
		return 0, nil // バグ: エラーを返すべき
	}
	return a / b, nil
}