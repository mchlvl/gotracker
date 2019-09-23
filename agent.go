package main

import (
	"encoding/json"
	"fmt"
	"math"
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
	procGetLastInputInfo         = mod.NewProc("GetLastInputInfo")
	kernel32                     = syscall.MustLoadDLL("kernel32.dll")
	procGetTickCount             = kernel32.MustFindProc("GetTickCount")

	lastInputInfo struct {
		cbSize uint32
		dwTime uint32
	}
)

type Entry struct {
	Executable      string
	Window          string
	Start           string
	DurationSeconds float64
	Date            string
	Day             string
	TZ              string
	Name            string
	Username        string
}
type tagPOINT struct {
	x uintptr
	y uintptr
}
type tagLASTINPUTINFO struct {
	cbSize uintptr
	dwTime uintptr
}

func GetWindowText(hwnd uintptr) string {
	textLen, _, _ := procGetWindowTextLengthW.Call(hwnd)
	textLen += 1

	buffer := make([]uint16, textLen)
	procGetWindowTextW.Call(uintptr(hwnd),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(textLen))

	return syscall.UTF16ToString(buffer)
}

func GetWindowThreadProcessID(hwnd uintptr) (uintptr, uintptr) {
	var processID uintptr
	ret, _, _ := procGetWindowThreadProcessId.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&processID)))

	return uintptr(ret), processID
}

func IdleTime() time.Duration {
	lastInputInfo.cbSize = uint32(unsafe.Sizeof(lastInputInfo))
	currentTickCount, _, _ := procGetTickCount.Call()
	r1, _, err := procGetLastInputInfo.Call(uintptr(unsafe.Pointer(&lastInputInfo)))
	if r1 == 0 {
		panic("error getting last input info: " + err.Error())
	}
	return time.Duration((uint32(currentTickCount) - lastInputInfo.dwTime)) * time.Millisecond
}

func GetProcessExecutable(hwnd uintptr) string {
	_, pid := GetWindowThreadProcessID(hwnd)
	process, _ := gops.FindProcess(int(pid))
	return process.Executable()
}

func Save(text string, filename string) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	if _, err = f.WriteString(text + "\n"); err != nil {
		panic(err)
	}
}

func SaveEvent(
	start time.Time,
	prevHwnd uintptr,
	prevText string,
	penalty time.Duration,
	threshold time.Duration) time.Time {
	dur := time.Since(start) - penalty
	if dur > threshold {
		user, _ := user.Current()
		exe := GetProcessExecutable(prevHwnd)
		entry := &Entry{
			Executable:      exe,
			Window:          prevText,
			Start:           start.Format("15:04:05"),
			DurationSeconds: math.Round(dur.Seconds()*1000) / 1000,
			Date:            start.Format("2006-01-02"),
			Day:             start.Format("Monday"),
			TZ:              start.Format("MST"),
			Name:            user.Name,
			Username:        user.Username,
		}
		b, _ := json.Marshal(entry)
		path := path.Join(user.HomeDir, "AppData/Roaming/TimeTrackerLogs/", start.Format("2006-01-02")+".txt")
		Save(string(b), path)
		// fmt.Println("Saved", time.Now())
	}
	return time.Now()
}

func SaveAwayEvent(start time.Time, penalty time.Duration) time.Time {
	user, _ := user.Current()
	dur := time.Since(start)
	entry := &Entry{
		Executable:      "Away",
		Window:          "Away",
		Start:           start.Format("15:04:05"),
		DurationSeconds: math.Round((dur+penalty).Seconds()*1000) / 1000,
		Date:            start.Format("2006-01-02"),
		Day:             start.Format("Monday"),
		TZ:              start.Format("MST"),
		Name:            user.Name,
		Username:        user.Username,
	}
	b, _ := json.Marshal(entry)
	start = time.Now()
	path := path.Join(user.HomeDir, "AppData/Roaming/TimeTrackerLogs/", start.Format("2006-01-02")+".txt")
	Save(string(b), path)
	// fmt.Println("Saved Away", start)
	return start
}

func main() {
	// user flagged away after this long
	var awayTimeout time.Duration = time.Second * 120
	// activity continues for this many seconds after away
	var awayTolerance time.Duration = time.Second * 30
	// minimum duration to be captured
	var minDuration time.Duration = time.Second * 2

	var iddleFor time.Duration
	var prevText string
	var prevHwnd uintptr
	var isAway bool = false
	var prevAway bool = false
	var saveNow bool = false
	var start time.Time = time.Now()
	user, _ := user.Current()
	var dir string = path.Join(user.HomeDir, "AppData/Roaming/TimeTrackerLogs/")
	os.MkdirAll(dir, os.ModePerm)
	fmt.Println("Saving logs to ", dir)

	for {

		// get foreground window
		if hwnd, _, _ := procGetForegroundWindow.Call(); hwnd != 0 {

			// get the text of the window
			text := GetWindowText(hwnd)

			// check away
			iddleFor = IdleTime()
			isAway = iddleFor >= awayTimeout

			// if the text changed, or changed away status,
			// save entry
			saveNow = (text != prevText) || (isAway != prevAway)
			if saveNow {

				if prevAway && isAway == false {
					// came from away = save away event
					// fmt.Println("Came from away", start)
					start = SaveAwayEvent(start, awayTimeout-awayTolerance)
				} else if isAway && prevAway == false {
					// went away - duration of previous activity cut
					// fmt.Println("Went away", start)
					start = SaveEvent(start, prevHwnd, prevText, awayTimeout-awayTolerance, minDuration)
				} else {
					// window change = save
					// fmt.Println("Window changed", start)
					start = SaveEvent(start, prevHwnd, prevText, 0, minDuration)
				}
			}

			// set new previous
			prevText = text
			prevHwnd = hwnd
			prevAway = isAway
		}

		// fmt.Println("iddleFor", iddleFor)
		time.Sleep(time.Second)
	}
}
