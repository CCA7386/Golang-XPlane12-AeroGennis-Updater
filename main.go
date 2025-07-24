package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"archive/zip" // 添加此行以修复 zip 未定义的问题
)

var downloadURLAg330 = []string{"https://files.zohopublic.com.cn/public/workdrive-public/download/dqd1m03114168cdbd47608183f4445c9b557c?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxry78-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D"}
var downloadURLUpdater = []string{"https://files.zohopublic.com.cn/public/workdrive-public/download/dqd1ma5b2ddd90a0647ed918d5ec5fe42de34?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrBKN-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D"} // 替换为实际的exe直连地址

func main() {
	// 主循环，保持程序运行直到用户选择退出
	for {
		// 显示欢迎信息和操作选项
		fmt.Println("\n欢迎使用AeroGennis Airbus A330-300机模/涂装安装工具")
		fmt.Printf("Version: 0.2 Preview\n")
		fmt.Println("Github仓库链接：https://github.com/CCA7386/Golang-XPlane12-AeroGennis-Updater/")
		fmt.Println("请选择你要的操作。")
		fmt.Println("1. 安装/更新机模")
		fmt.Println("2. 安装涂装")
		fmt.Println("3. 退出")
		fmt.Println("4. 更新此应用程序（直接下载exe）")
		fmt.Println("更新机模与安装机模的代码相同，但是更新机模只会替换已经安装的同名旧文件，不会影响已安装涂装。")
		fmt.Print("请输入你的选择（1/2/3/4）：")

		var choice string
		fmt.Scanln(&choice)

		switch choice {
		case "1":
			handleAircraftInstall()
		case "2":
			handleLiveryInstall()
		case "3":
			fmt.Println("\n感谢使用，程序即将退出")
			return
		case "4":
			handleExeUpdate()
		default:
			fmt.Println("\n无效的选择，请重新输入")
		}
	}
}

func handleAircraftInstall() {
	// 处理机模安装逻辑
	var xpPath string
	for {
		fmt.Print("\n请输入你的X-plane12根目录：")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("读取输入时出错，请重新输入")
			continue
		}
		xpPath = strings.TrimSpace(input)
		fmt.Println("你输入的 X-Plane 12 根目录是：", xpPath)
		fmt.Println("正在检查目录是否正确")

		if valid, missing := validateXPlaneDirectory(xpPath); !valid {
			for _, item := range missing {
				if strings.HasSuffix(item, ".exe") {
					fmt.Println("缺少文件：X-Plane.exe")
				} else {
					fmt.Printf("缺少文件夹：%s\n", item)
				}
			}
			fmt.Println("你的X-Plane12可能已经损坏，请前往Steam或者Installer进行更新/完整性检查")
			fmt.Println("如果没有损坏，请重新输入xplane12根目录地址")
		} else {
			break
		}
	}

	fmt.Println("\n开始下载并安装机模...")
	tmpDir, err := os.MkdirTemp("", "xplane_aircraft_*")
	if err != nil {
		fmt.Println("创建临时目录失败:", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	zipPath := filepath.Join(tmpDir, "aircraft.zip")
	fmt.Println("正在下载机模包...")
	if err := downloadFile(downloadURLAg330[0], zipPath); err != nil {
		fmt.Println("下载机模包失败:", err)
		return
	}

	aircraftDir := filepath.Join(xpPath, "Aircraft")
	fmt.Println("正在解压机模文件到Aircraft目录...")
	if err := extractZipDirect(zipPath, aircraftDir); err != nil {
		fmt.Println("解压机模包失败:", err)
		return
	}

	fmt.Println("\n机模安装完成！文件已解压到Aircraft目录")
	fmt.Println("请在X-Plane 12中选择AeroGennis Airbus A330-300进行飞行")

	fmt.Println("\n请选择后续操作：")
	fmt.Println("1. 返回主菜单")
	fmt.Println("2. 退出程序")
	fmt.Print("请输入选择（1/2）：")

	var nextChoice string
	fmt.Scanln(&nextChoice)
	if nextChoice == "2" {
		fmt.Println("\n感谢使用，程序即将退出")
		os.Exit(0)
	}
}

func handleLiveryInstall() {
	// 处理涂装安装逻辑
	fmt.Println("\n你选择了安装涂装")
	fmt.Println("请确保你已经安装了AeroGennis Airbus A330-300机模")
	fmt.Println("并且机模已正确放置在X-Plane 12的Aircraft目录下")
	fmt.Println("如果你还没有安装机模，请先选择选项1进行安装")
	fmt.Println("\n当前可用涂装：")
	fmt.Println("1. 此功能正在开发中")

	fmt.Println("\n请选择后续操作：")
	fmt.Println("1. 返回主菜单")
	fmt.Println("2. 退出程序")
	fmt.Print("请输入选择（1/2）：")

	var nextChoice string
	fmt.Scanln(&nextChoice)
	if nextChoice == "2" {
		fmt.Println("\n感谢使用，程序即将退出")
		os.Exit(0)
	}
}

func handleExeUpdate() {
	// 处理程序更新逻辑
	fmt.Println("\n你选择了更新应用程序（直接下载exe）")
	fmt.Println("如果安装失败，请关闭windows Defender或其他杀毒软件，因为此文件没有数字签名，可能会被误报为病毒。")

	var savePath string
	for {
		fmt.Print("\n请输入要保存的目录路径：")
		fmt.Scanln(&savePath)
		if savePath != "" {
			break
		}
		fmt.Println("路径不能为空，请重新输入")
	}

	// 确保目录存在
	if _, err := os.Stat(savePath); os.IsNotExist(err) {
		fmt.Printf("\n目录 %s 不存在，正在创建...\n", savePath)
		if err := os.MkdirAll(savePath, 0755); err != nil {
			fmt.Printf("创建目录失败: %v\n", err)
			return
		}
	}

	// 下载exe文件
	exeName := "AeroGennis_Updater_New.exe"
	exePath := filepath.Join(savePath, exeName)
	
	fmt.Printf("\n正在从 %s 下载更新...\n", downloadURLUpdater[0])
	if err := downloadFile(downloadURLUpdater[0], exePath); err != nil {
		fmt.Printf("下载失败: %v\n", err)
		return
	}

	fmt.Printf("\n更新成功！新版本已保存到:\n%s\n", exePath)
	fmt.Println("请手动运行下载的exe文件完成更新")

	fmt.Println("\n请选择后续操作：")
	fmt.Println("1. 返回主菜单")
	fmt.Println("2. 退出程序")
	fmt.Print("请输入选择（1/2）：")

	var nextChoice string
	fmt.Scanln(&nextChoice)
	if nextChoice == "2" {
		fmt.Println("\n感谢使用，程序即将退出")
		os.Exit(0)
	}
}

func validateXPlaneDirectory(path string) (bool, []string) {
	// 验证X-Plane目录是否完整
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

	if !filepath.IsAbs(path) {
		return false, []string{"必须使用绝对路径！"}
	}

	cleanPath := filepath.Clean(path)
	if cleanPath != path {
		return false, []string{"路径包含非法字符"}
	}

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

func downloadFile(url string, filepath string) error {
	// 下载文件的实现
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
	if err != nil {
		return err
	}

	// 设置可执行权限（Unix-like系统）
	if err := os.Chmod(filepath, 0755); err != nil {
		fmt.Printf("警告: 无法设置可执行权限 (%v)\n", err)
	}
	return nil
}

func extractZipDirect(zipFile, aircraftDir string) error {
	// 解压zip文件到指定目录
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer r.Close()

	destRoot := filepath.Join(aircraftDir, "AeroGennis Airbus A330-300")

	for _, f := range r.File {
		relPath := f.Name
		dstPath := filepath.Join(destRoot, relPath)

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
	// 提取单个文件
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

func copyFile(src, dst string) error {
	// 复制文件
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}