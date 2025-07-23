package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	downloadURL = "https://files.zohopublic.com.cn/public/workdrive-public/download/dqd1m03114168cdbd47608183f4445c9b557c?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxry78-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D"
)

func main() {
	// 初始提示信息（完全保留原始格式）
	fmt.Println("您的目录应该有以下文件夹：“Aircraft”“Airfoils” “Custom Data” “Custom Scenery” “Global Scenery” “Instructions” Output“ ”Resources’文件夹和 ”X-Plane.exe“")

	var xpPath string
	for {
		// 用户输入部分（完全保留原始输出）
		fmt.Print("请输入你的X-plane12根目录：")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("读取输入时出错，请重新输入")
			continue
		}
		xpPath = strings.TrimSpace(input)
		fmt.Println("你输入的 X-Plane 12 根目录是：", xpPath)
		fmt.Println("正在检查目录是否正确")

		// 验证部分（增强安全性）
		if valid, missing := validateXPlaneDirectory(xpPath); !valid {
			// 完全保留原始错误输出格式
			for _, item := range missing {
				if strings.HasSuffix(item, ".exe") {
					fmt.Println("缺少文件：X-Plane.exe")
				} else {
					fmt.Printf("缺少文件夹：%s\n", item)
				}
			}
			fmt.Println("目录不正确，请重新输入。")
		} else {
			// 完全保留原始成功输出
			fmt.Println("目录检查通过，继续执行后续代码...")
			break
		}
	}

	// 新增的下载和解压功能
	fmt.Println("\n开始下载并安装涂装...")
	
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "xplane_livery_*")
	if err != nil {
		fmt.Println("创建临时目录失败:", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	// 下载涂装包
	zipPath := filepath.Join(tmpDir, "livery.zip")
	fmt.Println("正在下载涂装包...")
	if err := downloadFile(downloadURL, zipPath); err != nil {
		fmt.Println("下载涂装包失败:", err)
		return
	}

	// 解压到Aircraft目录（直接解压到Aircraft下，不创建子文件夹）
	aircraftDir := filepath.Join(xpPath, "Aircraft")
	fmt.Println("正在解压涂装文件到Aircraft目录...")
	if err := extractZipDirect(zipPath, aircraftDir); err != nil {
		fmt.Println("解压涂装包失败:", err)
		return
	}

	fmt.Println("\n涂装安装完成！文件已解压到Aircraft目录")
}

// 原始验证函数（保持不变）
func validateXPlaneDirectory(path string) (bool, []string) {
	requiredItems := []string{
		"Aircraft",
		"Airfoils",
		"Custom Data",
		"Custom Scenery",
		"Global Scenery",
		"Instructions",
		"Output",
		"Resources",
		"X-Plane.exe",
	}

	var missingItems []string

	// 先检查是否为合法路径
	if !filepath.IsAbs(path) {
		return false, []string{"必须使用绝对路径！"}
	}

	cleanPath := filepath.Clean(path)
	if cleanPath != path {
		return false, []string{"路径包含非法字符"}
	}

	// 检查每个必需项目
	for _, item := range requiredItems {
		fullPath := filepath.Join(cleanPath, item)
		
		if strings.HasSuffix(item, ".exe") {
			if info, err := os.Stat(fullPath); err != nil || info.IsDir() {
				missingItems = append(missingItems, item)
			}
		} else {
			if info, err := os.Stat(fullPath); err != nil || !info.IsDir() {
				missingItems = append(missingItems, item)
			}
		}
	}

	return len(missingItems) == 0, missingItems
}

// 新增的下载功能
func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// 修改后的解压函数：直接解压到目标目录，不保留压缩包的目录结构
func extractZipDirect(zipFile, aircraftDir string) error {
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer r.Close()

	// 设定解压目标根目录为指定名称
	destRoot := filepath.Join(aircraftDir, "AeroGennis Airbus A330-300")

	for _, f := range r.File {
		relPath := f.Name
		dstPath := filepath.Join(destRoot, relPath)

		// 防止路径穿越
		if !strings.HasPrefix(filepath.Clean(dstPath), filepath.Clean(destRoot)+string(os.PathSeparator)) &&
			filepath.Clean(dstPath) != filepath.Clean(destRoot) {
			return fmt.Errorf("检测到非法路径: %s", dstPath)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(dstPath, 0755); err != nil {
				return err
			}
			continue
		}

		// 确保父目录存在
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}

		if err := extractFile(f, dstPath); err != nil {
			return err
		}
	}

	return nil
}

func extractFile(f *zip.File, dstPath string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}