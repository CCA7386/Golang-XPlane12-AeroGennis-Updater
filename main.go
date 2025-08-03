package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// Livery 结构体用于将涂装名称和下载链接关联起来。
type Livery struct {
	Name string
	URL  string
}

// AppState 保存应用程序的状态。
type AppState struct {
	app                 fyne.App // 将 App 实例保存在 state 中
	xpPath              string
	ag330Path           string
	isAircraftInstalled bool
	language            string
	translations        map[string]string
	mainWindow          fyne.Window
	statusLabel         *widget.Label
	progressBar         *widget.ProgressBar
	installAircraftBtn  *widget.Button
	updateExeBtn        *widget.Button
	liveryCheckGroup    *widget.CheckGroup
	installLiveryBtn    *widget.Button
	updateListBtn       *widget.Button
	uninstallBtn        *widget.Button
	liveries            []Livery
}

const (
	LiveryListURL       = "https://files.zohopublic.com.cn/public/workdrive-public/download/kpgnr1efdca4ab9ed48a280b91151e177fa0c?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrDQf-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D"
	ConcurrentDownloads = 4
)

var downloadURLAg330 = []string{"https://files.zohopublic.com.cn/public/workdrive-public/download/dqd1m03114168cdbd47608183f4445c9b557c?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrBKN-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D"}
var downloadURLUpdater = []string{"https://files.zohopublic.com.cn/public/workdrive-public/download/dqd1ma5b2ddd90a0647ed918d5ec5fe42de34?x-cli-msg=%7B%22linkId%22%3A%221GNlXvxrBKN-36kFa%22%2C%22isFileOwner%22%3Afalse%2C%22version%22%3A%221.0%22%2C%22isWDSupport%22%3Afalse%7D"}

func (state *AppState) tr(key string, args ...interface{}) string {
	format, ok := state.translations[key]
	if !ok {
		return key
	}
	if len(args) > 0 {
		return fmt.Sprintf(format, args...)
	}
	return format
}

func getExecutablePath(filename string) (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exePath), filename), nil
}

func loadLiveriesFromFile() ([]Livery, error) {
	path, err := getExecutablePath("LiveriesList.txt")
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Livery{}, nil
		}
		return nil, fmt.Errorf("无法打开 LiveriesList.txt: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, `"`) && strings.HasSuffix(line, `",`) {
			line = line[1 : len(line)-2]
		} else if strings.HasPrefix(line, `"`) && strings.HasSuffix(line, `"`) {
			line = line[1 : len(line)-1]
		}
		if line != "" {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取 LiveriesList.txt 时出错: %w", err)
	}

	var names, urls []string
	for _, line := range lines {
		if strings.HasPrefix(line, "http") {
			urls = append(urls, line)
		} else {
			names = append(names, line)
		}
	}
	if len(names) != len(urls) {
		return nil, fmt.Errorf("涂装名称数量 (%d) 与下载链接数量 (%d) 不匹配", len(names), len(urls))
	}

	var liveries []Livery
	for i := 0; i < len(names); i++ {
		liveries = append(liveries, Livery{Name: names[i], URL: urls[i]})
	}
	return liveries, nil
}

func main() {
	a := app.New()
	w := a.NewWindow("AeroGennis A330-300 Installer")
	w.Resize(fyne.NewSize(700, 500))
	state := &AppState{app: a, mainWindow: w} // 在 state 中初始化 app
	loadedLiveries, err := loadLiveriesFromFile()
	if err != nil {
		dialog.ShowError(err, w)
		state.liveries = []Livery{}
	} else {
		state.liveries = loadedLiveries
	}
	xpPath, lang, ag330Path := readConfig()
	if lang == "" {
		w.SetContent(createLanguageSelectionUI(state))
	} else {
		state.language = lang
		loadTranslations(state)
		w.SetTitle(state.tr("window_title"))
		if xpPath != "" {
			if valid, _ := validateXPlaneDirectory(xpPath); valid {
				state.xpPath = xpPath
				state.ag330Path = ag330Path
				checkAircraftInstallation(state)
			} else {
				state.xpPath = ""
				writeConfig(state)
			}
		}
		if state.xpPath == "" {
			w.SetContent(createSetupUI(state))
		} else {
			w.SetContent(createMainUI(state))
		}
	}
	w.ShowAndRun()
}

func createLanguageSelectionUI(state *AppState) fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Select Language / 语言选择", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	prompt := widget.NewLabel("Please select your language:")
	langOptions := []string{"English", "简体中文", "繁體中文", "Français", "Русский"}
	langCodes := map[string]string{"English": "en-US", "简体中文": "zh-CN", "繁體中文": "zh-TW", "Français": "fr-FR", "Русский": "ru-RU"}
	langSelect := widget.NewSelect(langOptions, func(selected string) { state.language = langCodes[selected] })
	langSelect.SetSelectedIndex(0)
	continueBtn := widget.NewButton("Continue", func() {
		if state.language == "" {
			state.language = "en-US"
		}
		loadTranslations(state)
		if err := writeConfig(state); err != nil {
			dialog.ShowError(fmt.Errorf("%s: %w", state.tr("save_config_error"), err), state.mainWindow)
			return
		}
		state.mainWindow.SetTitle(state.tr("window_title"))
		state.mainWindow.SetContent(createSetupUI(state))
	})
	return container.NewVBox(title, prompt, langSelect, continueBtn)
}

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
		checkAircraftInstallation(state)
		state.mainWindow.SetContent(createMainUI(state))
	})
	return container.NewVBox(widget.NewLabel(state.tr("setup_welcome")), pathEntry, browseBtn, saveBtn)
}

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
	return container.NewBorder(nil, container.NewVBox(state.statusLabel, state.progressBar), nil, nil, tabs)
}

func createAircraftTab(state *AppState) fyne.CanvasObject {
	var content fyne.CanvasObject
	if state.isAircraftInstalled {
		state.installAircraftBtn = widget.NewButton(state.tr("reinstall_button"), func() { handleAircraftInstall(state) })
		content = container.NewVBox(widget.NewLabelWithStyle(state.tr("aircraft_installed_title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}), widget.NewLabel(state.tr("aircraft_installed_desc")), state.installAircraftBtn)
	} else {
		state.installAircraftBtn = widget.NewButton(state.tr("install_aircraft_button"), func() { handleAircraftInstall(state) })
		content = container.NewVBox(widget.NewLabelWithStyle(state.tr("aircraft_tab_title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}), widget.NewLabel(state.tr("aircraft_tab_desc")), state.installAircraftBtn)
	}
	return content
}

func createLiveryTab(state *AppState) fyne.CanvasObject {
	var liveryNames []string
	for _, livery := range state.liveries {
		liveryNames = append(liveryNames, livery.Name)
	}
	state.liveryCheckGroup = widget.NewCheckGroup(liveryNames, func(selected []string) {
		if len(selected) > 0 {
			state.installLiveryBtn.Enable()
			state.statusLabel.SetText(state.tr("liveries_selected_status", len(selected)))
		} else {
			state.installLiveryBtn.Disable()
			state.statusLabel.SetText(state.tr("no_livery_selected_status"))
		}
	})
	state.installLiveryBtn = widget.NewButton(state.tr("install_selected_liveries_button"), func() { handleBatchLiveryInstall(state) })
	state.installLiveryBtn.Disable()
	state.updateListBtn = widget.NewButton(state.tr("update_livery_list_button"), func() { handleUpdateLiveryList(state) })
	state.uninstallBtn = widget.NewButton(state.tr("uninstall_liveries_button"), func() { handleUninstallLiveries(state) })
	bottomBar := container.NewVBox(state.installLiveryBtn, container.NewGridWithColumns(2, state.updateListBtn, state.uninstallBtn))
	if len(state.liveries) == 0 {
		return container.NewCenter(container.NewVBox(widget.NewLabel(state.tr("livery_list_load_fail")), state.updateListBtn))
	}
	return container.NewBorder(nil, bottomBar, nil, nil, container.NewScroll(state.liveryCheckGroup))
}

func createUpdateTab(state *AppState) fyne.CanvasObject {
	state.updateExeBtn = widget.NewButton(state.tr("download_latest_button"), func() { handleExeUpdate(state) })
	return container.NewVBox(widget.NewLabelWithStyle(state.tr("update_tab_title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}), widget.NewLabel(state.tr("update_tab_desc")), widget.NewLabel(state.tr("update_tab_warning")), state.updateExeBtn)
}

func createSettingsTab(state *AppState) fyne.CanvasObject {
	pathLabel := widget.NewLabel(state.tr("current_path_label", state.xpPath))
	pathLabel.Wrapping = fyne.TextWrapWord
	changePathBtn := widget.NewButton(state.tr("change_path_button"), func() { state.mainWindow.SetContent(createSetupUI(state)) })
	changeLangBtn := widget.NewButton(state.tr("change_language_button"), func() { state.mainWindow.SetContent(createLanguageSelectionUI(state)) })
	ag330PathEntry := widget.NewEntry()
	ag330PathEntry.SetText(state.ag330Path)
	ag330PathEntry.SetPlaceHolder(state.tr("manual_path_placeholder"))
	saveAg330PathBtn := widget.NewButton(state.tr("save_manual_path_button"), func() {
		manualPath := ag330PathEntry.Text
		if info, err := os.Stat(manualPath); err != nil || !info.IsDir() {
			dialog.ShowError(fmt.Errorf("%s", state.tr("manual_path_error")), state.mainWindow)
			return
		}
		state.ag330Path = manualPath
		if err := writeConfig(state); err != nil {
			dialog.ShowError(fmt.Errorf("%s: %w", state.tr("save_config_error"), err), state.mainWindow)
			return
		}
		checkAircraftInstallation(state)
		state.mainWindow.SetContent(createMainUI(state))
		dialog.ShowInformation(state.tr("save_success_title"), state.tr("save_manual_path_success"), state.mainWindow)
	})
	uninstallAircraftBtn := widget.NewButton(state.tr("uninstall_aircraft_button"), func() { handleUninstallAircraft(state) })
	uninstallAircraftBtn.Importance = widget.DangerImportance
	if !state.isAircraftInstalled {
		uninstallAircraftBtn.Disable()
	}
	selfUninstallWarning := widget.NewLabel(state.tr("self_uninstall_warning"))
	selfUninstallWarning.Wrapping = fyne.TextWrapWord
	selfUninstallBtn := widget.NewButton(state.tr("self_uninstall_button"), func() { handleSelfUninstall(state) })
	selfUninstallBtn.Importance = widget.DangerImportance
	return container.NewVBox(
		widget.NewLabelWithStyle(state.tr("settings_tab_title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		pathLabel, changePathBtn, changeLangBtn, widget.NewSeparator(),
		widget.NewLabelWithStyle(state.tr("manual_path_label"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		ag330PathEntry, saveAg330PathBtn, widget.NewSeparator(),
		widget.NewLabelWithStyle(state.tr("danger_zone_label"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		uninstallAircraftBtn, widget.NewSeparator(),
		selfUninstallWarning, selfUninstallBtn,
	)
}

func downloadFileWithProgress(url, destPath string, state *AppState) error {
	// URL 验证
	if !strings.HasPrefix(url, "https://files.zohopublic.com.cn") {
		return fmt.Errorf("无效的下载 URL: %s", url)
	}

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
	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 使用 bufio.Writer 优化写入性能
	writer := bufio.NewWriter(file)
	defer writer.Flush()

	if totalBytes <= 0 {
		state.statusLabel.SetText(state.tr("download_no_progress"))
		state.progressBar.SetValue(0.5) // Indicate activity
		_, err = io.Copy(writer, resp.Body)
		return err
	}

	// 优化缓冲区大小为 64KB
	buf := make([]byte, 64*1024)
	var downloadedBytes int64
	startTime := time.Now()
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := writer.Write(buf[0:n]); writeErr != nil {
				return writeErr
			}
			downloadedBytes += int64(n)
			speed := float64(downloadedBytes) / time.Since(startTime).Seconds() / (1024 * 1024)

			// 计算进度百分比 (0.0 到 1.0)
			progress := float64(downloadedBytes) / float64(totalBytes)
			state.statusLabel.SetText(state.tr("download_progress_label", float64(downloadedBytes)/(1024*1024), float64(totalBytes)/(1024*1024), speed))
			state.progressBar.SetValue(progress)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	return nil
}

func extractZipGUI(zipFile, destDir string, isAircraft bool, state *AppState) error {
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer r.Close()

	totalFiles := len(r.File)
	destRoot := destDir
	if isAircraft {
		destRoot = filepath.Join(destDir, "AeroGennis Airbus A330-300")
	}

	for i, f := range r.File {
		fpath := filepath.Join(destRoot, f.Name)
		// 路径安全验证
		if !strings.HasPrefix(filepath.Clean(fpath), filepath.Clean(destRoot)+string(os.PathSeparator)) {
			return fmt.Errorf("非法文件路径: %s", fpath)
		}

		// 计算进度百分比 (0.0 到 1.0)
		progress := float64(i+1) / float64(totalFiles)
		state.progressBar.SetValue(progress)
		state.statusLabel.SetText(state.tr("extract_progress_label", f.Name))

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

func handleUninstallLiveries(state *AppState) {
	if !state.isAircraftInstalled || state.ag330Path == "" {
		dialog.ShowError(fmt.Errorf("%s", state.tr("find_aircraft_dir_error")), state.mainWindow)
		return
	}
	liveriesPath := filepath.Join(state.ag330Path, "liveries")
	entries, err := os.ReadDir(liveriesPath)
	if err != nil {
		dialog.ShowError(fmt.Errorf("%s: %w", state.tr("scan_liveries_dir_error"), err), state.mainWindow)
		return
	}
	var installedLiveryNames []string
	for _, entry := range entries {
		if entry.IsDir() {
			installedLiveryNames = append(installedLiveryNames, entry.Name())
		}
	}
	if len(installedLiveryNames) == 0 {
		dialog.ShowInformation(state.tr("no_installed_liveries_title"), state.tr("no_installed_liveries_message"), state.mainWindow)
		return
	}
	checkGroup := widget.NewCheckGroup(installedLiveryNames, nil)
	uninstallDialog := dialog.NewCustomConfirm(
		state.tr("uninstall_dialog_title"), state.tr("confirm_button"), state.tr("cancel_button"), container.NewScroll(checkGroup),
		func(confirm bool) {
			if !confirm {
				return
			}
			selectedToUninstall := checkGroup.Selected
			if len(selectedToUninstall) == 0 {
				return
			}
			confirmMsg := state.tr("uninstall_final_confirm_message", len(selectedToUninstall), strings.Join(selectedToUninstall, "\n- "))
			dialog.ShowConfirm(state.tr("uninstall_final_confirm_title"), confirmMsg, func(finalConfirm bool) {
				if !finalConfirm {
					return
				}
				var deletedCount, errorCount int
				var errorMessages []string
				for _, name := range selectedToUninstall {
					pathToDelete := filepath.Join(liveriesPath, name)
					if err := os.RemoveAll(pathToDelete); err != nil {
						errorCount++
						errorMessages = append(errorMessages, fmt.Sprintf("%s: %v", name, err))
					} else {
						deletedCount++
					}
				}
				resultMsg := state.tr("uninstall_report_message", deletedCount)
				if errorCount > 0 {
					resultMsg += "\n" + state.tr("uninstall_report_errors", errorCount, strings.Join(errorMessages, "\n"))
				}
				dialog.ShowInformation(state.tr("uninstall_complete_title"), resultMsg, state.mainWindow)
			}, state.mainWindow)
		}, state.mainWindow)
	uninstallDialog.Resize(fyne.NewSize(400, 300))
	uninstallDialog.Show()
}

func handleExeUpdate(state *AppState) {
	dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil || uri == nil {
			return
		}
		savePath := uri.Path()
		state.updateExeBtn.Disable()
		state.progressBar.SetValue(0)

		go func() {
			defer func() {
				state.updateExeBtn.Enable()
			}()

			exeName := "AeroGennis_Updater_New.exe"
			exePath := filepath.Join(savePath, exeName)
			state.statusLabel.SetText(state.tr("status_downloading_update"))

			err := downloadFileWithProgress(downloadURLUpdater[0], exePath, state)
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

// (removed duplicate definition of downloadFileWithProgressSafe)

// (Removed duplicate definition of extractZipGUISafe)
func handleUninstallAircraft(state *AppState) {
	if !state.isAircraftInstalled || state.ag330Path == "" {
		return
	}
	dialog.ShowConfirm(
		state.tr("uninstall_aircraft_confirm_title"),
		state.tr("uninstall_aircraft_confirm_message", state.ag330Path),
		func(confirm bool) {
			if !confirm {
				return
			}
			dialog.ShowConfirm(
				state.tr("uninstall_final_confirm_title"),
				state.tr("uninstall_aircraft_final_confirm_message"),
				func(finalConfirm bool) {
					if !finalConfirm {
						return
					}
					err := os.RemoveAll(state.ag330Path)
					if err != nil {
						dialog.ShowError(fmt.Errorf("%s: %w", state.tr("uninstall_aircraft_error"), err), state.mainWindow)
						return
					}
					dialog.ShowInformation(state.tr("uninstall_complete_title"), state.tr("uninstall_aircraft_success_message"), state.mainWindow)
					state.isAircraftInstalled = false
					state.ag330Path = ""
					writeConfig(state)
					state.mainWindow.SetContent(createMainUI(state))
				},
				state.mainWindow,
			)
		},
		state.mainWindow,
	)
}

func handleSelfUninstall(state *AppState) {
	dialog.ShowConfirm(
		state.tr("self_uninstall_confirm_title"),
		state.tr("self_uninstall_confirm_message"),
		func(confirm bool) {
			if !confirm {
				return
			}
			exePath, err := os.Executable()
			if err != nil {
				dialog.ShowError(err, state.mainWindow)
				return
			}
			confPath, _ := getExecutablePath("Ag330UpdaterConf.txt")
			listPath, _ := getExecutablePath("LiveriesList.txt")
			scriptContent := fmt.Sprintf(
				`@echo off
timeout /t 2 /nobreak > NUL
del "%s"
del "%s"
del "%s"
(goto) 2>nul & del "%%~f0"
`, confPath, listPath, exePath)
			tempBatFile, err := os.CreateTemp("", "uninstall_*.bat")
			if err != nil {
				dialog.ShowError(err, state.mainWindow)
				return
			}
			if _, err := tempBatFile.WriteString(scriptContent); err != nil {
				tempBatFile.Close()
				dialog.ShowError(err, state.mainWindow)
				return
			}
			tempBatFile.Close()
			cmd := exec.Command("cmd", "/C", "start", "/b", tempBatFile.Name())
			if err := cmd.Start(); err != nil {
				dialog.ShowError(err, state.mainWindow)
				return
			}
			state.mainWindow.Close()
		},
		state.mainWindow,
	)
}

func handleAircraftInstall(state *AppState) {
	state.installAircraftBtn.Disable()

	go func() {
		defer func() {
			state.installAircraftBtn.Enable()
		}()

		// 更新状态标签
		state.statusLabel.SetText(state.tr("status_creating_temp_dir"))
		state.progressBar.SetValue(0)

		tmpDir, err := os.MkdirTemp("", "xplane_aircraft_*")
		if err != nil {
			dialog.ShowError(fmt.Errorf("%s: %w", state.tr("temp_dir_error"), err), state.mainWindow)
			return
		}
		defer os.RemoveAll(tmpDir)

		zipPath := filepath.Join(tmpDir, "aircraft.zip")
		state.statusLabel.SetText(state.tr("status_downloading", state.tr("aircraft_package")))

		err = downloadFileWithProgress(downloadURLAg330[0], zipPath, state)
		if err != nil {
			state.statusLabel.SetText(state.tr("download_failed_status"))
			dialog.ShowError(fmt.Errorf("%s: %w", state.tr("download_error", state.tr("aircraft_package")), err), state.mainWindow)
			return
		}

		aircraftDir := filepath.Join(state.xpPath, "Aircraft", "Laminar Research")
		state.statusLabel.SetText(state.tr("status_extracting", aircraftDir))

		err = extractZipGUI(zipPath, aircraftDir, true, state)
		if err != nil {
			state.statusLabel.SetText(state.tr("extraction_failed_status"))
			dialog.ShowError(fmt.Errorf("%s: %w", state.tr("extraction_error", state.tr("aircraft_package")), err), state.mainWindow)
			return
		}

		checkAircraftInstallation(state)
		state.mainWindow.SetContent(createMainUI(state))
		state.statusLabel.SetText(state.tr("install_complete_status"))
		dialog.ShowInformation(state.tr("install_success_title"), state.tr("aircraft_install_success_message"), state.mainWindow)
	}()
}

func handleUpdateLiveryList(state *AppState) {
	state.updateListBtn.Disable()
	state.statusLabel.SetText(state.tr("status_updating_livery_list"))
	go func() {
		defer state.updateListBtn.Enable()
		resp, err := http.Get(LiveryListURL)
		if err != nil {
			dialog.ShowError(fmt.Errorf("%s: %w", state.tr("livery_list_download_error"), err), state.mainWindow)
			return
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			dialog.ShowError(fmt.Errorf("%s: %w", state.tr("livery_list_read_error"), err), state.mainWindow)
			return
		}
		path, err := getExecutablePath("LiveriesList.txt")
		if err != nil {
			dialog.ShowError(err, state.mainWindow)
			return
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			dialog.ShowError(fmt.Errorf("%s: %w", state.tr("livery_list_save_error"), err), state.mainWindow)
			return
		}
		loadedLiveries, err := loadLiveriesFromFile()
		if err != nil {
			dialog.ShowError(err, state.mainWindow)
		} else {
			state.liveries = loadedLiveries
			state.mainWindow.SetContent(createMainUI(state))
			dialog.ShowInformation(state.tr("update_success_title"), state.tr("livery_list_update_success"), state.mainWindow)
		}

	}()
}

func handleBatchLiveryInstall(state *AppState) {
	selectedNames := state.liveryCheckGroup.Selected
	if len(selectedNames) == 0 {
		dialog.ShowInformation(state.tr("no_livery_selected_title"), state.tr("no_livery_selected_message"), state.mainWindow)
		return
	}
	if !state.isAircraftInstalled || state.ag330Path == "" {
		dialog.ShowError(fmt.Errorf("%s", state.tr("find_aircraft_dir_error")), state.mainWindow)
		return
	}

	// 去重处理
	uniqueNames := make(map[string]struct{})
	var downloadQueue []Livery
	for _, name := range selectedNames {
		if _, exists := uniqueNames[name]; exists {
			continue // 跳过重复的涂装
		}
		uniqueNames[name] = struct{}{}
		for _, livery := range state.liveries {
			if livery.Name == name {
				downloadQueue = append(downloadQueue, livery)
				break
			}
		}
	}

	state.installLiveryBtn.Disable()
	state.updateListBtn.Disable()
	state.uninstallBtn.Disable()

	go func() {
		totalJobs := len(downloadQueue)
		defer func() {
			// 使用 RunOnMain 确保 UI 更新在主线程中执行
			state.mainWindow.Canvas().Refresh(state.mainWindow.Content())
			state.installLiveryBtn.Enable()
			state.updateListBtn.Enable()
			state.uninstallBtn.Enable()
			state.liveryCheckGroup.SetSelected([]string{})
			state.statusLabel.SetText(state.tr("batch_install_complete_status", totalJobs))
			dialog.ShowInformation(state.tr("install_success_title"), state.tr("batch_install_complete_message", totalJobs), state.mainWindow)
		}()

		var wg sync.WaitGroup
		var completedCount atomic.Int32
		jobs := make(chan Livery, totalJobs)

		// 创建一个通道来传递状态更新
		statusUpdates := make(chan string, 100)
		progressUpdates := make(chan float64, 100)

		// 启动一个 goroutine 来处理所有 UI 更新
		go func() {
			for {
				select {
				case status, ok := <-statusUpdates:
					if !ok {
						return
					}
					state.statusLabel.SetText(status)
				case progress, ok := <-progressUpdates:
					if !ok {
						return
					}
					state.progressBar.SetValue(progress)
				}
			}
		}()

		for i := 1; i <= ConcurrentDownloads; i++ {
			wg.Add(1)
			go liveryInstallWorker(i, state, jobs, &wg, &completedCount, totalJobs, statusUpdates, progressUpdates)
		}

		for _, livery := range downloadQueue {
			jobs <- livery
		}
		close(jobs)
		wg.Wait()

		// 关闭更新通道
		close(statusUpdates)
		close(progressUpdates)
	}()
}

func liveryInstallWorker(id int, state *AppState, jobs <-chan Livery, wg *sync.WaitGroup, counter *atomic.Int32, total int, statusUpdates chan<- string, progressUpdates chan<- float64) {
	defer wg.Done()
	for livery := range jobs {
		currentNum := counter.Add(1)

		// 发送状态更新到通道而不是直接更新 UI
		select {
		case statusUpdates <- state.tr("batch_download_progress_label", currentNum, total, livery.Name):
		default:
		}
		select {
		case progressUpdates <- 0:
		default:
		}

		liveryDir := filepath.Join(state.ag330Path, "liveries")
		os.MkdirAll(liveryDir, 0755)
		tmpDir, err := os.MkdirTemp("", "xplane_livery_*")
		if err != nil {
			fmt.Printf("Worker %d: 创建临时目录失败: %v\n", id, err)
			continue
		}
		zipPath := filepath.Join(tmpDir, "livery.zip")

		// 创建一个包装的 state 来安全地更新进度
		wrappedState := &AppState{
			app:          state.app,
			statusLabel:  state.statusLabel,
			progressBar:  state.progressBar,
			translations: state.translations,
			language:     state.language,
		}

		err = downloadFileWithProgressSafe(livery.URL, zipPath, wrappedState, statusUpdates, progressUpdates)
		if err != nil {
			fmt.Printf("Worker %d: 下载 '%s' 失败: %v\n", id, livery.Name, err)
			os.RemoveAll(tmpDir)
			continue
		}

		err = extractZipGUISafe(zipPath, liveryDir, false, wrappedState, statusUpdates, progressUpdates)
		if err != nil {
			fmt.Printf("Worker %d: 解压 '%s' 失败: %v\n", id, livery.Name, err)
		}
		os.RemoveAll(tmpDir)
	}
}

// 创建线程安全的下载函数
func downloadFileWithProgressSafe(url, destPath string, state *AppState, statusUpdates chan<- string, progressUpdates chan<- float64) error {
	// URL 验证
	if !strings.HasPrefix(url, "https://files.zohopublic.com.cn") {
		return fmt.Errorf("无效的下载 URL: %s", url)
	}

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
	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 使用 bufio.Writer 优化写入性能
	writer := bufio.NewWriter(file)
	defer writer.Flush()

	if totalBytes <= 0 {
		statusUpdates <- state.tr("download_no_progress")
		progressUpdates <- 0.5
		_, err = io.Copy(writer, resp.Body)
		return err
	}

	// 优化缓冲区大小为 64KB
	buf := make([]byte, 64*1024)
	var downloadedBytes int64
	startTime := time.Now()
	lastUpdateTime := time.Now()

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := writer.Write(buf[0:n]); writeErr != nil {
				return writeErr
			}
			downloadedBytes += int64(n)

			// 限制更新频率，每100ms更新一次
			if time.Since(lastUpdateTime) > 100*time.Millisecond {
				speed := float64(downloadedBytes) / time.Since(startTime).Seconds() / (1024 * 1024)
				progress := float64(downloadedBytes) / float64(totalBytes)

				statusUpdates <- state.tr("download_progress_label", float64(downloadedBytes)/(1024*1024), float64(totalBytes)/(1024*1024), speed)
				progressUpdates <- progress
				lastUpdateTime = time.Now()
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	return nil
}

// 创建线程安全的解压函数
func extractZipGUISafe(zipFile, destDir string, isAircraft bool, state *AppState, statusUpdates chan<- string, progressUpdates chan<- float64) error {
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer r.Close()

	totalFiles := len(r.File)
	destRoot := destDir
	if isAircraft {
		destRoot = filepath.Join(destDir, "AeroGennis Airbus A330-300")
	}

	lastUpdateTime := time.Now()

	for i, f := range r.File {
		fpath := filepath.Join(destRoot, f.Name)
		// 路径安全验证
		if !strings.HasPrefix(filepath.Clean(fpath), filepath.Clean(destRoot)+string(os.PathSeparator)) {
			return fmt.Errorf("非法文件路径: %s", fpath)
		}

		// 限制更新频率
		if time.Since(lastUpdateTime) > 50*time.Millisecond {
			progress := float64(i+1) / float64(totalFiles)
			progressUpdates <- progress
			statusUpdates <- state.tr("extract_progress_label", f.Name)
			lastUpdateTime = time.Now()
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

func checkAircraftInstallation(state *AppState) {
	state.isAircraftInstalled = false
	var finalPath string
	if state.ag330Path != "" {
		if info, err := os.Stat(state.ag330Path); err == nil && info.IsDir() {
			finalPath = state.ag330Path
		}
	}
	if finalPath == "" && state.xpPath != "" {
		foundPath, err := findAerogennisDir(state.xpPath)
		if err == nil {
			finalPath = foundPath
		}
	}
	if finalPath != "" {
		size, err := getDirSize(finalPath)
		if err == nil {
			const requiredSize = 1395864371
			if size > requiredSize {
				state.isAircraftInstalled = true
				state.ag330Path = finalPath
			}
		}
	}
}

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

func validateXPlaneDirectory(path string) (bool, []string) {
	requiredItems := []string{"Aircraft", "Custom Scenery", "Global Scenery", "Resources", "X-Plane.exe"}
	var missingItems []string
	if !filepath.IsAbs(path) {
		return false, []string{"Path must be an absolute directory path."}
	}
	for _, item := range requiredItems {
		if _, err := os.Stat(filepath.Join(path, item)); os.IsNotExist(err) {
			missingItems = append(missingItems, item)
		}
	}
	return len(missingItems) == 0, missingItems
}

func findAerogennisDir(basePath string) (string, error) {
	aircraftPath := filepath.Join(basePath, "Aircraft")
	entries, err := os.ReadDir(aircraftPath)
	if err != nil {
		return "", fmt.Errorf("could not read Aircraft directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() && strings.Contains(strings.ToLower(entry.Name()), "aerogennis") {
			return filepath.Join(aircraftPath, entry.Name()), nil
		}
	}
	return "", fmt.Errorf("no directory containing 'Aerogennis' found in '%s'", aircraftPath)
}

func readConfig() (xpPath, lang, ag330Path string) {
	configPath, err := getExecutablePath("Ag330UpdaterConf.txt")
	if err != nil {
		return "", "", ""
	}
	data, err := os.ReadFile(configPath)
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

func writeConfig(state *AppState) error {
	configPath, err := getExecutablePath("Ag330UpdaterConf.txt")
	if err != nil {
		return err
	}
	content := fmt.Sprintf("%s\n%s\n%s", state.xpPath, state.language, state.ag330Path)
	return os.WriteFile(configPath, []byte(content), 0644)
}

// 翻译部分保持不变
var translations = map[string]map[string]string{
	"en-US": {
		"window_title":                             "AeroGennis A330-300 Installer - v2025.8.3.20-Preview",
		"reinstall_button":                         "Check for Updates / Reinstall",
		"aircraft_installed_title":                 "Installation Detected",
		"aircraft_installed_desc":                  "The application has detected that AeroGennis A330-300 is already installed. You can check for updates or reinstall if needed.",
		"manual_path_label":                        "Manual AG330 Path Override",
		"manual_path_placeholder":                  "Optional: Enter full path to 'AeroGennis Airbus A330-300' folder",
		"save_manual_path_button":                  "Save Manual Path",
		"manual_path_error":                        "The provided path is invalid or does not exist. Please check and try again.",
		"save_success_title":                       "Saved",
		"save_manual_path_success":                 "Manual path has been saved successfully. The application has been refreshed.",
		"setup_welcome":                            "Welcome! Please select your X-Plane 12 root directory to begin.",
		"path_placeholder":                         "Enter or browse to your X-Plane 12 root directory...",
		"browse_button":                            "Browse...",
		"save_continue_button":                     "Save and Continue",
		"invalid_xp_path_error":                    "The selected directory is not a valid X-Plane 12 installation.",
		"missing_items_label":                      "Missing items:",
		"save_config_error":                        "Failed to save configuration",
		"status_ready":                             "Ready. Select an option.",
		"tab_aircraft":                             "Aircraft",
		"tab_liveries":                             "Liveries",
		"tab_update_app":                           "Update App",
		"tab_settings":                             "Settings",
		"aircraft_tab_title":                       "Install or Update AeroGennis A330-300",
		"aircraft_tab_desc":                        "This will download the latest version of the AeroGennis A330-300 and install it into your X-Plane 12/Aircraft directory.",
		"install_aircraft_button":                  "Install AeroGennis A330-300",
		"no_livery_selected_title":                 "No Livery Selected",
		"no_livery_selected_message":               "Please check one or more liveries from the list before installing.",
		"liveries_selected_status":                 "%d liveries selected.",
		"no_livery_selected_status":                "No livery selected. Check a box to begin.",
		"update_tab_title":                         "Update This Application",
		"update_tab_desc":                          "This will download the latest version of this installer (as a .exe file). You will need to close this application and run the new one manually.",
		"update_tab_warning":                       "Note: This file is not digitally signed and may be flagged by antivirus software. This is a false positive.",
		"download_latest_button":                   "Download Latest Application Version",
		"settings_tab_title":                       "Application Settings",
		"current_path_label":                       "Current X-Plane 12 Path: %s",
		"change_path_button":                       "Change X-Plane 12 Directory",
		"change_language_button":                   "Change Language",
		"status_creating_temp_dir":                 "Creating temporary directory...",
		"temp_dir_error":                           "Failed to create temp directory",
		"status_downloading":                       "Downloading %s...",
		"aircraft_package":                         "aircraft package",
		"download_failed_status":                   "Download failed.",
		"download_error":                           "Failed to download %s",
		"status_extracting":                        "Extracting files to %s...",
		"extraction_failed_status":                 "Extraction failed.",
		"extraction_error":                         "Failed to extract %s package",
		"install_complete_status":                  "Aircraft installation complete!",
		"install_success_title":                    "Success",
		"aircraft_install_success_message":         "AeroGennis A330-300 has been installed successfully!",
		"find_aircraft_dir_error":                  "Could not find a valid AeroGennis A330 directory. Please install the aircraft first or set the path manually in Settings.",
		"status_downloading_update":                "Downloading new application version...",
		"download_update_error":                    "Failed to download update",
		"update_download_complete_status":          "Update downloaded successfully!",
		"update_download_complete_title":           "Download Complete",
		"update_download_complete_message":         "New version saved to:\n%s\nPlease close this application and run the new one.",
		"download_progress_label":                  "Downloading... %.2f / %.2f MB (%.2f MB/s)",
		"download_no_progress":                     "Warning: Content length unknown. Progress will not be shown.",
		"extract_progress_label":                   "Extracting: %s",
		"install_selected_liveries_button":         "Install Selected Liveries",
		"update_livery_list_button":                "Update Livery List",
		"uninstall_liveries_button":                "Uninstall Liveries...",
		"status_updating_livery_list":              "Downloading latest livery list...",
		"livery_list_download_error":               "Failed to download livery list",
		"livery_list_read_error":                   "Failed to read downloaded livery list data",
		"livery_list_save_error":                   "Failed to save livery list file",
		"update_success_title":                     "Updated",
		"livery_list_update_success":               "Livery list has been successfully updated and reloaded!",
		"livery_list_load_fail":                    "Could not load livery list. Please try updating it.",
		"batch_download_progress_label":            "Downloading (%d/%d): %s",
		"batch_install_complete_status":            "%d liveries processed.",
		"batch_install_complete_message":           "%d selected liveries have been processed and installed.",
		"scan_liveries_dir_error":                  "Failed to scan the liveries directory",
		"no_installed_liveries_title":              "No Liveries Found",
		"no_installed_liveries_message":            "No installed liveries were found in the expected directory.",
		"uninstall_dialog_title":                   "Select Liveries to Uninstall",
		"confirm_button":                           "Confirm",
		"cancel_button":                            "Cancel",
		"uninstall_final_confirm_title":            "Permanent Deletion",
		"uninstall_final_confirm_message":          "Are you sure you want to permanently delete these %d selected folders?\n\n- %s",
		"uninstall_complete_title":                 "Uninstallation Report",
		"uninstall_report_message":                 "Successfully deleted %d liveries.",
		"uninstall_report_errors":                  "Failed to delete %d liveries:\n%s",
		"danger_zone_label":                        "Danger Zone",
		"uninstall_aircraft_button":                "Uninstall AeroGennis A330-300",
		"uninstall_aircraft_confirm_title":         "Confirm Aircraft Uninstallation",
		"uninstall_aircraft_confirm_message":       "This will permanently delete the entire aircraft folder located at:\n\n%s\n\nThis action cannot be undone. Are you sure?",
		"uninstall_aircraft_final_confirm_message": "Final warning! All aircraft files, including any modifications or added liveries, will be deleted. Proceed?",
		"uninstall_aircraft_error":                 "Failed to uninstall the aircraft",
		"uninstall_aircraft_success_message":       "The AeroGennis A330-300 has been successfully uninstalled.",
		"self_uninstall_button":                    "Uninstall This Application",
		"self_uninstall_warning":                   "WARNING: This will remove the installer executable itself, along with its configuration and livery list files. You will need to re-download the application to use it again.",
		"self_uninstall_confirm_title":             "Confirm Application Uninstallation",
		"self_uninstall_confirm_message":           "Are you sure you want to completely remove this application and its related files from your computer?",
	},
	"zh-CN": {
		"window_title":                             "AeroGennis A330-300 安装程序 - v2025.8.3.20-Preview",
		"reinstall_button":                         "检查更新/重新安装",
		"aircraft_installed_title":                 "检测到已安装",
		"aircraft_installed_desc":                  "程序检测到 AeroGennis A330-300 已经安装。如果需要，您可以检查更新或重新安装。",
		"manual_path_label":                        "手动覆盖AG330路径",
		"manual_path_placeholder":                  "可选：输入 'AeroGennis Airbus A330-300' 文件夹的完整路径",
		"save_manual_path_button":                  "保存手动路径",
		"manual_path_error":                        "提供的路径无效或不存在。请检查后重试。",
		"save_success_title":                       "已保存",
		"save_manual_path_success":                 "手动路径已成功保存。应用程序已刷新。",
		"setup_welcome":                            "欢迎！请选择您的 X-Plane 12 根目录以开始。",
		"path_placeholder":                         "输入或浏览您的 X-Plane 12 根目录...",
		"browse_button":                            "浏览...",
		"save_continue_button":                     "保存并继续",
		"invalid_xp_path_error":                    "所选目录不是有效的 X-Plane 12 安装目录。",
		"missing_items_label":                      "缺少项目:",
		"save_config_error":                        "无法保存配置",
		"status_ready":                             "准备就绪。请选择一个选项。",
		"tab_aircraft":                             "飞机",
		"tab_liveries":                             "涂装",
		"tab_update_app":                           "更新程序",
		"tab_settings":                             "设置",
		"aircraft_tab_title":                       "安装或更新 AeroGennis A330-300",
		"aircraft_tab_desc":                        "这将下载最新版本的 AeroGennis A330-300 并将其安装到您的 X-Plane 12/Aircraft 目录中。",
		"install_aircraft_button":                  "安装 AeroGennis A330-300",
		"no_livery_selected_title":                 "未选择涂装",
		"no_livery_selected_message":               "请在安装前从列表中勾选一个或多个涂装。",
		"liveries_selected_status":                 "已选择 %d 个涂装。",
		"no_livery_selected_status":                "未选择涂装。请勾选以开始。",
		"update_tab_title":                         "更新此应用程序",
		"update_tab_desc":                          "这将下载此安装程序的最新版本（.exe 文件）。您需要关闭此应用程序并手动运行新程序。",
		"update_tab_warning":                       "注意：此文件未经数字签名，可能会被杀毒软件标记。这是一个误报。",
		"download_latest_button":                   "下载最新应用程序版本",
		"settings_tab_title":                       "应用程序设置",
		"current_path_label":                       "当前 X-Plane 12 路径: %s",
		"change_path_button":                       "更改 X-Plane 12 目录",
		"change_language_button":                   "更改语言",
		"status_creating_temp_dir":                 "正在创建临时目录...",
		"temp_dir_error":                           "创建临时目录失败",
		"status_downloading":                       "正在下载 %s...",
		"aircraft_package":                         "飞机包",
		"download_failed_status":                   "下载失败。",
		"download_error":                           "下载 %s 失败",
		"status_extracting":                        "正在解压文件到 %s...",
		"extraction_failed_status":                 "解压失败。",
		"extraction_error":                         "解压 %s 包失败",
		"install_complete_status":                  "飞机安装完成！",
		"install_success_title":                    "成功",
		"aircraft_install_success_message":         "AeroGennis A330-300 已成功安装！",
		"find_aircraft_dir_error":                  "未能找到有效的 AeroGennis A330 目录。请先安装飞机，或在设置中手动指定路径。",
		"status_downloading_update":                "正在下载新版应用程序...",
		"download_update_error":                    "下载更新失败",
		"update_download_complete_status":          "更新下载成功！",
		"update_download_complete_title":           "下载完成",
		"update_download_complete_message":         "新版本已保存至：\n%s\n请关闭此应用程序并运行新程序。",
		"download_progress_label":                  "下载中... %.2f / %.2f MB (%.2f MB/s)",
		"download_no_progress":                     "警告：内容长度未知。将不显示进度。",
		"extract_progress_label":                   "正在解压: %s",
		"install_selected_liveries_button":         "安装所选涂装",
		"update_livery_list_button":                "更新涂装列表",
		"uninstall_liveries_button":                "卸载涂装...",
		"status_updating_livery_list":              "正在下载最新的涂装列表...",
		"livery_list_download_error":               "下载涂装列表失败",
		"livery_list_read_error":                   "读取下载的涂装列表数据失败",
		"livery_list_save_error":                   "保存涂装列表文件失败",
		"update_success_title":                     "已更新",
		"livery_list_update_success":               "涂装列表已成功更新并重新加载！",
		"livery_list_load_fail":                    "无法加载涂装列表。请尝试更新它。",
		"batch_download_progress_label":            "下载中 (%d/%d): %s",
		"batch_install_complete_status":            "%d 个涂装处理完毕。",
		"batch_install_complete_message":           "%d 个选中的涂装已处理并安装。",
		"scan_liveries_dir_error":                  "扫描涂装目录失败",
		"no_installed_liveries_title":              "未找到涂装",
		"no_installed_liveries_message":            "在预期的目录中没有找到已安装的涂装。",
		"uninstall_dialog_title":                   "选择要卸载的涂装",
		"confirm_button":                           "确认",
		"cancel_button":                            "取消",
		"uninstall_final_confirm_title":            "永久删除",
		"uninstall_final_confirm_message":          "您确定要永久删除这 %d 个选中的文件夹吗？\n\n- %s",
		"uninstall_complete_title":                 "卸载报告",
		"uninstall_report_message":                 "成功删除 %d 个涂装。",
		"uninstall_report_errors":                  "删除 %d 个涂装失败:\n%s",
		"danger_zone_label":                        "危险区域",
		"uninstall_aircraft_button":                "卸载 AeroGennis A330-300",
		"uninstall_aircraft_confirm_title":         "确认卸载机模",
		"uninstall_aircraft_confirm_message":       "这将永久删除位于以下位置的整个机模文件夹：\n\n%s\n\n此操作无法撤销。您确定吗？",
		"uninstall_aircraft_final_confirm_message": "最后警告！所有机模文件，包括任何修改或已添加的涂装，都将被删除。要继续吗？",
		"uninstall_aircraft_error":                 "卸载机模失败",
		"uninstall_aircraft_success_message":       "AeroGennis A330-300 已成功卸载。",
		"self_uninstall_button":                    "卸载此应用程序",
		"self_uninstall_warning":                   "警告：这将移除安装程序本身（.exe）、其配置文件和涂装列表文件。您需要重新下载才能再次使用本程序。",
		"self_uninstall_confirm_title":             "确认卸载应用程序",
		"self_uninstall_confirm_message":           "您确定要从您的电脑上完全移除此应用程序及其相关文件吗？",
	},
	"zh-TW": {
		"window_title":                             "AeroGennis A330-300 安裝程式 - v2025.8.3.20-Preview",
		"reinstall_button":                         "檢查更新/重新安裝",
		"aircraft_installed_title":                 "偵測到已安裝",
		"aircraft_installed_desc":                  "應用程式偵測到 AeroGennis A330-300 已經安裝。如果需要，您可以檢查更新或重新安裝。",
		"manual_path_label":                        "手動覆寫 AG330 路徑",
		"manual_path_placeholder":                  "可選：輸入 'AeroGennis Airbus A330-300' 資料夾的完整路徑",
		"save_manual_path_button":                  "儲存手動路徑",
		"manual_path_error":                        "提供的路徑無效或不存在。請檢查後重試。",
		"save_success_title":                       "已儲存",
		"save_manual_path_success":                 "手動路徑已成功儲存。應用程式已重新整理。",
		"setup_welcome":                            "歡迎！請選擇您的 X-Plane 12 根目錄以開始。",
		"path_placeholder":                         "輸入或瀏覽您的 X-Plane 12 根目錄...",
		"browse_button":                            "瀏覽...",
		"save_continue_button":                     "儲存並繼續",
		"invalid_xp_path_error":                    "所選目錄不是有效的 X-Plane 12 安裝目錄。",
		"missing_items_label":                      "缺少項目:",
		"save_config_error":                        "無法儲存設定",
		"status_ready":                             "準備就緒。請選擇一個選項。",
		"tab_aircraft":                             "飛機",
		"tab_liveries":                             "塗裝",
		"tab_update_app":                           "更新程式",
		"tab_settings":                             "設定",
		"aircraft_tab_title":                       "安裝或更新 AeroGennis A330-300",
		"aircraft_tab_desc":                        "這將會下載最新版本的 AeroGennis A330-300 並將其安裝到您的 X-Plane 12/Aircraft 目錄中。",
		"install_aircraft_button":                  "安裝 AeroGennis A330-300",
		"no_livery_selected_title":                 "未選擇塗裝",
		"no_livery_selected_message":               "請在安裝前從列表中勾選一個或多個塗裝。",
		"liveries_selected_status":                 "已選擇 %d 個塗裝。",
		"no_livery_selected_status":                "未選擇塗裝。請勾選以開始。",
		"update_tab_title":                         "更新此應用程式",
		"update_tab_desc":                          "這將會下載此安裝程式的最新版本（.exe 檔案）。您需要關閉此應用程式並手動執行新程式。",
		"update_tab_warning":                       "注意：此檔案未經數位簽章，可能會被防毒軟體標記。這是誤報。",
		"download_latest_button":                   "下載最新應用程式版本",
		"settings_tab_title":                       "應用程式設定",
		"current_path_label":                       "目前 X-Plane 12 路徑: %s",
		"change_path_button":                       "變更 X-Plane 12 目錄",
		"change_language_button":                   "變更語言",
		"status_creating_temp_dir":                 "正在建立暫存目錄...",
		"temp_dir_error":                           "建立暫存目錄失敗",
		"status_downloading":                       "正在下載 %s...",
		"aircraft_package":                         "飛機套件",
		"download_failed_status":                   "下載失敗。",
		"download_error":                           "下載 %s 失敗",
		"status_extracting":                        "正在解壓縮檔案至 %s...",
		"extraction_failed_status":                 "解壓縮失敗。",
		"extraction_error":                         "解壓縮 %s 套件失敗",
		"install_complete_status":                  "飛機安裝完成！",
		"install_success_title":                    "成功",
		"aircraft_install_success_message":         "AeroGennis A330-300 已成功安裝！",
		"find_aircraft_dir_error":                  "未能找到有效的 AeroGennis A330 目錄。請先安裝飛機，或在設定中手動指定路徑。",
		"status_downloading_update":                "正在下載新版應用程式...",
		"download_update_error":                    "下載更新失敗",
		"update_download_complete_status":          "更新下載成功！",
		"update_download_complete_title":           "下載完成",
		"update_download_complete_message":         "新版本已儲存至：\n%s\n請關閉此應用程式並執行新程式。",
		"download_progress_label":                  "下載中... %.2f / %.2f MB (%.2f MB/s)",
		"download_no_progress":                     "警告：內容長度未知。將不會顯示進度。",
		"extract_progress_label":                   "正在解壓縮: %s",
		"install_selected_liveries_button":         "安裝所選塗裝",
		"update_livery_list_button":                "更新塗裝列表",
		"uninstall_liveries_button":                "卸載塗裝...",
		"status_updating_livery_list":              "正在下載最新的塗装列表...",
		"livery_list_download_error":               "下載塗裝列表失敗",
		"livery_list_read_error":                   "讀取下載的塗裝列表資料失敗",
		"livery_list_save_error":                   "儲存塗裝列表檔案失败",
		"update_success_title":                     "已更新",
		"livery_list_update_success":               "塗裝列表已成功更新並重新載入！",
		"livery_list_load_fail":                    "無法載入塗装列表。請嘗試更新它。",
		"batch_download_progress_label":            "下載中 (%d/%d): %s",
		"batch_install_complete_status":            "%d 個塗裝處理完畢。",
		"batch_install_complete_message":           "%d 個選中的塗裝已處理並安裝。",
		"scan_liveries_dir_error":                  "掃描塗裝目錄失敗",
		"no_installed_liveries_title":              "未找到塗裝",
		"no_installed_liveries_message":            "在預期的目錄中沒有找到已安裝的塗裝。",
		"uninstall_dialog_title":                   "選擇要卸載的塗裝",
		"confirm_button":                           "確認",
		"cancel_button":                            "取消",
		"uninstall_final_confirm_title":            "永久刪除",
		"uninstall_final_confirm_message":          "您確定要永久刪除這 %d 個選中的資料夾嗎？\n\n- %s",
		"uninstall_complete_title":                 "卸載報告",
		"uninstall_report_message":                 "成功刪除 %d 個塗裝。",
		"uninstall_report_errors":                  "刪除 %d 個塗裝失敗:\n%s",
		"danger_zone_label":                        "危險區域",
		"uninstall_aircraft_button":                "卸載 AeroGennis A330-300",
		"uninstall_aircraft_confirm_title":         "確認卸載機模",
		"uninstall_aircraft_confirm_message":       "這將永久刪除位於以下位置的整個機模資料夾：\n\n%s\n\n此操作無法復原。您確定嗎？",
		"uninstall_aircraft_final_confirm_message": "最終警告！所有機模檔案，包含任何修改或已新增的塗裝，都將被刪除。要繼續嗎？",
		"uninstall_aircraft_error":                 "卸載機模失敗",
		"uninstall_aircraft_success_message":       "AeroGennis A330-300 已成功卸載。",
		"self_uninstall_button":                    "卸載此應用程式",
		"self_uninstall_warning":                   "警告：這將移除安裝程式本身（.exe）、其設定檔和塗裝列表檔案。您需要重新下載才能再次使用本程式。",
		"self_uninstall_confirm_title":             "確認卸載應用程式",
		"self_uninstall_confirm_message":           "您確定要從您的電腦上完全移除此應用程式及其相關檔案嗎？",
	},
	"fr-FR": {
		"window_title":                             "Installeur AeroGennis A330-300 - v2025.8.3.20-Preview",
		"reinstall_button":                         "Vérifier les mises à jour / Réinstaller",
		"aircraft_installed_title":                 "Installation Détectée",
		"aircraft_installed_desc":                  "L'application a détecté que l'AeroGennis A330-300 est déjà installé. Vous pouvez vérifier les mises à jour ou le réinstaller si nécessaire.",
		"manual_path_label":                        "Forcer le chemin manuel de l'AG330",
		"manual_path_placeholder":                  "Optionnel : Entrez le chemin complet vers le dossier 'AeroGennis Airbus A330-300'",
		"save_manual_path_button":                  "Enregistrer le chemin manuel",
		"manual_path_error":                        "Le chemin fourni est invalide ou n'existe pas. Veuillez vérifier et réessayer.",
		"save_success_title":                       "Enregistré",
		"save_manual_path_success":                 "Le chemin manuel a été enregistré avec succès. L'application a été actualisée.",
		"setup_welcome":                            "Bienvenue ! Veuillez sélectionner votre répertoire racine de X-Plane 12 pour commencer.",
		"path_placeholder":                         "Entrez ou parcourez jusqu'à votre répertoire racine de X-Plane 12...",
		"browse_button":                            "Parcourir...",
		"save_continue_button":                     "Enregistrer et Continuer",
		"invalid_xp_path_error":                    "Le répertoire sélectionné n'est pas une installation valide de X-Plane 12.",
		"missing_items_label":                      "Éléments manquants :",
		"save_config_error":                        "Échec de la sauvegarde de la configuration",
		"status_ready":                             "Prêt. Sélectionnez une option.",
		"tab_aircraft":                             "Avion",
		"tab_liveries":                             "Livrées",
		"tab_update_app":                           "Mettre à jour l'app",
		"tab_settings":                             "Paramètres",
		"aircraft_tab_title":                       "Installer ou Mettre à Jour l'AeroGennis A330-300",
		"aircraft_tab_desc":                        "Ceci téléchargera la dernière version de l'AeroGennis A330-300 et l'installera dans votre répertoire X-Plane 12/Aircraft.",
		"install_aircraft_button":                  "Installer l'AeroGennis A330-300",
		"no_livery_selected_title":                 "Aucune Livrée Sélectionnée",
		"no_livery_selected_message":               "Veuillez cocher une ou plusieurs livrées dans la liste avant l'installation.",
		"liveries_selected_status":                 "%d livrées sélectionnées.",
		"no_livery_selected_status":                "Aucune livrée sélectionnée. Cochez une case pour commencer.",
		"update_tab_title":                         "Mettre à Jour Cette Application",
		"update_tab_desc":                          "Ceci téléchargera la dernière version de cet installeur (en tant que fichier .exe). Vous devrez fermer cette application et exécuter la nouvelle manuellement.",
		"update_tab_warning":                       "Note : Ce fichier n'est pas signé numériquement et peut être signalé par un logiciel antivirus. C'est un faux positif.",
		"download_latest_button":                   "Télécharger la Dernière Version de l'Application",
		"settings_tab_title":                       "Paramètres de l'Application",
		"current_path_label":                       "Chemin Actuel de X-Plane 12 : %s",
		"change_path_button":                       "Changer le Répertoire de X-Plane 12",
		"change_language_button":                   "Changer de Langue",
		"status_creating_temp_dir":                 "Création du répertoire temporaire...",
		"temp_dir_error":                           "Échec de la création du répertoire temporaire",
		"status_downloading":                       "Téléchargement de %s...",
		"aircraft_package":                         "le pack de l'avion",
		"download_failed_status":                   "Téléchargement échoué.",
		"download_error":                           "Échec du téléchargement de %s",
		"status_extracting":                        "Extraction des fichiers vers %s...",
		"extraction_failed_status":                 "Extraction échouée.",
		"extraction_error":                         "Échec de l'extraction du pack %s",
		"install_complete_status":                  "Installation de l'avion terminée !",
		"install_success_title":                    "Succès",
		"aircraft_install_success_message":         "L'AeroGennis A330-300 a été installé avec succès !",
		"find_aircraft_dir_error":                  "Impossible de trouver un répertoire valide pour l'AeroGennis A330. Veuillez d'abord installer l'avion ou définir le chemin manuellement dans les Paramètres.",
		"status_downloading_update":                "Téléchargement de la nouvelle version de l'application...",
		"download_update_error":                    "Échec du téléchargement de la mise à jour",
		"update_download_complete_status":          "Mise à jour téléchargée avec succès !",
		"update_download_complete_title":           "Téléchargement Terminé",
		"update_download_complete_message":         "Nouvelle version enregistrée dans :\n%s\nVeuillez fermer cette application et exécuter la nouvelle.",
		"download_progress_label":                  "Téléchargement... %.2f / %.2f Mo (%.2f Mo/s)",
		"download_no_progress":                     "Avertissement : Longueur du contenu inconnue. La progression ne sera pas affichée.",
		"extract_progress_label":                   "Extraction : %s",
		"install_selected_liveries_button":         "Installer les Livrées Sélectionnées",
		"update_livery_list_button":                "Mettre à Jour la Liste",
		"uninstall_liveries_button":                "Désinstaller des Livrées...",
		"status_updating_livery_list":              "Téléchargement de la dernière liste de livrées...",
		"livery_list_download_error":               "Échec du téléchargement de la liste de livrées",
		"livery_list_read_error":                   "Échec de la lecture des données de la liste de livrées",
		"livery_list_save_error":                   "Échec de l'enregistrement du fichier de la liste de livrées",
		"update_success_title":                     "Mis à Jour",
		"livery_list_update_success":               "La liste de livrées a été mise à jour et rechargée avec succès !",
		"livery_list_load_fail":                    "Impossible de charger la liste de livrées. Veuillez essayer de la mettre à jour.",
		"batch_download_progress_label":            "Téléchargement (%d/%d) : %s",
		"batch_install_complete_status":            "%d livrées traitées.",
		"batch_install_complete_message":           "%d livrées sélectionnées ont été traitées et installées.",
		"scan_liveries_dir_error":                  "Échec de l'analyse du répertoire des livrées",
		"no_installed_liveries_title":              "Aucune Livrée Trouvée",
		"no_installed_liveries_message":            "Aucune livrée installée n'a été trouvée dans le répertoire attendu.",
		"uninstall_dialog_title":                   "Sélectionner les Livrées à Désinstaller",
		"confirm_button":                           "Confirmer",
		"cancel_button":                            "Annuler",
		"uninstall_final_confirm_title":            "Suppression Permanente",
		"uninstall_final_confirm_message":          "Êtes-vous sûr de vouloir supprimer définitivement ces %d dossiers sélectionnés ?\n\n- %s",
		"uninstall_complete_title":                 "Rapport de Désinstallation",
		"uninstall_report_message":                 "%d livrées supprimées avec succès.",
		"uninstall_report_errors":                  "Échec de la suppression de %d livrées :\n%s",
		"danger_zone_label":                        "Zone de Danger",
		"uninstall_aircraft_button":                "Désinstaller l'AeroGennis A330-300",
		"uninstall_aircraft_confirm_title":         "Confirmer la Désinstallation de l'Avion",
		"uninstall_aircraft_confirm_message":       "Ceci supprimera définitivement le dossier entier de l'avion situé à :\n\n%s\n\nCette action est irréversible. Êtes-vous sûr ?",
		"uninstall_aircraft_final_confirm_message": "Dernier avertissement ! Tous les fichiers de l'avion, y compris les modifications ou livrées ajoutées, seront supprimés. Continuer ?",
		"uninstall_aircraft_error":                 "Échec de la désinstallation de l'avion",
		"uninstall_aircraft_success_message":       "L'AeroGennis A330-300 a été désinstallé avec succès.",
		"self_uninstall_button":                    "Désinstaller Cette Application",
		"self_uninstall_warning":                   "AVERTISSEMENT : Ceci supprimera l'exécutable de l'installeur lui-même, ainsi que ses fichiers de configuration et de liste de livrées. Vous devrez le retélécharger pour l'utiliser à nouveau.",
		"self_uninstall_confirm_title":             "Confirmer la Désinstallation de l'Application",
		"self_uninstall_confirm_message":           "Êtes-vous sûr de vouloir supprimer complètement cette application et ses fichiers associés de votre ordinateur ?",
	},
	"ru-RU": {
		"window_title":                             "Установщик AeroGennis A330-300 - v2025.8.3.20-Preview",
		"reinstall_button":                         "Проверить обновления / Переустановить",
		"aircraft_installed_title":                 "Обнаружена Установка",
		"aircraft_installed_desc":                  "Приложение обнаружило, что AeroGennis A330-300 уже установлен. Вы можете проверить наличие обновлений или переустановить при необходимости.",
		"manual_path_label":                        "Переопределить путь к AG330 вручную",
		"manual_path_placeholder":                  "Необязательно: Введите полный путь к папке 'AeroGennis Airbus A330-300'",
		"save_manual_path_button":                  "Сохранить ручной путь",
		"manual_path_error":                        "Указанный путь недействителен или не существует. Пожалуйста, проверьте и попробуйте снова.",
		"save_success_title":                       "Сохранено",
		"save_manual_path_success":                 "Путь, указанный вручную, успешно сохранен. Приложение было обновлено.",
		"setup_welcome":                            "Добро пожаловать! Пожалуйста, выберите корневой каталог X-Plane 12, чтобы начать.",
		"path_placeholder":                         "Введите или выберите корневой каталог X-Plane 12...",
		"browse_button":                            "Обзор...",
		"save_continue_button":                     "Сохранить и Продолжить",
		"invalid_xp_path_error":                    "Выбранный каталог не является действительной установкой X-Plane 12.",
		"missing_items_label":                      "Отсутствующие элементы:",
		"save_config_error":                        "Не удалось сохранить конфигурацию",
		"status_ready":                             "Готово. Выберите действие.",
		"tab_aircraft":                             "Самолёт",
		"tab_liveries":                             "Ливреи",
		"tab_update_app":                           "Обновить ПО",
		"tab_settings":                             "Настройки",
		"aircraft_tab_title":                       "Установить или Обновить AeroGennis A330-300",
		"aircraft_tab_desc":                        "Будет загружена последняя версия AeroGennis A330-300 и установлена в ваш каталог X-Plane 12/Aircraft.",
		"install_aircraft_button":                  "Установить AeroGennis A330-300",
		"no_livery_selected_title":                 "Ливрея не выбрана",
		"no_livery_selected_message":               "Пожалуйста, отметьте одну или несколько ливрей из списка перед установкой.",
		"liveries_selected_status":                 "Выбрано %d ливрей.",
		"no_livery_selected_status":                "Ливрея не выбрана. Поставьте галочку, чтобы начать.",
		"update_tab_title":                         "Обновить Это Приложение",
		"update_tab_desc":                          "Будет загружена последняя версия этого установщика (как .exe файл). Вам нужно будет закрыть это приложение и запустить новое вручную.",
		"update_tab_warning":                       "Примечание: Этот файл не имеет цифровой подписи и может быть помечен антивирусным ПО. Это ложное срабатывание.",
		"download_latest_button":                   "Загрузить Последнюю Версию Приложения",
		"settings_tab_title":                       "Настройки Приложения",
		"current_path_label":                       "Текущий путь X-Plane 12: %s",
		"change_path_button":                       "Изменить Каталог X-Plane 12",
		"change_language_button":                   "Изменить Язык",
		"status_creating_temp_dir":                 "Создание временного каталога...",
		"temp_dir_error":                           "Не удалось создать временный каталог",
		"status_downloading":                       "Загрузка %s...",
		"aircraft_package":                         "пакета самолёта",
		"download_failed_status":                   "Загрузка не удалась.",
		"download_error":                           "Не удалось загрузить %s",
		"status_extracting":                        "Извлечение файлов в %s...",
		"extraction_failed_status":                 "Извлечение не удалось.",
		"extraction_error":                         "Не удалось извлечь пакет %s",
		"install_complete_status":                  "Установка самолёта завершена!",
		"install_success_title":                    "Успешно",
		"aircraft_install_success_message":         "AeroGennis A330-300 был успешно установлен!",
		"find_aircraft_dir_error":                  "Не удалось найти действительный каталог AeroGennis A330. Пожалуйста, сначала установите самолёт или задайте путь вручную в Настройках.",
		"status_downloading_update":                "Загрузка новой версии приложения...",
		"download_update_error":                    "Не удалось загрузить обновление",
		"update_download_complete_status":          "Обновление загружено успешно!",
		"update_download_complete_title":           "Загрузка Завершена",
		"update_download_complete_message":         "Новая версия сохранена в:\n%s\nПожалуйста, закройте это приложение и запустите новое.",
		"download_progress_label":                  "Загрузка... %.2f / %.2f МБ (%.2f МБ/с)",
		"download_no_progress":                     "Предупреждение: Длина содержимого неизвестна. Прогресс не будет показан.",
		"extract_progress_label":                   "Извлечение: %s",
		"install_selected_liveries_button":         "Установить Выбранные Ливреи",
		"update_livery_list_button":                "Обновить Список Ливрей",
		"uninstall_liveries_button":                "Удалить Ливреи...",
		"status_updating_livery_list":              "Загрузка последнего списка ливрей...",
		"livery_list_download_error":               "Не удалось загрузить список ливрей",
		"livery_list_read_error":                   "Не удалось прочитать данные загруженного списка ливрей",
		"livery_list_save_error":                   "Не удалось сохранить файл списка ливрей",
		"update_success_title":                     "Обновлено",
		"livery_list_update_success":               "Список ливрей был успешно обновлён и перезагружен!",
		"livery_list_load_fail":                    "Не удалось загрузить список ливрей. Попробуйте обновить его.",
		"batch_download_progress_label":            "Загрузка (%d/%d): %s",
		"batch_install_complete_status":            "%d ливрей обработано.",
		"batch_install_complete_message":           "%d выбранных ливрей были обработаны и установлены.",
		"scan_liveries_dir_error":                  "Не удалось сканировать каталог ливрей",
		"no_installed_liveries_title":              "Ливреи не найдены",
		"no_installed_liveries_message":            "Установленные ливреи не найдены в ожидаемом каталоге.",
		"uninstall_dialog_title":                   "Выберите Ливреи для Удаления",
		"confirm_button":                           "Подтвердить",
		"cancel_button":                            "Отмена",
		"uninstall_final_confirm_title":            "Окончательное Удаление",
		"uninstall_final_confirm_message":          "Вы уверены, что хотите окончательно удалить эти %d выбранных папок?\n\n- %s",
		"uninstall_complete_title":                 "Отчёт об Удалении",
		"uninstall_report_message":                 "Успешно удалено %d ливрей.",
		"uninstall_report_errors":                  "Не удалось удалить %d ливрей:\n%s",
		"danger_zone_label":                        "Опасная Зона",
		"uninstall_aircraft_button":                "Удалить AeroGennis A330-300",
		"uninstall_aircraft_confirm_title":         "Подтвердить Удаление Самолёта",
		"uninstall_aircraft_confirm_message":       "Это окончательно удалит всю папку самолёта, расположенную по адресу:\n\n%s\n\nЭто действие нельзя отменить. Вы уверены?",
		"uninstall_aircraft_final_confirm_message": "Последнее предупреждение! Все файлы самолёта, включая любые модификации или добавленные ливреи, будут удалены. Продолжить?",
		"uninstall_aircraft_error":                 "Не удалось удалить самолёт",
		"uninstall_aircraft_success_message":       "AeroGennis A330-300 был успешно удалён.",
		"self_uninstall_button":                    "Удалить Это Приложение",
		"self_uninstall_warning":                   "ПРЕДУПРЕЖДЕНИЕ: Это удалит исполняемый файл установщика, а также его конфигурационные файлы и файлы списка ливрей. Вам нужно будет повторно загрузить приложение, чтобы использовать его снова.",
		"self_uninstall_confirm_title":             "Подтвердить Удаление Приложения",
		"self_uninstall_confirm_message":           "Вы уверены, что хотите полностью удалить это приложение и связанные с ним файлы с вашего компьютера?",
	},
}

func loadTranslations(state *AppState) {
	if trans, ok := translations[state.language]; ok {
		state.translations = trans
	} else {
		state.translations = translations["en-US"] // fallback to English
	}
}
