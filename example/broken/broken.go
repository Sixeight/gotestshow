// ABOUTME: コンパイルエラーを含むファイルです。
// ABOUTME: パッケージレベルの失敗をテストするために使用します。

package broken

// この関数は意図的にコンパイルエラーになります
func BrokenFunction() {
	// 未定義の変数を参照
	return undefinedVariable
}