// Package output 负责 ikuai-cli 的结果格式化输出（表格 / JSON / 单位换算）。
package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// HumanBytes 将字节数格式化为人类可读字符串。
func HumanBytes(n uint64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := uint64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

// HumanSpeed 将速率（byte/s）格式化。
func HumanSpeed(v float64) string {
	rate := v * 8 // 转 bit
	const unit = 1000.0
	if rate < unit {
		return fmt.Sprintf("%.0f bps", rate)
	}
	div, exp := unit, 0
	for r := rate / unit; r >= unit; r /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cbps", rate/div, "kMGTPE"[exp])
}

// PrintJSON 以 JSON 输出任意对象。
func PrintJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// Table 渲染对齐文本表格。
func Table(headers []string, rows [][]string) {
	if len(rows) == 0 {
		fmt.Println("(无数据)")
		return
	}
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, r := range rows {
		for i, c := range r {
			if i < len(widths) && len(c) > widths[i] {
				widths[i] = len(c)
			}
		}
	}
	sep := make([]string, len(widths))
	for i, w := range widths {
		sep[i] = strings.Repeat("-", w)
	}
	fmt.Println("| " + joinRow(headers, widths) + " |")
	fmt.Println("| " + strings.Join(sep, " | ") + " |")
	for _, r := range rows {
		fmt.Println("| " + joinRow(r, widths) + " |")
	}
}

func joinRow(row []string, widths []int) string {
	cells := make([]string, len(row))
	for i, c := range row {
		w := 0
		if i < len(widths) {
			w = widths[i]
		}
		cells[i] = pad(c, w)
	}
	return strings.Join(cells, " | ")
}

func pad(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

// KV 以 key: value 形式输出单行。
func KV(key, val string) {
	fmt.Printf("%-14s: %s\n", key, val)
}
