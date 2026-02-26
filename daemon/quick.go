package daemon

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
)

// StartQuick 启动免域名模式（前台运行，Ctrl+C 退出）
func StartQuick(port string) error {
	binPath, err := EnsureApp()
	if err != nil {
		return err
	}
	if Running() {
		return fmt.Errorf("app 已在运行，请先执行 app down")
	}

	cmd := exec.Command(binPath, "tunnel", "--url", "http://localhost:"+port)

	// 捕获 stderr 提取随机域名
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 app 失败: %w", err)
	}

	// 后台读取 stderr，提取域名并转发输出
	go scanForURL(stderr)

	// 捕获 Ctrl+C 优雅退出
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case <-sig:
		stopChildProcess(cmd)
		<-done
	case err := <-done:
		if err != nil {
			return fmt.Errorf("app 异常退出: %w", err)
		}
	}
	return nil
}

func scanForURL(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		// app 输出格式: ... https://xxx.trycloudflare.com ...
		if strings.Contains(line, "trycloudflare.com") {
			url := extractURL(line)
			if url != "" {
				fmt.Printf("\n✔ 隧道已启动: %s\n\n", url)
			}
		}
		fmt.Fprintln(os.Stderr, line)
	}
}

func extractURL(line string) string {
	for _, part := range strings.Fields(line) {
		if strings.Contains(part, "trycloudflare.com") && strings.HasPrefix(part, "http") {
			return part
		}
	}
	return ""
}
