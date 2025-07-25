package main

import (
	"archive/zip" // 添加此行以修复 zip 未定义的问题
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
)

type DownloadStatus struct {
	TotalBytes      int64
	DownloadedBytes int64
	SpeedMB         float64
	IsPaused        bool
	mu              sync.Mutex
	PauseChan       chan struct{}
	ResumeChan      chan struct{}
	DoneChan        chan struct{}
}

var downloadURLAg330 = []string{"https://files.zohopublic.com.cn/public/workdrive-public/download/dqd1m03114168cdbd47608183f4445c9b557c?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxry78-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D"}
var downloadURLUpdater = []string{"https://files.zohopublic.com.cn/public/workdrive-public/download/dqd1ma5b2ddd90a0647ed918d5ec5fe42de34?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrBKN-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D"} // 替换为实际的exe直连地址
var downloadURLLivery = []string{"https://files.zohopublic.com.cn/public/workdrive-public/download/dqd1m9be8830efd474542b4778f2486aaec09?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrBKQ-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://files.zohopublic.com.cn/public/workdrive-public/download/dqd1m1fd8be5eb91e4c0cbcfd43ae5afa3dcd?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrBKS-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://download.zoho.com.cn/v1/workdrive/download/dqd1m8a66561b090541c19e2d691735404943?x-cli-msg=%7B%22isFileOwner%22%3Atrue%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://download.zoho.com.cn/v1/workdrive/download/dqd1m4307c6d8bd27413ebb17399d293dfd23?x-cli-msg=%7B%22isFileOwner%22%3Atrue%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://download.zoho.com.cn/v1/workdrive/download/dqd1m9dbf164690ce4b9db3588720ed092623?x-cli-msg=%7B%22isFileOwner%22%3Atrue%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://download.zoho.com.cn/v1/workdrive/download/dqd1m42697a63df1e4d999eb28f11153c56c1?x-cli-msg=%7B%22isFileOwner%22%3Atrue%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://download.zoho.com.cn/v1/workdrive/download/dqd1m907a1ecfb3fe40158eca76ccc2088335?x-cli-msg=%7B%22isFileOwner%22%3Atrue%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://download.zoho.com.cn/v1/workdrive/download/dqd1m8971d4b509da44378dce876ee5921b2e?x-cli-msg=%7B%22isFileOwner%22%3Atrue%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://download.zoho.com.cn/v1/workdrive/download/dqd1m490f534324b14ee4b193103325d80429?x-cli-msg=%7B%22isFileOwner%22%3Atrue%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://download.zoho.com.cn/v1/workdrive/download/dqd1m21bb46a30bef464685a2629c226436ab?x-cli-msg=%7B%22isFileOwner%22%3Atrue%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://download.zoho.com.cn/v1/workdrive/download/dqd1m4be49eaa32634a40921078fdea3a35c1?x-cli-msg=%7B%22isFileOwner%22%3Atrue%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://download.zoho.com.cn/v1/workdrive/download/dqd1m4a3e8f7908ad4f18afe6df95a72f4bd3?x-cli-msg=%7B%22isFileOwner%22%3Atrue%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://download.zoho.com.cn/v1/workdrive/download/dqd1m97bce1d2e60d491f92d3e200a08aeee0?x-cli-msg=%7B%22isFileOwner%22%3Atrue%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://download.zoho.com.cn/v1/workdrive/download/dqd1m9c6bed1728fa4a2eb34a1974bf7fa88f?x-cli-msg=%7B%22isFileOwner%22%3Atrue%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://download.zoho.com.cn/v1/workdrive/download/dqd1m3504679b19e94bfbaceb3095c694ca04?x-cli-msg=%7B%22isFileOwner%22%3Atrue%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://download.zoho.com.cn/v1/workdrive/download/dqd1mb3d70c791f2d4cd7ba98a4dba3993c92?x-cli-msg=%7B%22isFileOwner%22%3Atrue%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://download.zoho.com.cn/v1/workdrive/download/dqd1m70b9ddaa970a43de8eb9ae1bce14fa05?x-cli-msg=%7B%22isFileOwner%22%3Atrue%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://download.zoho.com.cn/v1/workdrive/download/dqd1mc5f2ba434a8543e9970f4f7c02d664ad?x-cli-msg=%7B%22isFileOwner%22%3Atrue%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://download.zoho.com.cn/v1/workdrive/download/dqd1m2a6a889b6ef84760938575ba8a5046d7?x-cli-msg=%7B%22isFileOwner%22%3Atrue%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://download.zoho.com.cn/v1/workdrive/download/dqd1mb06e369077cf4e9e910ece9647eda973?x-cli-msg=%7B%22isFileOwner%22%3Atrue%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://download.zoho.com.cn/v1/workdrive/download/dqd1m39e9f30085924b1b84e7aea5f007283d?x-cli-msg=%7B%22isFileOwner%22%3Atrue%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://download.zoho.com.cn/v1/workdrive/download/dqd1ma4a294be1e764379b18297197ee74425?x-cli-msg=%7B%22isFileOwner%22%3Atrue%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://download.zoho.com.cn/v1/workdrive/download/dqd1m07a8da978a514d54bf2c93fe5564fb9e?x-cli-msg=%7B%22isFileOwner%22%3Atrue%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D"}

var xpPath string // 添加全局变量 xpPath

func main() {
	// 1. 尝试读取配置文件
	var err error
	xpPath, err = readXP12Path() // 使用全局变量 xpPath
	if err != nil || xpPath == "" {
		for {
			fmt.Print("\n请输入X-Plane 12根目录: ")
			reader := bufio.NewReader(os.Stdin)
			input, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("输入错误，请重试")
				continue
			}
			xpPath = strings.TrimSpace(input)

			// 3. 验证目录有效性（复用case1的检查）
			if valid, missing := validateXPlaneDirectory(xpPath); !valid {
				fmt.Println("目录验证失败，缺少以下文件/目录:")
				for _, item := range missing {
					fmt.Println("-", item)
				}
				continue
			}

			// 4. 验证通过后保存配置
			if err := writeXP12Path(xpPath); err != nil {
				fmt.Println("警告：无法保存配置，下次仍需手动输入")
			}
			break
		}
	} else {
		// 5. 如果配置文件存在，直接验证目录
		if valid, _ := validateXPlaneDirectory(xpPath); !valid {
			fmt.Println("配置中的X-Plane目录已失效，请重新输入")
			configPath, err := getConfigPath() // 获取配置文件路径
			if err != nil {
				fmt.Println("无法获取配置文件路径:", err)
			} else {
				os.Remove(configPath) // 删除无效配置
			}
			main() // 重新开始
			return
		}
	}
	// 主循环，保持程序运行直到用户选择退出
	for {
		// 显示欢迎信息和操作选项
		fmt.Println("\n欢迎使用AeroGennis Airbus A330-300机模/涂装安装工具")
		fmt.Printf("Version: 0.4 Preview\n")
		fmt.Println("Github仓库链接：https://github.com/CCA7386/Golang-XPlane12-AeroGennis-Updater/")
		fmt.Println("请选择你要的操作。")
		fmt.Println("1. 安装/更新机模")
		fmt.Println("2. 安装涂装")
		fmt.Println("3. 退出")
		fmt.Println("4. 更新此应用程序（直接下载exe）")
		fmt.Println("更新机模与安装机模的代码相同，但是更新机模只会替换已经安装的同名旧文件，不会影响已安装涂装。")
		fmt.Print("请输入你的选择（1/2/3/4）：")

		// 使用统一的输入方法
		choice, err := readInput()
		if err != nil {
			fmt.Println("输入错误:", err)
			continue
		}

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
	// 直接读取全局变量 xpPath（已在main中验证过）
	fmt.Println("\n开始安装机模到:", xpPath)

	tmpDir, err := os.MkdirTemp("", "xplane_aircraft_*")
	if err != nil {
		fmt.Println("创建临时目录失败:", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	zipPath := filepath.Join(tmpDir, "aircraft.zip")
	fmt.Println("正在下载机模包...")

	// 修改这一行，使用新的下载函数
	if err := downloadFileWithProgress(downloadURLAg330[0], zipPath); err != nil {
		fmt.Println("\n下载机模包失败:", err)
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

	nextChoice, err := readInput()
	if err != nil {
		fmt.Println("输入错误:", err)
		return
	}

	if nextChoice == "2" {
		fmt.Println("\n感谢使用，程序即将退出")
		os.Exit(0)
	}
	// 默认返回主菜单
}

func handleLiveryInstall() {
	for {
		fmt.Println("\n你选择了安装涂装")
		fmt.Println("\n欢迎使用AeroGennis Airbus A330-300涂装安装工具")
		fmt.Println("请确保你已经安装了AeroGennis Airbus A330-300机模")
		fmt.Println("并且机模已正确放置在X-Plane 12的Aircraft目录下")
		fmt.Println("如果你还没有安装机模，请先选择选项1进行安装")
		fmt.Println("\n当前可用涂装：")
		fmt.Println("1. EVA Air A330-300 Livery 3-Pack EVA Air B-16338，EVA Air B-16336 (2015-2020)，EVA Air B-16332 (Joyful Dream)")
		fmt.Println("原链接：https://x-plane.to/file/1563/eva-air-laminar-a330-300-livery-three-pack")
		fmt.Println("2.A330-300 Russia-special flight squadron ")
		fmt.Println("原链接：https://x-plane.to/file/1384/laminar-a330-300-russia-special-flight-squadron")
		fmt.Println("3.Airbus A330-300 Singapore Airlines Star alliance livery")
		fmt.Println("原链接：https://x-plane.to/file/1363/airbus-a330-300-singapore-airlines-star-alliance-livery")
		fmt.Println("4.Air Transat Airbus A330-300")
		fmt.Println("原链接：https://x-plane.to/file/1063/air-transat-airbus-a330-300-laminar-xplane-12")
		fmt.Println("5.Discover Airlines A330 D-AIKA")
		fmt.Println("原链接：https://x-plane.to/file/1042/discover-airlines-a330-d-aika")
		fmt.Println("6.A330-300 | Turkish Airlines TC-JNJ")
		fmt.Println("原链接：https://x-plane.to/file/1008/laminar-research-a330-300-turkish-airlines-4k-az83")
		fmt.Println("7.A330-300 Lovehansa (Pride) Repaint")
		fmt.Println("原链接： https://x-plane.to/file/850/a330-300-lovehansa-pride-repaint")
		fmt.Println("8.Star Alliance, Scandinavian Airlines [SE-REF] [8K]")
		fmt.Println("原链接：https://x-plane.to/file/783/star-alliance-scandinavian-airlines-se-ref-8k-laminar-a330-300")
		fmt.Println("9. Star Alliance, Thai Airways International [HS-TBD] [8K]")
		fmt.Println("原链接：https://x-plane.to/file/780/star-alliance-thai-airways-international-hs-tbd-8k-laminar-a330-300")
		fmt.Println("10. Laminar Research Airbus A330-300 | American Airlines")
		fmt.Println("原链接：https://x-plane.to/file/410/laminar-research-airbus-a330-300-american-airlines")
		fmt.Println("11. X-Plane 12 - Default A330 Finnair (OH-LTS)")
		fmt.Println("原链接：https://x-plane.to/file/648/x-plane-12-default-a330-finnair-oh-lts")
		fmt.Println("12. A330-300 Turkish Airlines UEFA Champions League 2023 (TC-JNM)")
		fmt.Println("原链接：https://x-plane.to/file/549/a330-300-turkish-airlines-uefa-champions-league-2023-tc-jnm")
		fmt.Println("13. Laminar Research A330 - China Airlines B-18355")
		fmt.Println("原链接：https://x-plane.to/file/523/laminar-research-a330-china-airlines-b-18355")
		fmt.Println("14. AirAsia Girls' Frontline A330-300 (Default XP12)")
		fmt.Println("原链接：https://x-plane.to/file/345/airasia-girls-frontline-a330-300-default-xp12")
		fmt.Println("15. Dragon Airlines' Livery for Laminar Research A330")
		fmt.Println("原链接：https://x-plane.to/file/459/dragon-airlines-livery-for-laminar-research-a330")
		fmt.Println("16. Aeroflot RA73786 | Airbus A330 Laminar Livery")
		fmt.Println("原链接：https://x-plane.to/file/453/aeroflot-ra73786-airbus-a330-laminar-livery")
		fmt.Println("17. A330-300 | Air China B-5947")
		fmt.Println("原链接：https://x-plane.to/file/363/a330-300-air-china-b-5947")
		fmt.Println("18. HongKong Airlines B-LHA for Laminar Research A330")
		fmt.Println("原链接：https://x-plane.to/file/339/hongkong-airlines-b-lha-for-laminar-research-a330")
		fmt.Println("19. A330-300 China Eastern B-6II9")
		fmt.Println("原链接：https://x-plane.to/file/219/a330-300-china-eastern-b-6ii9")
		fmt.Println("20. Lufthansa 2-Pack for Laminar A330-300")
		fmt.Println("原链接：https://x-plane.to/file/186/lufthansa-2-pack-for-laminar-a330-300")
		fmt.Println("21. KLM (PH-AKA)")
		fmt.Println("原链接：https://x-plane.to/file/111/klm-ph-aka")
		fmt.Println("22. Cathay Dragon (B-HLE)")
		fmt.Println("原链接：https://x-plane.to/file/13/cathay-dragon-b-hle")
		fmt.Println("23. Lufthansa (D-AIKI)")
		fmt.Println("原链接：https://x-plane.to/file/10/lufthansa-d-aiki")
		fmt.Println("请选择你要安装的涂装序号（1-23）")
		fmt.Println("请选择你要安装的涂装序号（1-23）")
		fmt.Println("\n例如安装7.A330-300 Lovehansa (Pride) Repaint，就输入7")
		fmt.Print("请输入序号: ")

		input, err := readInput()
		if err != nil {
			fmt.Println("输入错误:", err)
			continue
		}

		LiveryInstallChoose, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("请输入有效的数字")
			continue
		}

		if LiveryInstallChoose < 1 || LiveryInstallChoose > 23 {
			fmt.Println("输入序号无效，请重新输入")
			continue
		}

		liveryInstaller(downloadURLLivery, LiveryInstallChoose-1)

		fmt.Println("1. 返回主菜单")
		fmt.Println("2. 退出程序")
		fmt.Println("3. 继续安装涂装")
		fmt.Print("请输入选择（1/2/3）：")

		nextChoice, err := readInput()
		if err != nil {
			fmt.Println("输入错误:", err)
			return
		}

		switch nextChoice {
		case "2":
			fmt.Println("\n感谢使用，程序即将退出")
			os.Exit(0)
		case "3":
			continue
		default:
			return
		}
	}
}

func handleExeUpdate() {
	// 处理程序更新逻辑
	fmt.Println("\n你选择了更新应用程序（直接下载exe）")
	fmt.Println("如果安装失败，请关闭windows Defender或其他杀毒软件，因为此文件没有数字签名，可能会被误报为病毒。")

	var savePath string
	for {
		fmt.Print("\n请输入要保存的目录路径：")
		input, err := readInput()
		if err != nil {
			fmt.Println("输入错误:", err)
			continue
		}

		savePath = strings.TrimSpace(input)
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
	if err := downloadFileWithProgress(downloadURLUpdater[0], exePath); err != nil {
		fmt.Printf("下载失败: %v\n", err)
		return
	}

	fmt.Printf("\n更新成功！新版本已保存到:\n%s\n", exePath)
	fmt.Println("请手动运行下载的exe文件完成更新")

	fmt.Println("\n请选择后续操作：")
	fmt.Println("1. 返回主菜单")
	fmt.Println("2. 退出程序")
	fmt.Print("请输入选择（1/2）：")

	// 修复：使用 bufio.NewReader 并清除输入缓冲区
	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n') // 清除残留换行符
	nextChoice, _ := reader.ReadString('\n')
	nextChoice = strings.TrimSpace(nextChoice)

	if nextChoice == "2" {
		fmt.Println("\n感谢使用，程序即将退出")
		os.Exit(0)
	}
	// 默认返回主菜单
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

func downloadFileWithProgress(url string, filepath string) error {
	// 创建HTTP请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	// 支持断点续传
	if info, err := os.Stat(filepath); err == nil {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", info.Size()))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 获取文件总大小
	totalBytes := resp.ContentLength
	if totalBytes == -1 {
		return fmt.Errorf("服务器未提供文件大小信息")
	}

	// 打开或创建文件
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// 设置文件指针位置
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		return err
	}

	// 初始化下载状态
	status := &DownloadStatus{
		TotalBytes: totalBytes,
		PauseChan:  make(chan struct{}),
		ResumeChan: make(chan struct{}),
		DoneChan:   make(chan struct{}),
	}

	// 创建进度条
	bar := progressbar.NewOptions64(
		totalBytes,
		progressbar.OptionSetDescription("下载中"),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(50),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			fmt.Println("\n下载完成!")
		}),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
	)

	// 启动控制协程
	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			select {
			case <-status.DoneChan:
				return
			default:
				fmt.Print("\n输入 'p' 暂停下载，'r' 继续下载: ")
				input, _ := reader.ReadString('\n')
				input = strings.TrimSpace(input)

				status.mu.Lock()
				switch input {
				case "p":
					if !status.IsPaused {
						status.IsPaused = true
						status.PauseChan <- struct{}{}
						fmt.Println("下载已暂停")
					}
				case "r":
					if status.IsPaused {
						status.IsPaused = false
						status.ResumeChan <- struct{}{}
						fmt.Println("下载已继续")
					}
				}
				status.mu.Unlock()
			}
		}
	}()

	// 下载协程
	errChan := make(chan error, 1)
	go func() {
		buf := make([]byte, 32*1024) // 32KB缓冲区
		var lastBytes int64
		var lastTime = time.Now()

		for {
			select {
			case <-status.PauseChan:
				<-status.ResumeChan
				lastTime = time.Now() // 重置计时器
				lastBytes = status.DownloadedBytes
			default:
				n, err := resp.Body.Read(buf)
				if n > 0 {
					// 写入文件
					if _, err := file.Write(buf[:n]); err != nil {
						errChan <- err
						return
					}

					// 更新状态
					status.mu.Lock()
					status.DownloadedBytes += int64(n)
					now := time.Now()
					elapsed := now.Sub(lastTime).Seconds()

					if elapsed > 0 {
						status.SpeedMB = float64(status.DownloadedBytes-lastBytes) / (1024 * 1024) / elapsed
						lastTime = now
						lastBytes = status.DownloadedBytes
					}

					// 更新进度条
					bar.Set64(status.DownloadedBytes)
					status.mu.Unlock()
				}

				if err != nil {
					if err != io.EOF {
						errChan <- err
					}
					close(status.DoneChan)
					return
				}
			}
		}
	}()

	// 等待下载完成或出错
	select {
	case err := <-errChan:
		return err
	case <-status.DoneChan:
		return nil
	}
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
func liveryInstaller(liveryURLs []string, chosenIndex int) {
	// 验证索引范围
	if chosenIndex < 0 || chosenIndex >= len(liveryURLs) {
		fmt.Printf("错误：无效的涂装序号 %d (有效范围: 1-%d)\n", chosenIndex+1, len(liveryURLs))
		return
	}

	// 查找Aerogennis机模目录
	aircraftDir, err := findAerogennisDir(xpPath)
	if err != nil {
		fmt.Println("错误:", err)
		fmt.Println("请确保已正确安装AeroGennis A330-300机模")
		return
	}

	// 创建涂装目录
	liveryDir := filepath.Join(aircraftDir, "liveries")
	if err := os.MkdirAll(liveryDir, 0755); err != nil {
		fmt.Println("创建涂装目录失败:", err)
		return
	}

	fmt.Printf("\n安装涂装 #%d 到目录: %s\n", chosenIndex+1, liveryDir)
	fmt.Printf("下载URL: %s\n", liveryURLs[chosenIndex])

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "xplane_livery_*")
	if err != nil {
		fmt.Println("创建临时目录失败:", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	// 下载涂装包（使用完整进度显示）
	zipPath := filepath.Join(tmpDir, "livery.zip")
	fmt.Println("\n正在下载涂装包...")
	if err := downloadFileWithProgress(liveryURLs[chosenIndex], zipPath); err != nil {
		fmt.Println("下载失败:", err)
		return
	}

	// 解压涂装文件（保持原始目录结构）
	fmt.Println("\n正在解压涂装文件...")
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		fmt.Println("解压失败:", err)
		return
	}
	defer r.Close()

	// 创建进度条（基于文件数量）
	bar := progressbar.NewOptions(len(r.File),
		progressbar.OptionSetDescription("解压中..."),
		progressbar.OptionShowCount(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	// 先计算总文件数用于进度条
	totalFiles := 0
	for _, f := range r.File {
		if !f.FileInfo().IsDir() {
			totalFiles++
		}
	}
	bar.ChangeMax(totalFiles)

	for _, f := range r.File {
		// 跳过Mac系统文件
		if strings.Contains(f.Name, "__MACOSX") {
			continue
		}

		// 构建目标路径（保持原始相对路径）
		relPath := strings.TrimPrefix(f.Name, "13/") // 移除压缩包顶层目录"13/"
		dstPath := filepath.Join(liveryDir, relPath)

		if f.FileInfo().IsDir() {
			// 创建目录
			if err := os.MkdirAll(dstPath, 0755); err != nil {
				fmt.Printf("\n创建目录失败: %s\n", dstPath)
				continue
			}
		} else {
			// 更新进度条
			bar.Add(1)

			// 确保目录存在
			if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
				fmt.Printf("\n创建目录失败: %s\n", filepath.Dir(dstPath))
				continue
			}

			// 提取文件
			if err := extractFile(f, dstPath); err != nil {
				fmt.Printf("\n文件提取失败: %s\n", dstPath)
				continue
			}
		}
	}
	bar.Finish()

	fmt.Printf("\n涂装 #%d 安装完成！\n", chosenIndex+1)
	fmt.Println("涂装文件已保存到:", liveryDir)
}

// extractZipToLiveryDir 解压zip文件到涂装目录

// 获取配置文件路径（放在exe同级目录）
// 获取配置文件路径（放在exe同级目录）
// 获取配置文件路径（放在exe同级目录）
func getConfigPath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exePath), "Ag330UpdaterConf.txt"), nil
}

// 读取配置文件中的X-Plane路径
func readXP12Path() (string, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		return "", nil // 文件不存在不算错误
	}
	if err != nil {
		return "", err // 其他读取错误
	}
	return strings.TrimSpace(string(data)), nil
}

// 写入X-Plane路径到配置文件
func writeXP12Path(path string) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, []byte(path), 0644)
}

// findAerogennisDir 在指定目录下查找包含"aerogennis"的文件夹（不区分大小写）
func findAerogennisDir(basePath string) (string, error) {
	// 需要检查的目录列表
	searchDirs := []string{
		filepath.Join(basePath, "Aircraft"),
		filepath.Join(basePath, "Laminar Research"),
	}

	for _, dir := range searchDirs {
		// 检查目录是否存在
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		// 读取目录内容
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		// 遍历目录项
		for _, entry := range entries {
			if entry.IsDir() {
				dirName := entry.Name()
				// 不区分大小写检查是否包含"aerogennis"
				if strings.Contains(strings.ToLower(dirName), "aerogennis") {
					return filepath.Join(dir, dirName), nil
				}
			}
		}
	}

	return "", fmt.Errorf("未找到包含'Aerogennis'的文件夹")
}
func readInput() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}
