package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// AppState 保存应用程序的状态，包括UI元素和配置。
type AppState struct {
	xpPath              string            // X-Plane 12 的根路径
	ag330Path           string            // AG330 飞机的具体路径
	isAircraftInstalled bool              // 飞机是否已安装的标志
	language            string            // UI语言
	translations        map[string]string // 翻译文本映射
	mainWindow          fyne.Window
	statusLabel         *widget.Label
	progressBar         *widget.ProgressBar
	installAircraftBtn  *widget.Button
	installLiveryBtn    *widget.Button
	updateExeBtn        *widget.Button
	liveryList          *widget.List
	selectedLiveryIndex int
}

// 全局下载URL，与原始代码相同。
var downloadURLAg330 = []string{"https://files.zohopublic.com.cn/public/workdrive-public/download/dqd1m03114168cdbd47608183f4445c9b557c?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxry78-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D"}
var downloadURLUpdater = []string{"https://files.zohopublic.com.cn/public/workdrive-public/download/dqd1ma5b2ddd90a0647ed918d5ec5fe42de34?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrBKN-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D"}
var downloadURLLivery = []string{
	"https://files.zohopublic.com.cn/public/workdrive-public/download/dqd1m9be8830efd474542b4778f2486aaec09?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrBKQ-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://files.zohopublic.com.cn/public/workdrive-public/download/kpgnr45890ce3669945b5aaae0525d2fab844?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrDPT-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://files.zohopublic.com.cn/public/workdrive-public/download/kpgnrc7e9a99452864f5798568d42f8ff43e5?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrDPU-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://files.zohopublic.com.cn/public/workdrive-public/download/kpgnr0e032d8a4d2743479fda1524b653f830?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrDPV-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://files.zohopublic.com.cn/public/workdrive-public/download/kpgnrfc472e366491472f864e22c8398bed78?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrDPW-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://files.zohopublic.com.cn/public/workdrive-public/download/kpgnr35985aaa1731470b8aece33ebcd16b19?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrDPX-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://files.zohopublic.com.cn/public/workdrive-public/download/kpgnr8bbe8bb2c16c45bbaddb25efe884f995?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrDPY-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://files.zohopublic.com.cn/public/workdrive-public/download/kpgnr732812bce962427183883b6d06173cb4?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrDPZ-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://files.zohopublic.com.cn/public/workdrive-public/download/kpgnrbe859972aa584e6bb907aee8bab3abab?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrDQ0-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://files.zohopublic.com.cn/public/workdrive-public/download/kpgnree3b7df05b6740228f736ef53be81ff6?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrDQ1-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://files.zohopublic.com.cn/public/workdrive-public/download/kpgnr2e206185db774b97a7bd6bc04e0bd4fb?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrDQ2-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://files.zohopublic.com.cn/public/workdrive-public/download/kpgnr2573f38071da427793b280182840d544?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrDQ3-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://files.zohopublic.com.cn/public/workdrive-public/download/kpgnr1ee22fa7e8ad4a9e87c91ecae6fad749?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrDQ4-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://files.zohopublic.com.cn/public/workdrive-public/download/kpgnra67b3ec773434a17b4a9579087bfb710?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrDQ5-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://files.zohopublic.com.cn/public/workdrive-public/download/kpgnr6f124fba0052410fbd1be5fadcba4345?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrDQ6-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://files.zohopublic.com.cn/public/workdrive-public/download/kpgnr86ff6964bd54478b962bbdad8d2496c1?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrDQ7-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://files.zohopublic.com.cn/public/workdrive-public/download/kpgnr81aeaa72d0f74d3480a994e966114d7e?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrDQ8-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://files.zohopublic.com.cn/public/workdrive-public/download/kpgnr68c5d7985c8c4eb281de3eb251c3d584?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrDQ9-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://files.zohopublic.com.cn/public/workdrive-public/download/kpgnrf4a06033bfa6497ba27ce5667b1aabe2?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrDQa-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://files.zohopublic.com.cn/public/workdrive-public/download/kpgnrf3ef582ce95a45a9b6c497d8cd59af0b?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrDQb-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://files.zohopublic.com.cn/public/workdrive-public/download/kpgnrbe6b455b0be14ccba92c095b6764fc51?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrDQc-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://files.zohopublic.com.cn/public/workdrive-public/download/kpgnredcb6179f8cc47a6ba0668f2e1f93307?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrDQd-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
	"https://files.zohopublic.com.cn/public/workdrive-public/download/kpgnr1e182d0650fa4d679b2c93ae4a5fbd46?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrDQe-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D",
}

// tr 是一个辅助函数，用于获取翻译后的字符串。
func (state *AppState) tr(key string, args ...interface{}) string {
	format, ok := state.translations[key]
	if !ok {
		return key // 如果找不到键，则返回键本身
	}
	if len(args) > 0 {
		return fmt.Sprintf(format, args...)
	}
	return format
}

// main 是 Fyne 应用程序的入口点。
// 重要提示: 为了在 Windows 上运行时隐藏控制台窗口，请使用以下命令进行编译:
// go build -ldflags "-H=windowsgui"
func main() {
	a := app.New()
	w := a.NewWindow("AeroGennis A330-300 Installer") // 这个标题将在选择语言后更新
	w.Resize(fyne.NewSize(700, 500))

	state := &AppState{
		mainWindow:          w,
		selectedLiveryIndex: -1,
	}

	// 读取配置文件中的路径和语言。
	xpPath, lang, ag330Path := readConfig()
	if lang == "" { // 没有配置文件或配置无效。
		w.SetContent(createLanguageSelectionUI(state))
	} else {
		// 配置文件存在，加载翻译。
		state.language = lang
		loadTranslations(state)
		w.SetTitle(state.tr("window_title"))

		// 检查 X-Plane 路径是否有效。
		if xpPath != "" {
			if valid, _ := validateXPlaneDirectory(xpPath); valid {
				state.xpPath = xpPath
				state.ag330Path = ag330Path      // 加载手动的AG330路径
				checkAircraftInstallation(state) // 检查飞机安装状态
			} else {
				// 路径无效，重置它。
				state.xpPath = ""
				writeConfig(state)
			}
		}

		// 显示相应的屏幕。
		if state.xpPath == "" {
			w.SetContent(createSetupUI(state))
		} else {
			w.SetContent(createMainUI(state))
		}
	}

	w.ShowAndRun()
}

// createLanguageSelectionUI 创建第一个屏幕，用于选择语言。
func createLanguageSelectionUI(state *AppState) fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Select Language / 语言选择", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	prompt := widget.NewLabel("Please select your language:")

	langOptions := []string{"English", "简体中文", "繁體中文", "Français", "Русский"}
	langCodes := map[string]string{
		"English":  "en-US",
		"简体中文":     "zh-CN",
		"繁體中文":     "zh-TW",
		"Français": "fr-FR",
		"Русский":  "ru-RU",
	}

	langSelect := widget.NewSelect(langOptions, func(selected string) {
		state.language = langCodes[selected]
	})
	langSelect.SetSelectedIndex(0) // 默认选择英语

	continueBtn := widget.NewButton("Continue", func() {
		if state.language == "" {
			state.language = "en-US" // 备用选项
		}
		// 加载翻译并进入设置界面。
		loadTranslations(state)
		if err := writeConfig(state); err != nil {
			dialog.ShowError(fmt.Errorf("%s: %w", state.tr("save_config_error"), err), state.mainWindow)
			return
		}
		state.mainWindow.SetTitle(state.tr("window_title"))
		state.mainWindow.SetContent(createSetupUI(state))
	})

	return container.NewVBox(
		title,
		prompt,
		langSelect,
		continueBtn,
	)
}

// createSetupUI 创建用于设置 X-Plane 12 路径的屏幕。
func createSetupUI(state *AppState) fyne.CanvasObject {
	pathEntry := widget.NewEntry()
	pathEntry.SetPlaceHolder(state.tr("path_placeholder"))

	browseBtn := widget.NewButton(state.tr("browse_button"), func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			pathEntry.SetText(uri.Path())
		}, state.mainWindow)
	})

	saveBtn := widget.NewButton(state.tr("save_continue_button"), func() {
		path := pathEntry.Text
		if valid, missing := validateXPlaneDirectory(path); !valid {
			msg := fmt.Sprintf("%s\n%s\n- %s", state.tr("invalid_xp_path_error"), state.tr("missing_items_label"), strings.Join(missing, "\n- "))
			dialog.ShowError(fmt.Errorf("%s", msg), state.mainWindow)
			return
		}

		state.xpPath = path
		if err := writeConfig(state); err != nil {
			dialog.ShowError(fmt.Errorf("%s: %w", state.tr("save_config_error"), err), state.mainWindow)
			return
		}

		// 在进入主界面前，检查飞机安装状态
		checkAircraftInstallation(state)
		state.mainWindow.SetContent(createMainUI(state))
	})

	return container.NewVBox(
		widget.NewLabel(state.tr("setup_welcome")),
		pathEntry,
		browseBtn,
		saveBtn,
	)
}

// createMainUI 创建应用程序的主选项卡界面。
func createMainUI(state *AppState) fyne.CanvasObject {
	state.statusLabel = widget.NewLabel(state.tr("status_ready"))
	state.progressBar = widget.NewProgressBar()

	tabs := container.NewAppTabs(
		container.NewTabItem(state.tr("tab_aircraft"), createAircraftTab(state)),
		container.NewTabItem(state.tr("tab_liveries"), createLiveryTab(state)),
		container.NewTabItem(state.tr("tab_update_app"), createUpdateTab(state)),
		container.NewTabItem(state.tr("tab_settings"), createSettingsTab(state)),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	return container.NewBorder(
		nil,
		container.NewVBox(state.statusLabel, state.progressBar), // 放置在底部
		nil,
		nil,
		tabs, // 放置在中央
	)
}

// createAircraftTab 构建 '飞机' 选项卡的内容。
func createAircraftTab(state *AppState) fyne.CanvasObject {
	var content fyne.CanvasObject

	if state.isAircraftInstalled {
		// 如果飞机已安装，显示更新/重装信息
		state.installAircraftBtn = widget.NewButton(state.tr("reinstall_button"), func() {
			handleAircraftInstall(state)
		})
		content = container.NewVBox(
			widget.NewLabelWithStyle(state.tr("aircraft_installed_title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewLabel(state.tr("aircraft_installed_desc")),
			state.installAircraftBtn,
		)
	} else {
		// 如果飞机未安装，显示标准安装信息
		state.installAircraftBtn = widget.NewButton(state.tr("install_aircraft_button"), func() {
			handleAircraftInstall(state)
		})
		content = container.NewVBox(
			widget.NewLabelWithStyle(state.tr("aircraft_tab_title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewLabel(state.tr("aircraft_tab_desc")),
			state.installAircraftBtn,
		)
	}
	return content
}

// createLiveryTab 构建 '涂装' 选项卡的内容。
func createLiveryTab(state *AppState) fyne.CanvasObject {
	// 涂装名称是专有名词，不应被翻译。
	var liveryNames = []string{
		"1. EVA Air A330-300 Livery 3-Pack",
		"2. A330-300 Russia-special flight squadron",
		"3. Airbus A330-300 Singapore Airlines Star alliance livery",
		"4. Air Transat Airbus A330-300",
		"5. Discover Airlines A330 D-AIKA",
		"6. A330-300 | Turkish Airlines TC-JNJ",
		"7. A330-300 Lovehansa (Pride) Repaint",
		"8. Star Alliance, Scandinavian Airlines [SE-REF] [8K]",
		"9. Star Alliance, Thai Airways International [HS-TBD] [8K]",
		"10. Laminar Research Airbus A330-300 | American Airlines",
		"11. X-Plane 12 - Default A330 Finnair (OH-LTS)",
		"12. A330-300 Turkish Airlines UEFA Champions League 2023 (TC-JNM)",
		"13. Laminar Research A330 - China Airlines B-18355",
		"14. AirAsia Girls' Frontline A330-300 (Default XP12)",
		"15. Dragon Airlines' Livery for Laminar Research A330",
		"16. Aeroflot RA73786 | Airbus A330 Laminar Livery",
		"17. A330-300 | Air China B-5947", "18. HongKong Airlines B-LHA for Laminar Research A330",
		"19. A330-300 China Eastern B-6II9",
		"20. Lufthansa 2-Pack for Laminar A330-300",
		"21. KLM (PH-AKA)",
		"22. Cathay Dragon (B-HLE)",
		"23. Lufthansa (D-AIKI)",
	}

	state.installLiveryBtn = widget.NewButton(state.tr("install_livery_button"), func() {
		if state.selectedLiveryIndex == -1 {
			dialog.ShowInformation(state.tr("no_livery_selected_title"), state.tr("no_livery_selected_message"), state.mainWindow)
			return
		}
		handleLiveryInstall(state, liveryNames)
	})
	state.installLiveryBtn.Disable()

	state.liveryList = widget.NewList(
		func() int { return len(liveryNames) },
		func() fyne.CanvasObject { return widget.NewLabel("Template") },
		func(i widget.ListItemID, o fyne.CanvasObject) { o.(*widget.Label).SetText(liveryNames[i]) },
	)

	state.liveryList.OnSelected = func(id widget.ListItemID) {
		state.selectedLiveryIndex = id
		state.installLiveryBtn.Enable()
		state.statusLabel.SetText(state.tr("livery_selected_status", liveryNames[id]))
	}

	state.liveryList.OnUnselected = func(id widget.ListItemID) {
		state.selectedLiveryIndex = -1
		state.installLiveryBtn.Disable()
		state.statusLabel.SetText(state.tr("no_livery_selected_status"))
	}

	return container.NewBorder(nil, state.installLiveryBtn, nil, nil, state.liveryList)
}

// createUpdateTab 构建 '更新程序' 选项卡的内容。
func createUpdateTab(state *AppState) fyne.CanvasObject {
	state.updateExeBtn = widget.NewButton(state.tr("download_latest_button"), func() {
		handleExeUpdate(state)
	})

	return container.NewVBox(
		widget.NewLabelWithStyle(state.tr("update_tab_title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabel(state.tr("update_tab_desc")),
		widget.NewLabel(state.tr("update_tab_warning")),
		state.updateExeBtn,
	)
}

// createSettingsTab 构建 '设置' 选项卡的内容。
func createSettingsTab(state *AppState) fyne.CanvasObject {
	pathLabel := widget.NewLabel(state.tr("current_path_label", state.xpPath))
	pathLabel.Wrapping = fyne.TextWrapWord

	changePathBtn := widget.NewButton(state.tr("change_path_button"), func() {
		state.mainWindow.SetContent(createSetupUI(state))
	})

	changeLangBtn := widget.NewButton(state.tr("change_language_button"), func() {
		state.mainWindow.SetContent(createLanguageSelectionUI(state))
	})

	// 手动设置 AG330 路径的 UI
	ag330PathEntry := widget.NewEntry()
	ag330PathEntry.SetText(state.ag330Path)
	ag330PathEntry.SetPlaceHolder(state.tr("manual_path_placeholder"))

	saveAg330PathBtn := widget.NewButton(state.tr("save_manual_path_button"), func() {
		manualPath := ag330PathEntry.Text
		// 简单的验证，检查路径是否存在且为目录
		if info, err := os.Stat(manualPath); err != nil || !info.IsDir() {
			dialog.ShowError(fmt.Errorf("%s", state.tr("manual_path_error")), state.mainWindow) // <<< 已修复
			return
		}

		state.ag330Path = manualPath
		if err := writeConfig(state); err != nil {
			dialog.ShowError(fmt.Errorf("%s: %w", state.tr("save_config_error"), err), state.mainWindow)
			return
		}

		// 保存后，重新检查安装状态并刷新主界面
		checkAircraftInstallation(state)
		state.mainWindow.SetContent(createMainUI(state))
		dialog.ShowInformation(state.tr("save_success_title"), state.tr("save_manual_path_success"), state.mainWindow)
	})

	return container.NewVBox(
		widget.NewLabelWithStyle(state.tr("settings_tab_title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		pathLabel,
		changePathBtn,
		changeLangBtn,
		widget.NewSeparator(),
		widget.NewLabelWithStyle(state.tr("manual_path_label"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		ag330PathEntry,
		saveAg330PathBtn,
	)
}

// handleAircraftInstall 管理飞机安装过程。
func handleAircraftInstall(state *AppState) {
	state.installAircraftBtn.Disable()
	state.progressBar.SetValue(0)

	go func() {
		defer state.installAircraftBtn.Enable()

		state.statusLabel.SetText(state.tr("status_creating_temp_dir"))
		tmpDir, err := os.MkdirTemp("", "xplane_aircraft_*")
		if err != nil {
			dialog.ShowError(fmt.Errorf("%s: %w", state.tr("temp_dir_error"), err), state.mainWindow)
			return
		}
		defer os.RemoveAll(tmpDir)

		zipPath := filepath.Join(tmpDir, "aircraft.zip")

		state.statusLabel.SetText(state.tr("status_downloading", state.tr("aircraft_package")))
		err = downloadFileWithProgress(downloadURLAg330[0], zipPath, state.progressBar, state.statusLabel, state)
		if err != nil {
			state.statusLabel.SetText(state.tr("download_failed_status"))
			dialog.ShowError(fmt.Errorf("%s: %w", state.tr("download_error", state.tr("aircraft_package")), err), state.mainWindow)
			return
		}

		aircraftDir := filepath.Join(state.xpPath, "Aircraft")
		state.statusLabel.SetText(state.tr("status_extracting", aircraftDir))
		err = extractZipGUI(zipPath, aircraftDir, true, state.progressBar, state.statusLabel, state)
		if err != nil {
			state.statusLabel.SetText(state.tr("extraction_failed_status"))
			dialog.ShowError(fmt.Errorf("%s: %w", state.tr("extraction_error", state.tr("aircraft_package")), err), state.mainWindow)
			return
		}

		// 安装完成后，再次检查状态并刷新UI
		checkAircraftInstallation(state)
		state.mainWindow.SetContent(createMainUI(state))

		state.statusLabel.SetText(state.tr("install_complete_status"))
		dialog.ShowInformation(state.tr("install_success_title"), state.tr("aircraft_install_success_message"), state.mainWindow)
	}()
}

// handleLiveryInstall 管理涂装安装过程。
func handleLiveryInstall(state *AppState, liveryNames []string) {
	// 检查飞机是否已安装，或路径是否已设置
	if !state.isAircraftInstalled || state.ag330Path == "" {
		dialog.ShowError(fmt.Errorf("%s", state.tr("find_aircraft_dir_error")), state.mainWindow) // <<< 已修复
		return
	}

	chosenIndex := state.selectedLiveryIndex
	state.installLiveryBtn.Disable()
	state.progressBar.SetValue(0)

	go func() {
		defer func() {
			state.installLiveryBtn.Enable()
			state.liveryList.UnselectAll()
		}()

		liveryDir := filepath.Join(state.ag330Path, "liveries")
		if err := os.MkdirAll(liveryDir, 0755); err != nil {
			dialog.ShowError(fmt.Errorf("%s: %w", state.tr("create_liveries_dir_error"), err), state.mainWindow)
			return
		}

		state.statusLabel.SetText(state.tr("status_creating_temp_dir"))
		tmpDir, err := os.MkdirTemp("", "xplane_livery_*")
		if err != nil {
			dialog.ShowError(fmt.Errorf("%s: %w", state.tr("temp_dir_error"), err), state.mainWindow)
			return
		}
		defer os.RemoveAll(tmpDir)

		zipPath := filepath.Join(tmpDir, "livery.zip")

		state.statusLabel.SetText(state.tr("status_downloading_livery", chosenIndex+1))
		err = downloadFileWithProgress(downloadURLLivery[chosenIndex], zipPath, state.progressBar, state.statusLabel, state)
		if err != nil {
			state.statusLabel.SetText(state.tr("download_failed_status"))
			dialog.ShowError(fmt.Errorf("%s: %w", state.tr("download_error", state.tr("livery_package")), err), state.mainWindow)
			return
		}

		state.statusLabel.SetText(state.tr("status_extracting_livery"))
		err = extractZipGUI(zipPath, liveryDir, false, state.progressBar, state.statusLabel, state)
		if err != nil {
			state.statusLabel.SetText(state.tr("extraction_failed_status"))
			dialog.ShowError(fmt.Errorf("%s: %w", state.tr("extraction_error", state.tr("livery_package")), err), state.mainWindow)
			return
		}

		state.statusLabel.SetText(state.tr("livery_install_complete_status"))
		dialog.ShowInformation(state.tr("install_success_title"), state.tr("livery_install_success_message", liveryNames[chosenIndex]), state.mainWindow)
	}()
}

// handleExeUpdate 管理应用程序更新过程。
func handleExeUpdate(state *AppState) {
	dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil || uri == nil {
			return
		}
		savePath := uri.Path()

		state.updateExeBtn.Disable()
		state.progressBar.SetValue(0)

		go func() {
			defer state.updateExeBtn.Enable()

			exeName := "AeroGennis_Updater_New.exe"
			exePath := filepath.Join(savePath, exeName)

			state.statusLabel.SetText(state.tr("status_downloading_update"))
			err := downloadFileWithProgress(downloadURLUpdater[0], exePath, state.progressBar, state.statusLabel, state)
			if err != nil {
				state.statusLabel.SetText(state.tr("download_failed_status"))
				dialog.ShowError(fmt.Errorf("%s: %w", state.tr("download_update_error"), err), state.mainWindow)
				return
			}

			state.statusLabel.SetText(state.tr("update_download_complete_status"))
			dialog.ShowInformation(state.tr("update_download_complete_title"), state.tr("update_download_complete_message", exePath), state.mainWindow)
		}()
	}, state.mainWindow)
}

// downloadFileWithProgress 下载文件并更新 Fyne UI 部件。
func downloadFileWithProgress(url, destPath string, bar *widget.ProgressBar, label *widget.Label, state *AppState) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	totalBytes := resp.ContentLength
	if totalBytes <= 0 {
		label.SetText(state.tr("download_no_progress"))
		bar.Max = 1
		bar.SetValue(0.5) // 不确定模式（用于进度条）
	} else {
		bar.Max = float64(totalBytes)
	}

	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()

	buf := make([]byte, 32*1024)
	var downloadedBytes int64
	startTime := time.Now()

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := file.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			downloadedBytes += int64(n)

			if totalBytes > 0 {
				bar.SetValue(float64(downloadedBytes))
				speed := float64(downloadedBytes) / time.Since(startTime).Seconds() / (1024 * 1024)
				label.SetText(state.tr("download_progress_label",
					float64(downloadedBytes)/(1024*1024),
					float64(totalBytes)/(1024*1024),
					speed))
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// extractZipGUI 解压 zip 文件并更新 Fyne UI 部件。
func extractZipGUI(zipFile, destDir string, isAircraft bool, bar *widget.ProgressBar, label *widget.Label, state *AppState) error {
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer r.Close()

	bar.Max = float64(len(r.File))
	bar.SetValue(0)

	destRoot := destDir
	if isAircraft {
		destRoot = filepath.Join(destDir, "AeroGennis Airbus A330-300")
	}

	for i, f := range r.File {
		bar.SetValue(float64(i + 1))
		label.SetText(state.tr("extract_progress_label", f.Name))

		fpath := filepath.Join(destRoot, f.Name)
		if !strings.HasPrefix(fpath, filepath.Clean(destRoot)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

// checkAircraftInstallation 检查飞机是否已正确安装（路径和大小）。
func checkAircraftInstallation(state *AppState) {
	// 默认设置为未安装
	state.isAircraftInstalled = false
	var finalPath string

	// 1. 检查是否有手动设置的路径
	if state.ag330Path != "" {
		if info, err := os.Stat(state.ag330Path); err == nil && info.IsDir() {
			finalPath = state.ag330Path
		}
	}

	// 2. 如果没有有效的手动路径，则自动搜索
	if finalPath == "" && state.xpPath != "" {
		foundPath, err := findAerogennisDir(state.xpPath)
		if err == nil {
			finalPath = foundPath
		}
	}

	// 3. 如果找到了路径（无论是手动还是自动），则检查大小
	if finalPath != "" {
		size, err := getDirSize(finalPath)
		if err == nil {
			// 1.3 GB in bytes = 1.3 * 1024 * 1024 * 1024
			const requiredSize = 1395864371
			if size > requiredSize {
				state.isAircraftInstalled = true
				state.ag330Path = finalPath // 确保状态中的路径是最终有效的路径
			}
		}
	}
}

// getDirSize 递归计算目录的总大小。
func getDirSize(path string) (int64, error) {
	var size int64
	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// validateXPlaneDirectory 检查给定路径是否为有效的 X-Plane 12 根目录。
func validateXPlaneDirectory(path string) (bool, []string) {
	requiredItems := []string{
		"Aircraft", "Custom Scenery", "Global Scenery", "Resources", "X-Plane.exe",
	}
	var missingItems []string

	if !filepath.IsAbs(path) {
		return false, []string{"Path must be an absolute directory path."}
	}

	for _, item := range requiredItems {
		fullPath := filepath.Join(path, item)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			missingItems = append(missingItems, item)
		}
	}

	return len(missingItems) == 0, missingItems
}

// findAerogennisDir 搜索飞机的主文件夹以安装涂装。
func findAerogennisDir(basePath string) (string, error) {
	aircraftPath := filepath.Join(basePath, "Aircraft")
	entries, err := os.ReadDir(aircraftPath)
	if err != nil {
		return "", fmt.Errorf("could not read Aircraft directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			if strings.Contains(strings.ToLower(entry.Name()), "aerogennis") {
				return filepath.Join(aircraftPath, entry.Name()), nil
			}
		}
	}

	return "", fmt.Errorf("no directory containing 'Aerogennis' found in '%s'", aircraftPath)
}

// getConfigPath 返回配置文件的路径。
func getConfigPath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exePath), "Ag330UpdaterConf.txt"), nil
}

// readConfig 从配置文件中读取 X-Plane 路径、语言和手动设置的AG330路径。
func readConfig() (xpPath, lang, ag330Path string) {
	configPath, err := getConfigPath()
	if err != nil {
		return "", "", ""
	}

	data, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		return "", "", "" // 文件不存在，这是首次运行的正常情况。
	}
	if err != nil {
		return "", "", ""
	}

	lines := strings.Split(string(data), "\n")

	if len(lines) >= 1 {
		xpPath = strings.TrimSpace(lines[0])
	}
	if len(lines) >= 2 {
		lang = strings.TrimSpace(lines[1])
	}
	if len(lines) >= 3 {
		ag330Path = strings.TrimSpace(lines[2])
	}

	return xpPath, lang, ag330Path
}

// writeConfig 将 X-Plane 路径、语言和手动AG330路径保存到配置文件中。
func writeConfig(state *AppState) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}
	content := fmt.Sprintf("%s\n%s\n%s", state.xpPath, state.language, state.ag330Path)
	return os.WriteFile(configPath, []byte(content), 0644)
}

// translations 存储了所有不同语言的 UI 字符串。
var translations = map[string]map[string]string{
	"en-US": {
		"window_title":                     "AeroGennis A330-300 Installer - v2025.7.26.20-Preview",
		"reinstall_button":                 "Check for Updates / Reinstall",
		"aircraft_installed_title":         "Installation Detected",
		"aircraft_installed_desc":          "The application has detected that AeroGennis A330-300 is already installed. You can check for updates or reinstall if needed.",
		"manual_path_label":                "Manual AG330 Path Override",
		"manual_path_placeholder":          "Optional: Enter full path to 'AeroGennis Airbus A330-300' folder",
		"save_manual_path_button":          "Save Manual Path",
		"manual_path_error":                "The provided path is invalid or does not exist. Please check and try again.",
		"save_success_title":               "Saved",
		"save_manual_path_success":         "Manual path has been saved successfully. The application has been refreshed.",
		"setup_welcome":                    "Welcome! Please select your X-Plane 12 root directory to begin.",
		"path_placeholder":                 "Enter or browse to your X-Plane 12 root directory...",
		"browse_button":                    "Browse...",
		"save_continue_button":             "Save and Continue",
		"invalid_xp_path_error":            "The selected directory is not a valid X-Plane 12 installation.",
		"missing_items_label":              "Missing items:",
		"save_config_error":                "Failed to save configuration",
		"status_ready":                     "Ready. Select an option.",
		"tab_aircraft":                     "Aircraft",
		"tab_liveries":                     "Liveries",
		"tab_update_app":                   "Update App",
		"tab_settings":                     "Settings",
		"aircraft_tab_title":               "Install or Update AeroGennis A330-300",
		"aircraft_tab_desc":                "This will download the latest version of the AeroGennis A330-300 and install it into your X-Plane 12/Aircraft directory.",
		"install_aircraft_button":          "Install AeroGennis A330-300",
		"install_livery_button":            "Install Selected Livery",
		"no_livery_selected_title":         "No Livery Selected",
		"no_livery_selected_message":       "Please select a livery from the list before installing.",
		"livery_selected_status":           "Selected: %s",
		"no_livery_selected_status":        "No livery selected.",
		"update_tab_title":                 "Update This Application",
		"update_tab_desc":                  "This will download the latest version of this installer (as a .exe file). You will need to close this application and run the new one manually.",
		"update_tab_warning":               "Note: This file is not digitally signed and may be flagged by antivirus software. This is a false positive.",
		"download_latest_button":           "Download Latest Application Version",
		"settings_tab_title":               "Application Settings",
		"current_path_label":               "Current X-Plane 12 Path: %s",
		"change_path_button":               "Change X-Plane 12 Directory",
		"change_language_button":           "Change Language",
		"status_creating_temp_dir":         "Creating temporary directory...",
		"temp_dir_error":                   "Failed to create temp directory",
		"status_downloading":               "Downloading %s...",
		"aircraft_package":                 "aircraft package",
		"livery_package":                   "livery",
		"download_failed_status":           "Download failed.",
		"download_error":                   "Failed to download %s",
		"status_extracting":                "Extracting files to %s...",
		"extraction_failed_status":         "Extraction failed.",
		"extraction_error":                 "Failed to extract %s package",
		"install_complete_status":          "Aircraft installation complete!",
		"install_success_title":            "Success",
		"aircraft_install_success_message": "AeroGennis A330-300 has been installed successfully!",
		"find_aircraft_dir_error":          "Could not find a valid AeroGennis A330 directory. Please install the aircraft first or set the path manually in Settings.",
		"create_liveries_dir_error":        "Failed to create liveries directory",
		"status_downloading_livery":        "Downloading livery #%d...",
		"status_extracting_livery":         "Extracting livery files...",
		"livery_install_complete_status":   "Livery installation complete!",
		"livery_install_success_message":   "Livery '%s' installed successfully!",
		"status_downloading_update":        "Downloading new application version...",
		"download_update_error":            "Failed to download update",
		"update_download_complete_status":  "Update downloaded successfully!",
		"update_download_complete_title":   "Download Complete",
		"update_download_complete_message": "New version saved to:\n%s\nPlease close this application and run the new one.",
		"download_progress_label":          "Downloading... %.2f / %.2f MB (%.2f MB/s)",
		"download_no_progress":             "Warning: Content length unknown. Progress will not be shown.",
		"extract_progress_label":           "Extracting: %s",
	},
	"zh-CN": {
		"window_title":                     "AeroGennis A330-300 安装程序 - v2025.7.26.20-Preview",
		"reinstall_button":                 "检查更新/重新安装",
		"aircraft_installed_title":         "检测到已安装",
		"aircraft_installed_desc":          "程序检测到 AeroGennis A330-300 已经安装。如果需要，您可以检查更新或重新安装。",
		"manual_path_label":                "手动覆盖AG330路径",
		"manual_path_placeholder":          "可选：输入 'AeroGennis Airbus A330-300' 文件夹的完整路径",
		"save_manual_path_button":          "保存手动路径",
		"manual_path_error":                "提供的路径无效或不存在。请检查后重试。",
		"save_success_title":               "已保存",
		"save_manual_path_success":         "手动路径已成功保存。应用程序已刷新。",
		"setup_welcome":                    "欢迎！请选择您的 X-Plane 12 根目录以开始。",
		"path_placeholder":                 "输入或浏览您的 X-Plane 12 根目录...",
		"browse_button":                    "浏览...",
		"save_continue_button":             "保存并继续",
		"invalid_xp_path_error":            "所选目录不是有效的 X-Plane 12 安装目录。",
		"missing_items_label":              "缺少项目:",
		"save_config_error":                "无法保存配置",
		"status_ready":                     "准备就绪。请选择一个选项。",
		"tab_aircraft":                     "飞机",
		"tab_liveries":                     "涂装",
		"tab_update_app":                   "更新程序",
		"tab_settings":                     "设置",
		"aircraft_tab_title":               "安装或更新 AeroGennis A330-300",
		"aircraft_tab_desc":                "这将下载最新版本的 AeroGennis A330-300 并将其安装到您的 X-Plane 12/Aircraft 目录中。",
		"install_aircraft_button":          "安装 AeroGennis A330-300",
		"install_livery_button":            "安装所选涂装",
		"no_livery_selected_title":         "未选择涂装",
		"no_livery_selected_message":       "请在安装前从列表中选择一个涂装。",
		"livery_selected_status":           "已选择: %s",
		"no_livery_selected_status":        "未选择涂装。",
		"update_tab_title":                 "更新此应用程序",
		"update_tab_desc":                  "这将下载此安装程序的最新版本（.exe 文件）。您需要关闭此应用程序并手动运行新程序。",
		"update_tab_warning":               "注意：此文件未经数字签名，可能会被杀毒软件标记。这是一个误报。",
		"download_latest_button":           "下载最新应用程序版本",
		"settings_tab_title":               "应用程序设置",
		"current_path_label":               "当前 X-Plane 12 路径: %s",
		"change_path_button":               "更改 X-Plane 12 目录",
		"change_language_button":           "更改语言",
		"status_creating_temp_dir":         "正在创建临时目录...",
		"temp_dir_error":                   "创建临时目录失败",
		"status_downloading":               "正在下载 %s...",
		"aircraft_package":                 "飞机包",
		"livery_package":                   "涂装",
		"download_failed_status":           "下载失败。",
		"download_error":                   "下载 %s 失败",
		"status_extracting":                "正在解压文件到 %s...",
		"extraction_failed_status":         "解压失败。",
		"extraction_error":                 "解压 %s 包失败",
		"install_complete_status":          "飞机安装完成！",
		"install_success_title":            "成功",
		"aircraft_install_success_message": "AeroGennis A330-300 已成功安装！",
		"find_aircraft_dir_error":          "未能找到有效的 AeroGennis A330 目录。请先安装飞机，或在设置中手动指定路径。",
		"create_liveries_dir_error":        "创建涂装目录失败",
		"status_downloading_livery":        "正在下载涂装 #%d...",
		"status_extracting_livery":         "正在解压涂装文件...",
		"livery_install_complete_status":   "涂装安装完成！",
		"livery_install_success_message":   "涂装 '%s' 已成功安装！",
		"status_downloading_update":        "正在下载新版应用程序...",
		"download_update_error":            "下载更新失败",
		"update_download_complete_status":  "更新下载成功！",
		"update_download_complete_title":   "下载完成",
		"update_download_complete_message": "新版本已保存至：\n%s\n请关闭此应用程序并运行新程序。",
		"download_progress_label":          "下载中... %.2f / %.2f MB (%.2f MB/s)",
		"download_no_progress":             "警告：内容长度未知。将不显示进度。",
		"extract_progress_label":           "正在解压: %s",
	},
	"zh-TW": {
		"window_title":                     "AeroGennis A330-300 安裝程式 - v2025.7.26.20-Preview",
		"reinstall_button":                 "檢查更新/重新安裝",
		"aircraft_installed_title":         "偵測到已安裝",
		"aircraft_installed_desc":          "應用程式偵測到 AeroGennis A330-300 已經安裝。如果需要，您可以檢查更新或重新安裝。",
		"manual_path_label":                "手動覆寫 AG330 路徑",
		"manual_path_placeholder":          "可選：輸入 'AeroGennis Airbus A330-300' 資料夾的完整路徑",
		"save_manual_path_button":          "儲存手動路徑",
		"manual_path_error":                "提供的路徑無效或不存在。請檢查後重試。",
		"save_success_title":               "已儲存",
		"save_manual_path_success":         "手動路徑已成功儲存。應用程式已重新整理。",
		"setup_welcome":                    "歡迎！請選擇您的 X-Plane 12 根目錄以開始。",
		"path_placeholder":                 "輸入或瀏覽您的 X-Plane 12 根目錄...",
		"browse_button":                    "瀏覽...",
		"save_continue_button":             "儲存並繼續",
		"invalid_xp_path_error":            "所選目錄不是有效的 X-Plane 12 安裝目錄。",
		"missing_items_label":              "缺少項目:",
		"save_config_error":                "無法儲存設定",
		"status_ready":                     "準備就緒。請選擇一個選項。",
		"tab_aircraft":                     "飛機",
		"tab_liveries":                     "塗裝",
		"tab_update_app":                   "更新程式",
		"tab_settings":                     "設定",
		"aircraft_tab_title":               "安裝或更新 AeroGennis A330-300",
		"aircraft_tab_desc":                "這將會下載最新版本的 AeroGennis A330-300 並將其安裝到您的 X-Plane 12/Aircraft 目錄中。",
		"install_aircraft_button":          "安裝 AeroGennis A330-300",
		"install_livery_button":            "安裝所選塗裝",
		"no_livery_selected_title":         "未選擇塗裝",
		"no_livery_selected_message":       "請在安裝前從列表中選擇一個塗裝。",
		"livery_selected_status":           "已選擇: %s",
		"no_livery_selected_status":        "未選擇塗裝。",
		"update_tab_title":                 "更新此應用程式",
		"update_tab_desc":                  "這將會下載此安裝程式的最新版本（.exe 檔案）。您需要關閉此應用程式並手動執行新程式。",
		"update_tab_warning":               "注意：此檔案未經數位簽章，可能會被防毒軟體標記。這是誤報。",
		"download_latest_button":           "下載最新應用程式版本",
		"settings_tab_title":               "應用程式設定",
		"current_path_label":               "目前 X-Plane 12 路徑: %s",
		"change_path_button":               "變更 X-Plane 12 目錄",
		"change_language_button":           "變更語言",
		"status_creating_temp_dir":         "正在建立暫存目錄...",
		"temp_dir_error":                   "建立暫存目錄失敗",
		"status_downloading":               "正在下載 %s...",
		"aircraft_package":                 "飛機套件",
		"livery_package":                   "塗裝",
		"download_failed_status":           "下載失敗。",
		"download_error":                   "下載 %s 失敗",
		"status_extracting":                "正在解壓縮檔案至 %s...",
		"extraction_failed_status":         "解壓縮失敗。",
		"extraction_error":                 "解壓縮 %s 套件失敗",
		"install_complete_status":          "飛機安裝完成！",
		"install_success_title":            "成功",
		"aircraft_install_success_message": "AeroGennis A330-300 已成功安裝！",
		"find_aircraft_dir_error":          "未能找到有效的 AeroGennis A330 目錄。請先安裝飛機，或在設定中手動指定路徑。",
		"create_liveries_dir_error":        "建立塗裝目錄失敗",
		"status_downloading_livery":        "正在下載塗裝 #%d...",
		"status_extracting_livery":         "正在解壓縮塗裝檔案...",
		"livery_install_complete_status":   "塗裝安裝完成！",
		"livery_install_success_message":   "塗裝 '%s' 已成功安裝！",
		"status_downloading_update":        "正在下載新版應用程式...",
		"download_update_error":            "下載更新失敗",
		"update_download_complete_status":  "更新下載成功！",
		"update_download_complete_title":   "下載完成",
		"update_download_complete_message": "新版本已儲存至：\n%s\n請關閉此應用程式並執行新程式。",
		"download_progress_label":          "下載中... %.2f / %.2f MB (%.2f MB/s)",
		"download_no_progress":             "警告：內容長度未知。將不會顯示進度。",
		"extract_progress_label":           "正在解壓縮: %s",
	},
	"fr-FR": {
		"window_title":                     "Installeur AeroGennis A330-300 - v2025.7.26.20-Preview",
		"reinstall_button":                 "Vérifier les mises à jour / Réinstaller",
		"aircraft_installed_title":         "Installation Détectée",
		"aircraft_installed_desc":          "L'application a détecté que l'AeroGennis A330-300 est déjà installé. Vous pouvez vérifier les mises à jour ou le réinstaller si nécessaire.",
		"manual_path_label":                "Forcer le chemin manuel de l'AG330",
		"manual_path_placeholder":          "Optionnel : Entrez le chemin complet vers le dossier 'AeroGennis Airbus A330-300'",
		"save_manual_path_button":          "Enregistrer le chemin manuel",
		"manual_path_error":                "Le chemin fourni est invalide ou n'existe pas. Veuillez vérifier et réessayer.",
		"save_success_title":               "Enregistré",
		"save_manual_path_success":         "Le chemin manuel a été enregistré avec succès. L'application a été actualisée.",
		"setup_welcome":                    "Bienvenue ! Veuillez sélectionner votre répertoire racine de X-Plane 12 pour commencer.",
		"path_placeholder":                 "Entrez ou parcourez jusqu'à votre répertoire racine de X-Plane 12...",
		"browse_button":                    "Parcourir...",
		"save_continue_button":             "Enregistrer et Continuer",
		"invalid_xp_path_error":            "Le répertoire sélectionné n'est pas une installation valide de X-Plane 12.",
		"missing_items_label":              "Éléments manquants :",
		"save_config_error":                "Échec de la sauvegarde de la configuration",
		"status_ready":                     "Prêt. Sélectionnez une option.",
		"tab_aircraft":                     "Avion",
		"tab_liveries":                     "Livrées",
		"tab_update_app":                   "Mettre à jour l'app",
		"tab_settings":                     "Paramètres",
		"aircraft_tab_title":               "Installer ou Mettre à Jour l'AeroGennis A330-300",
		"aircraft_tab_desc":                "Ceci téléchargera la dernière version de l'AeroGennis A330-300 et l'installera dans votre répertoire X-Plane 12/Aircraft.",
		"install_aircraft_button":          "Installer l'AeroGennis A330-300",
		"install_livery_button":            "Installer la Livrée Sélectionnée",
		"no_livery_selected_title":         "Aucune Livrée Sélectionnée",
		"no_livery_selected_message":       "Veuillez sélectionner une livrée dans la liste avant l'installation.",
		"livery_selected_status":           "Sélectionné : %s",
		"no_livery_selected_status":        "Aucune livrée sélectionnée.",
		"update_tab_title":                 "Mettre à Jour Cette Application",
		"update_tab_desc":                  "Ceci téléchargera la dernière version de cet installeur (en tant que fichier .exe). Vous devrez fermer cette application et exécuter la nouvelle manuellement.",
		"update_tab_warning":               "Note : Ce fichier n'est pas signé numériquement et peut être signalé par un logiciel antivirus. C'est un faux positif.",
		"download_latest_button":           "Télécharger la Dernière Version de l'Application",
		"settings_tab_title":               "Paramètres de l'Application",
		"current_path_label":               "Chemin Actuel de X-Plane 12 : %s",
		"change_path_button":               "Changer le Répertoire de X-Plane 12",
		"change_language_button":           "Changer de Langue",
		"status_creating_temp_dir":         "Création du répertoire temporaire...",
		"temp_dir_error":                   "Échec de la création du répertoire temporaire",
		"status_downloading":               "Téléchargement de %s...",
		"aircraft_package":                 "le pack de l'avion",
		"livery_package":                   "la livrée",
		"download_failed_status":           "Téléchargement échoué.",
		"download_error":                   "Échec du téléchargement de %s",
		"status_extracting":                "Extraction des fichiers vers %s...",
		"extraction_failed_status":         "Extraction échouée.",
		"extraction_error":                 "Échec de l'extraction du pack %s",
		"install_complete_status":          "Installation de l'avion terminée !",
		"install_success_title":            "Succès",
		"aircraft_install_success_message": "L'AeroGennis A330-300 a été installé avec succès !",
		"find_aircraft_dir_error":          "Impossible de trouver un répertoire valide pour l'AeroGennis A330. Veuillez d'abord installer l'avion ou définir le chemin manuellement dans les Paramètres.",
		"create_liveries_dir_error":        "Échec de la création du répertoire des livrées",
		"status_downloading_livery":        "Téléchargement de la livrée #%d...",
		"status_extracting_livery":         "Extraction des fichiers de la livrée...",
		"livery_install_complete_status":   "Installation de la livrée terminée !",
		"livery_install_success_message":   "La livrée '%s' a été installée avec succès !",
		"status_downloading_update":        "Téléchargement de la nouvelle version de l'application...",
		"download_update_error":            "Échec du téléchargement de la mise à jour",
		"update_download_complete_status":  "Mise à jour téléchargée avec succès !",
		"update_download_complete_title":   "Téléchargement Terminé",
		"update_download_complete_message": "Nouvelle version enregistrée dans :\n%s\nVeuillez fermer cette application et exécuter la nouvelle.",
		"download_progress_label":          "Téléchargement... %.2f / %.2f Mo (%.2f Mo/s)",
		"download_no_progress":             "Avertissement : Longueur du contenu inconnue. La progression ne sera pas affichée.",
		"extract_progress_label":           "Extraction : %s",
	},
	"ru-RU": {
		"window_title":                     "Установщик AeroGennis A330-300 - v2025.7.26.20-Preview",
		"reinstall_button":                 "Проверить обновления / Переустановить",
		"aircraft_installed_title":         "Обнаружена Установка",
		"aircraft_installed_desc":          "Приложение обнаружило, что AeroGennis A330-300 уже установлен. Вы можете проверить наличие обновлений или переустановить при необходимости.",
		"manual_path_label":                "Переопределить путь к AG330 вручную",
		"manual_path_placeholder":          "Необязательно: Введите полный путь к папке 'AeroGennis Airbus A330-300'",
		"save_manual_path_button":          "Сохранить ручной путь",
		"manual_path_error":                "Указанный путь недействителен или не существует. Пожалуйста, проверьте и попробуйте снова.",
		"save_success_title":               "Сохранено",
		"save_manual_path_success":         "Путь, указанный вручную, успешно сохранен. Приложение было обновлено.",
		"setup_welcome":                    "Добро пожаловать! Пожалуйста, выберите корневой каталог X-Plane 12, чтобы начать.",
		"path_placeholder":                 "Введите или выберите корневой каталог X-Plane 12...",
		"browse_button":                    "Обзор...",
		"save_continue_button":             "Сохранить и Продолжить",
		"invalid_xp_path_error":            "Выбранный каталог не является действительной установкой X-Plane 12.",
		"missing_items_label":              "Отсутствующие элементы:",
		"save_config_error":                "Не удалось сохранить конфигурацию",
		"status_ready":                     "Готово. Выберите действие.",
		"tab_aircraft":                     "Самолёт",
		"tab_liveries":                     "Ливреи",
		"tab_update_app":                   "Обновить ПО",
		"tab_settings":                     "Настройки",
		"aircraft_tab_title":               "Установить или Обновить AeroGennis A330-300",
		"aircraft_tab_desc":                "Будет загружена последняя версия AeroGennis A330-300 и установлена в ваш каталог X-Plane 12/Aircraft.",
		"install_aircraft_button":          "Установить AeroGennis A330-300",
		"install_livery_button":            "Установить Выбранную Ливрею",
		"no_livery_selected_title":         "Ливрея не выбрана",
		"no_livery_selected_message":       "Пожалуйста, выберите ливрею из списка перед установкой.",
		"livery_selected_status":           "Выбрано: %s",
		"no_livery_selected_status":        "Ливрея не выбрана.",
		"update_tab_title":                 "Обновить Это Приложение",
		"update_tab_desc":                  "Будет загружена последняя версия этого установщика (в виде .exe файла). Вам нужно будет закрыть это приложение и запустить новое вручную.",
		"update_tab_warning":               "Примечание: Этот файл не имеет цифровой подписи и может быть помечен антивирусным ПО. Это ложное срабатывание.",
		"download_latest_button":           "Скачать Последнюю Версию Приложения",
		"settings_tab_title":               "Настройки Приложения",
		"current_path_label":               "Текущий путь к X-Plane 12: %s",
		"change_path_button":               "Изменить Каталог X-Plane 12",
		"change_language_button":           "Изменить Язык",
		"status_creating_temp_dir":         "Создание временного каталога...",
		"temp_dir_error":                   "Не удалось создать временный каталог",
		"status_downloading":               "Загрузка %s...",
		"aircraft_package":                 "пакета самолёта",
		"livery_package":                   "ливреи",
		"download_failed_status":           "Загрузка не удалась.",
		"download_error":                   "Не удалось загрузить %s",
		"status_extracting":                "Извлечение файлов в %s...",
		"extraction_failed_status":         "Извлечение не удалось.",
		"extraction_error":                 "Не удалось извлечь пакет %s",
		"install_complete_status":          "Установка самолёта завершена!",
		"install_success_title":            "Успешно",
		"aircraft_install_success_message": "AeroGennis A330-300 успешно установлен!",
		"find_aircraft_dir_error":          "Не удалось найти действительный каталог AeroGennis A330. Пожалуйста, сначала установите самолёт или укажите путь вручную в Настройках.",
		"create_liveries_dir_error":        "Не удалось создать каталог для ливрей",
		"status_downloading_livery":        "Загрузка ливреи #%d...",
		"status_extracting_livery":         "Извлечение файлов ливреи...",
		"livery_install_complete_status":   "Установка ливреи завершена!",
		"livery_install_success_message":   "Ливрея '%s' успешно установлена!",
		"status_downloading_update":        "Загрузка новой версии приложения...",
		"download_update_error":            "Не удалось загрузить обновление",
		"update_download_complete_status":  "Обновление успешно загружено!",
		"update_download_complete_title":   "Загрузка Завершена",
		"update_download_complete_message": "Новая версия сохранена в:\n%s\nПожалуйста, закройте это приложение и запустите новое.",
		"download_progress_label":          "Загрузка... %.2f / %.2f МБ (%.2f МБ/с)",
		"download_no_progress":             "Внимание: Размер содержимого неизвестен. Прогресс не будет отображаться.",
		"extract_progress_label":           "Извлечение: %s",
	},
}

// loadTranslations 根据所选语言设置应用程序的翻译映射。
func loadTranslations(state *AppState) {
	trans, ok := translations[state.language]
	if !ok {
		// 如果找不到所选语言，则回退到英语。
		state.translations = translations["en-US"]
		return
	}
	state.translations = trans
}
