package daemon

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/w6xian/keeper/internal/pathx"
)

// EnsureApp 确保 app 已安装，未安装则自动下载
func EnsureApp() (string, error) {
	path := pathx.GetCurrentAbPath()
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	// 尝试系统 PATH
	if p, err := exec.LookPath("app"); err == nil {
		return p, nil
	}
	return path, download(path)
}

func download(dest string) error {
	url, err := downloadURL()
	if err != nil {
		return err
	}
	fmt.Printf("正在下载 app...\n")

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
	}

	// macOS 的 app 是 tgz 格式，需要解压
	if strings.HasSuffix(url, ".tgz") {
		return extractTgz(resp.Body, dest)
	}

	// Linux/Windows 是裸二进制，直接写入
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		os.Chmod(dest, 0755)
	}
	fmt.Printf("cloudflared 已下载到 %s\n", dest)
	return nil
}

func extractTgz(r io.Reader, dest string) error {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("解压失败: %w", err)
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("tgz 中未找到 cloudflared")
		}
		if err != nil {
			return fmt.Errorf("解压失败: %w", err)
		}
		if filepath.Base(hdr.Name) == "app" {
			f, err := os.Create(dest)
			if err != nil {
				return err
			}
			defer f.Close()
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}
			os.Chmod(dest, 0755)
			fmt.Printf("app 已下载到 %s\n", dest)
			return nil
		}
	}
}

func downloadURL() (string, error) {
	const base = "https://github.com/cloudflare/cloudflared/releases/latest/download/"
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "darwin/arm64":
		return base + "app-darwin-arm64.tgz", nil
	case "darwin/amd64":
		return base + "app-darwin-amd64.tgz", nil
	case "linux/amd64":
		return base + "app-linux-amd64", nil
	case "linux/arm64":
		return base + "app-linux-arm64", nil
	case "windows/amd64":
		return base + "app-windows-amd64.exe", nil
	case "windows/arm64":
		return base + "app-windows-arm64.exe", nil
	default:
		return "", fmt.Errorf("不支持的平台: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
}
