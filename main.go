// ABOUTME: gotestshowは`go test -json`の出力を解析して整形表示するCLIツールです。
// ABOUTME: 標準入力からJSONフォーマットのテスト結果を受け取り、リアルタイムで見やすい形式で表示します。

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// TestEvent represents a single test event from go test -json
type TestEvent struct {
	Time    time.Time `json:"Time"`
	Action  string    `json:"Action"`
	Package string    `json:"Package"`
	Test    string    `json:"Test"`
	Elapsed float64   `json:"Elapsed"`
	Output  string    `json:"Output"`
}

// TestResult holds the summary of a test
type TestResult struct {
	Package string
	Test    string
	Passed  bool
	Skipped bool
	Failed  bool
	Elapsed float64
	Output  []string
	Started bool
}

// PackageState tracks the state of tests in a package
type PackageState struct {
	Name    string
	Total   int
	Passed  int
	Failed  int
	Skipped int
	Running int
	Elapsed float64
}

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorGray   = "\033[90m"
	clearLine   = "\033[2K\r"
)

var spinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func getSpinner() string {
	// 100ms間隔でアニメーションフレームを切り替え
	now := time.Now().UnixNano() / int64(time.Millisecond)
	index := (now / 100) % int64(len(spinnerChars))
	return spinnerChars[index]
}

func main() {
	results := make(map[string]*TestResult)
	packages := make(map[string]*PackageState)
	var mu sync.RWMutex

	// 起動直後にプログレスバーを表示
	fmt.Printf("%s%s Initializing...%s", colorBlue, getSpinner(), colorReset)

	// 定期的な画面更新のためのgoroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				mu.RLock()
				// 常にプログレスバーを表示（コンパイル中やテスト実行中）
				displayProgress(packages)
				mu.RUnlock()
			}
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var event TestEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue
		}

		mu.Lock()
		// パッケージの初期化
		if _, exists := packages[event.Package]; !exists && event.Package != "" {
			packages[event.Package] = &PackageState{
				Name: event.Package,
			}
		}

		// テスト結果の追跡
		if event.Test != "" {
			key := fmt.Sprintf("%s/%s", event.Package, event.Test)
			if _, exists := results[key]; !exists {
				results[key] = &TestResult{
					Package: event.Package,
					Test:    event.Test,
					Output:  []string{},
				}
			}
			result := results[key]
			pkg := packages[event.Package]

			switch event.Action {
			case "run":
				result.Started = true
				pkg.Running++
				pkg.Total++
			case "output":
				result.Output = append(result.Output, event.Output)
			case "pass":
				result.Passed = true
				result.Elapsed = event.Elapsed
				pkg.Running--
				pkg.Passed++
				displayTestResult(result, true)
			case "fail":
				result.Failed = true
				result.Elapsed = event.Elapsed
				pkg.Running--
				pkg.Failed++
				displayTestResult(result, false)
			case "skip":
				result.Skipped = true
				result.Elapsed = event.Elapsed
				pkg.Running--
				pkg.Skipped++
				displayTestResult(result, true)
			}
		} else if event.Package != "" {
			// パッケージレベルのイベント
			pkg := packages[event.Package]
			switch event.Action {
			case "pass":
				pkg.Elapsed = event.Elapsed
			case "fail":
				pkg.Elapsed = event.Elapsed
				pkg.Failed++
				pkg.Total++
				// パッケージレベルの失敗を記録
				key := fmt.Sprintf("%s/[PACKAGE]", event.Package)
				results[key] = &TestResult{
					Package: event.Package,
					Test:    "[Package Build/Init Failed]",
					Failed:  true,
					Elapsed: event.Elapsed,
					Output:  []string{"Package failed to build or initialize\n"},
				}
				displayPackageFailure(event.Package)
			}
		}
		mu.Unlock()
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	// 最終結果の表示
	fmt.Println()
	displayFinalResults(packages, results)
}

func displayProgress(packages map[string]*PackageState) {
	fmt.Print(clearLine)

	totalTests := 0
	totalPassed := 0
	totalFailed := 0
	totalSkipped := 0
	totalRunning := 0

	for _, pkg := range packages {
		totalTests += pkg.Total
		totalPassed += pkg.Passed
		totalFailed += pkg.Failed
		totalSkipped += pkg.Skipped
		totalRunning += pkg.Running
	}

	// プログレスバーの表示
	animation := getSpinner()
	fmt.Printf("%s%s Running: %d | ✓ Passed: %d | ✗ Failed: %d | ⚡ Skipped: %d%s",
		colorBlue, animation, totalRunning, totalPassed, totalFailed, totalSkipped, colorReset)
}

func displayTestResult(result *TestResult, success bool) {
	fmt.Print(clearLine)

	if success {
		if result.Skipped {
			fmt.Printf("%s⚡ SKIP%s %s %s(%.2fs)%s\n",
				colorYellow, colorReset, result.Test, colorGray, result.Elapsed, colorReset)
		} else {
			fmt.Printf("%s✓ PASS%s %s %s(%.2fs)%s\n",
				colorGreen, colorReset, result.Test, colorGray, result.Elapsed, colorReset)
		}
	} else {
		fmt.Printf("%s✗ FAIL%s %s %s(%.2fs)%s\n",
			colorRed, colorReset, result.Test, colorGray, result.Elapsed, colorReset)

		// エラー出力の表示
		relevantOutput := extractRelevantOutput(result.Output)
		if len(relevantOutput) > 0 {
			for _, line := range relevantOutput {
				fmt.Printf("        %s%s%s", colorRed, line, colorReset)
			}
		}
	}
}

func extractRelevantOutput(output []string) []string {
	var relevant []string
	for _, line := range output {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" &&
			!strings.HasPrefix(trimmed, "===") &&
			!strings.HasPrefix(trimmed, "---") &&
			!strings.Contains(trimmed, "PASS:") &&
			!strings.Contains(trimmed, "FAIL:") {
			relevant = append(relevant, line)
		}
	}

	// 最後の5行を返す
	if len(relevant) > 5 {
		return relevant[len(relevant)-5:]
	}
	return relevant
}

func displayPackageFailure(packageName string) {
	fmt.Print(clearLine)
	fmt.Printf("%s✗ PACKAGE FAIL%s %s %s(build/init error)%s\n",
		colorRed, colorReset, packageName, colorGray, colorReset)
}

func displayFinalResults(packages map[string]*PackageState, results map[string]*TestResult) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("📊 Test Results Summary")
	fmt.Println(strings.Repeat("=", 50))

	totalPassed := 0
	totalFailed := 0
	totalSkipped := 0
	var totalElapsed float64
	exitCode := 0

	for pkgName, pkg := range packages {
		// テストが0件のパッケージはスキップ
		if pkg.Total == 0 {
			continue
		}

		totalPassed += pkg.Passed
		totalFailed += pkg.Failed
		totalSkipped += pkg.Skipped
		totalElapsed += pkg.Elapsed

		status := colorGreen + "✓ PASS" + colorReset
		if pkg.Failed > 0 {
			status = colorRed + "✗ FAIL" + colorReset
			exitCode = 1
		}

		fmt.Printf("\n%s %s %s(%.2fs)%s\n", status, pkgName, colorGray, pkg.Elapsed, colorReset)
		fmt.Printf("  Tests: %d | Passed: %d | Failed: %d | Skipped: %d\n",
			pkg.Total, pkg.Passed, pkg.Failed, pkg.Skipped)

		// 失敗したテストの詳細
		if pkg.Failed > 0 {
			fmt.Printf("\n  %sFailed Tests:%s\n", colorRed, colorReset)
			for _, result := range results {
				if result.Package == pkgName && result.Failed {
					if result.Test == "[Package Build/Init Failed]" {
						fmt.Printf("    %s✗ Package build/initialization failed%s\n",
							colorRed, colorReset)
					} else {
						fmt.Printf("    %s✗ %s%s %s(%.2fs)%s\n",
							colorRed, result.Test, colorReset, colorGray, result.Elapsed, colorReset)
					}
				}
			}
		}
	}

	// 全体のサマリー
	fmt.Println("\n" + strings.Repeat("-", 50))
	fmt.Printf("Total: %d tests, %s%d passed%s, %s%d failed%s, %s%d skipped%s %s(%.2fs)%s\n",
		totalPassed+totalFailed+totalSkipped,
		colorGreen, totalPassed, colorReset,
		colorRed, totalFailed, colorReset,
		colorYellow, totalSkipped, colorReset,
		colorGray, totalElapsed, colorReset)

	if exitCode != 0 {
		fmt.Printf("\n%s❌ Tests failed!%s\n", colorRed, colorReset)
		os.Exit(exitCode)
	} else {
		fmt.Printf("\n%s✨ All tests passed!%s\n", colorGreen, colorReset)
	}
}
