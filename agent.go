package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path"
	"syscall"
	"time"
	"unsafe"

	gops "github.com/mitchellh/go-ps"
	"golang.org/x/sys/windows"
)

var (
	mod                          = windows.NewLazyDLL("user32.dll")
	procGetForegroundWindow      = mod.NewProc("GetForegroundWindow")
	procGetWindowTextLengthW     = mod.NewProc("GetWindowTextLengthW")
	procGetWindowTextW           = mod.NewProc("GetWindowTextW")
	procGetWindowThreadProcessId = mod.NewProc("GetWindowThreadProcessId")
)

type Entry struct {
	Executable          string
	Window              string
	Start               string
	DurationMillisecond time.Duration
	Date                string
	Day                 string
	TZ                  string
	Name                string
	Username            string
}

func getWindowText(hwnd uintptr) string {
	textLen, _, _ := procGetWindowTextLengthW.Call(hwnd)
	textLen += 1

	buffer := make([]uint16, textLen)
	procGetWindowTextW.Call(uintptr(hwnd),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(textLen))

	return syscall.UTF16ToString(buffer)
}

func getWindowThreadProcessID(hwnd uintptr) (uintptr, uintptr) {
	var processID uintptr
	ret, _, _ := procGetWindowThreadProcessId.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&processID)))

	return uintptr(ret), processID
}

func getProcessExecutable(hwnd uintptr) string {
	_, pid := getWindowThreadProcessID(hwnd)
	process, _ := gops.FindProcess(int(pid))
	return process.Executable()
}

func save(text string, filename string) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	if _, err = f.WriteString(text + "\n"); err != nil {
		panic(err)
	}
}

func main() {
	start := time.Now()
	date := start.Format("2006-01-02")
	var prevText string
	var prevHwnd uintptr
	for {
		if hwnd, _, _ := procGetForegroundWindow.Call(); hwnd != 0 {
			text := getWindowText(hwnd)
			if text != prevText {
				exe := getProcessExecutable(prevHwnd)
				user, _ := user.Current()
				dur := time.Since(start) / time.Millisecond
				entry := &Entry{
					Executable:          exe,
					Window:              prevText,
					Start:               start.Format("15:04:05"),
					DurationMillisecond: dur,
					Date:                date,
					Day:                 start.Format("Monday"),
					TZ:                  start.Format("MST"),
					Name:                user.Name,
					Username:            user.Username,
				}
				b, _ := json.Marshal(entry)
				start = time.Now()
				path := path.Join(user.HomeDir, "AppData/Roaming/TimeTrackerLogs/", date+".txt")
				save(string(b), path)
				fmt.Println(string(b))
			}
			prevText = text
			prevHwnd = hwnd

		}
		time.Sleep(time.Second)
	}
}
