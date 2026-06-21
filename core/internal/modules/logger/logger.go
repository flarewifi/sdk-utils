package logger

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	jobque "core/utils/job-que"

	sdkutils "github.com/flarewifi/sdk-utils"
)

const (
	reset = "\033[0m"

	darkGray    = 90
	lightRed    = 91
	lightYellow = 93
	lightBlue   = 94

	logFilename = "app.log"
	// infoPrefix  = "[INFO] "
	// debugPrefix = "[DEBUG] "
	// errorPrefix = "[ERROR] "

	flarelogBaseMetadataCount = 10

	// maxLogBytes hard-caps the on-disk log file; rotateKeepBytes is how much of
	// the most recent log is retained when the cap is hit. Sized for routers with
	// limited flash.
	maxLogBytes     = 2 << 20 // 2 MiB
	rotateKeepBytes = 1 << 20 // 1 MiB
)

type LogLine struct {
	Level      int      `json:"level"`
	Title      string   `json:"title"`
	Year       int      `json:"year"`
	Month      int      `json:"month"`
	Day        int      `json:"day"`
	Hour       int      `json:"hour"`
	Minute     int      `json:"minute"`
	Second     int      `json:"second"`
	Nano       int      `json:"nano"`
	DateTime   string   `json:"datetime"`
	FullPath   string   `json:"fullpath"`
	Plugin     string   `json:"plugin"`
	PluginPath string   `json:"pluginpath"`
	Filename   string   `json:"filename"`
	Line       int      `json:"line"`
	Body       []string `json:"body"`
}

var (
	queId       = jobque.NewJobQueue[any]()
	LineCount   atomic.Int64
	logFilePath = filepath.Join(sdkutils.PathTmpDir, "logs", logFilename)

	// pkgCache memoizes plugin.json package lookups (keyed by plugin dir) so
	// parsing log lines doesn't hit the filesystem on every line.
	pkgCache sync.Map // map[string]string

	// Live subscribers for the SSE log tail. publish() fans out each new line
	// to every subscriber non-blockingly (slow consumers drop lines).
	subMu sync.RWMutex
	subs  = make(map[chan *LogLine]struct{})

	// viewWindow caps how many of the most recent file lines the admin viewer
	// reads/filters per request, bounding CPU/memory on large logs.
	viewWindow = 20000
)

// Subscribe registers a live log subscriber for the SSE tail. The returned
// channel receives every subsequently emitted LogLine; call the returned
// function to unsubscribe.
func Subscribe() (<-chan *LogLine, func()) {
	ch := make(chan *LogLine, 256)
	subMu.Lock()
	subs[ch] = struct{}{}
	subMu.Unlock()

	var once sync.Once
	return ch, func() {
		once.Do(func() {
			subMu.Lock()
			delete(subs, ch)
			close(ch)
			subMu.Unlock()
		})
	}
}

func publish(ll *LogLine) {
	subMu.RLock()
	defer subMu.RUnlock()
	for ch := range subs {
		select {
		case ch <- ll:
		default: // drop for slow consumers; never block the logger
		}
	}
}

func init() {
	logdir := filepath.Dir(logFilePath)
	if !sdkutils.FsExists(logdir) {
		os.MkdirAll(logdir, sdkutils.PermDir)
	}

	if !sdkutils.FsExists(logFilePath) {
		os.Create(logFilePath)
	}

	LineCount.Store(int64(GetLogLines(logFilename)))
}

// Returns the file path and line number of the caller function
func GetCallerFileLine(calldepth int) (file string, line int) {
	calldepth++

	_, file, line, _ = runtime.Caller(calldepth)

	return
}

// Returns the total number of lines of the current log file
func GetLogLines(logFile string) int {
	result, err := queId.Exec("GetLogLines", func() (any, error) {
		logFilePathToRead := filepath.Join(sdkutils.PathTmpDir, "logs", logFile)

		file, err := os.Open(logFilePathToRead)
		if err != nil {
			return 0, err
		}
		defer file.Close()

		// get log's lines count
		logLines, err := lineCounter(file)
		if err != nil {
			return 0, err
		}

		return logLines, nil
	})

	if err != nil {
		return 0
	}

	return result.(int)
}

// Returns a map of string : any, formatted based on the log parser
// and an error. Logs returned will be in the range of start to end,
// inclusive. Starts at index 0.
func ReadLogs(start int, end int) ([]*LogLine, error) {
	result, err := queId.Exec("ReadLogs", func() (any, error) {
		logs := []*LogLine{}

		// open logs
		file, err := os.Open(logFilePath)
		if err != nil {
			return nil, err
		}

		defer file.Close()

		rd := bufio.NewReader(file)

		currLine := 0

		for {
			l, err := rd.ReadString('\n')

			if currLine < start {
				currLine++
				continue
			}

			// file has no content left
			if err == io.EOF {
				break
			}

			if err != nil {
				return nil, err
			}

			// read of line successful
			dataInLine, err := sdkutils.ParseLineAsArray(l)
			if err != nil {
				return nil, err
			}

			parsedlog, err := parseLog(dataInLine)
			if err != nil {
				return nil, err
			}

			logs = append(logs, parsedlog)

			if currLine >= end {
				break
			}

			currLine++
		}

		return logs, nil
	})

	if err != nil {
		return nil, err
	}

	logs := result.([]*LogLine)
	return logs, nil
}

func ClearLogs() error {
	_, err := queId.Exec("ClearLogs", func() (any, error) {
		err := os.WriteFile(logFilePath, []byte(""), sdkutils.PermFile)
		return nil, err
	})
	return err
}

// Returns the total number of lines of the specified file
func lineCounter(r io.Reader) (int, error) {
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
}

// Accepts a slice of string and parses as a flare log line
// and returns the parsed string.
func parseLog(logLine []string) (*LogLine, error) {
	logLength := len(logLine)

	// check if valid flare log file
	if logLength < flarelogBaseMetadataCount {
		return nil, errors.New("Core Logger: invalid flarelog format")
	}

	pathRaw := logLine[9] // raw file path
	relativePath := sdkutils.StripRootPath(pathRaw)
	subpaths := strings.Split(relativePath, "/")

	// Caller paths are Go-module-relative, so subpaths[0] is the module name:
	// "core" for the core, and the package id itself for plugins (their go.mod
	// module name equals their package, e.g. "com.flarego.wifi-hotspot"). Derive
	// the package directly from it — no per-line plugin.json read needed for
	// plugins. For core, resolve the real package id from core/plugin.json
	// (cached), which is readable relative to the app's working directory.
	var pkg, filename, filepluginpath string

	filename = subpaths[len(subpaths)-1]
	if subpaths[0] == "core" {
		pkg = resolvePkg("core")
		if pkg == "" {
			pkg = "core"
		}
		filepluginpath = "******"
	} else {
		pkg = subpaths[0]
		if len(subpaths) > 1 {
			filepluginpath = strings.Join(subpaths[1:], "/")
		}
	}

	var body []string
	// check if log has body
	if logLength > flarelogBaseMetadataCount {
		body = logLine[flarelogBaseMetadataCount+1:]
	}

	level := sdkutils.AtoiOrDefault(logLine[0], 0)
	title := logLine[1]
	year := sdkutils.AtoiOrDefault(logLine[2], 0)
	month := sdkutils.AtoiOrDefault(logLine[3], 0)
	day := sdkutils.AtoiOrDefault(logLine[4], 0)
	hour := sdkutils.AtoiOrDefault(logLine[5], 0)
	minute := sdkutils.AtoiOrDefault(logLine[6], 0)
	second := sdkutils.AtoiOrDefault(logLine[7], 0)
	nano := sdkutils.AtoiOrDefault(logLine[8], 0)
	fullPath := sdkutils.StripRootPath(logLine[9])
	line := sdkutils.AtoiOrDefault(logLine[10], 0)

	return &LogLine{
		Level:      level,
		Title:      title,
		Year:       year,
		Month:      month,
		Day:        day,
		Hour:       hour,
		Minute:     minute,
		Second:     second,
		Nano:       nano,
		DateTime:   fmt.Sprintf("%d-%d-%d %d:%d:%d.%d", year, month, day, hour, minute, second, nano),
		FullPath:   fullPath,
		Plugin:     pkg,
		PluginPath: filepluginpath,
		Filename:   filename,
		Line:       line,
		Body:       body,
	}, nil
}

// Emit is the single logging sink. It writes the record to stdout (captured by
// syslog/logread on OpenWRT and `docker logs` in dev), appends it to the
// rotating app.log file (the source for the paginated admin viewer), and
// publishes it to live SSE subscribers. It never touches the database, so it is
// safe to call from inside a DB transaction.
func Emit(level int, file string, line int, title string) {
	logInfo := buildLogInfo(level, file, line, title)

	// stdout — human-readable single line for logread / docker logs.
	fmt.Printf("[%s] %s %s:%d %s\n",
		getLevelAsStr(level), time.Now().Format("2006-01-02 15:04:05"),
		sdkutils.StripRootPath(file), line, title)

	writeLogLine(logInfo)

	// Publish the parsed line to live subscribers (best-effort).
	if ll, err := parseLog(logInfo); err == nil {
		publish(ll)
	}
}

// LogFilter holds the filter/pagination options for the admin log viewer.
type LogFilter struct {
	Package    string // plugin package; "" or "all" = any
	Level      string // "info" | "debug" | "error"; "" or "all" = any
	SearchText string // case-insensitive match against message + package
	Page       int    // 1-based
	PerPage    int
}

// ReadLogsFiltered reads the most recent file lines (up to viewWindow), applies
// the package/level/text filters, orders newest-first, and returns the requested
// page along with the total number of matching lines.
func ReadLogsFiltered(f LogFilter) ([]*LogLine, int, error) {
	totalLines := GetLogLines(logFilename)
	if totalLines == 0 {
		return []*LogLine{}, 0, nil
	}

	start := 0
	if totalLines > viewWindow {
		start = totalLines - viewWindow
	}

	all, err := ReadLogs(start, totalLines)
	if err != nil {
		return nil, 0, err
	}

	levelFilter := -1
	if f.Level != "" && f.Level != "all" {
		levelFilter = levelStrToInt(f.Level)
	}
	search := strings.ToLower(strings.TrimSpace(f.SearchText))

	matched := make([]*LogLine, 0, len(all))
	for _, ll := range all {
		if f.Package != "" && f.Package != "all" && ll.Plugin != f.Package {
			continue
		}
		if levelFilter >= 0 && ll.Level != levelFilter {
			continue
		}
		if search != "" &&
			!strings.Contains(strings.ToLower(ll.Title), search) &&
			!strings.Contains(strings.ToLower(ll.Plugin), search) {
			continue
		}
		matched = append(matched, ll)
	}

	// Newest first.
	for i, j := 0, len(matched)-1; i < j; i, j = i+1, j-1 {
		matched[i], matched[j] = matched[j], matched[i]
	}

	total := len(matched)

	page := f.Page
	if page < 1 {
		page = 1
	}
	perPage := f.PerPage
	if perPage < 1 {
		perPage = 10
	}

	from := (page - 1) * perPage
	if from >= total {
		return []*LogLine{}, total, nil
	}
	to := from + perPage
	if to > total {
		to = total
	}
	return matched[from:to], total, nil
}

// buildLogInfo serializes a log record into the flarelog field array used by the
// file format and parseLog.
func buildLogInfo(level int, file string, line int, title string) []string {
	now := time.Now()
	hour, min, sec := now.Clock()
	year, month, day := now.Date()
	nano := itoa(now.Nanosecond(), 3)

	return []string{
		strconv.Itoa(level), title,
		strconv.Itoa(year), strconv.Itoa(int(month)), strconv.Itoa(day),
		strconv.Itoa(hour), strconv.Itoa(min), strconv.Itoa(sec), strconv.Itoa(nano),
		file, strconv.Itoa(line),
	}
}

// writeLogLine appends a prebuilt log record to the rotating file (serialized,
// one line). The file is hard-capped at maxLogBytes: when exceeded, the oldest
// lines are dropped and only the most recent rotateKeepBytes are kept, so the
// log can never grow without bound. Failures are swallowed so a full/unwritable
// disk never crashes the caller.
func writeLogLine(logInfo []string) {
	queId.Exec("LogToFile", func() (any, error) {
		logFile, err := openLogFile()
		if err != nil {
			return nil, nil
		}
		defer logFile.Close()

		rotateIfNeeded(logFile)

		serialized := sdkutils.Serialize([][]string{logInfo})
		if _, err := logFile.WriteString(serialized + "\n"); err != nil {
			return nil, nil
		}
		LineCount.Add(1)
		return nil, nil
	})
}

// rotateIfNeeded caps the log file at maxLogBytes. Once the file grows past the
// cap it rewrites the file to contain only its most recent rotateKeepBytes
// (trimmed to start on a line boundary), discarding older lines. This bounds
// disk usage while preserving recent history. Must be called holding the queId
// worker (serialized with all other file ops).
func rotateIfNeeded(f *os.File) {
	info, err := f.Stat()
	if err != nil || info.Size() < maxLogBytes {
		return
	}

	size := info.Size()
	start := size - rotateKeepBytes
	if start < 0 {
		start = 0
	}

	buf := make([]byte, size-start)
	if _, err := f.ReadAt(buf, start); err != nil {
		return
	}

	// Drop the leading partial line so the kept tail starts cleanly.
	if start > 0 {
		if i := bytes.IndexByte(buf, '\n'); i >= 0 && i+1 <= len(buf) {
			buf = buf[i+1:]
		}
	}

	if err := f.Truncate(0); err != nil {
		return
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return
	}
	if _, err := f.Write(buf); err != nil {
		return
	}
	LineCount.Store(int64(bytes.Count(buf, []byte{'\n'})))
}

// resolvePkg returns the package name declared in the plugin.json at pluginpath,
// memoized. Returns "" if it can't be read.
func resolvePkg(pluginpath string) string {
	if v, ok := pkgCache.Load(pluginpath); ok {
		return v.(string)
	}
	var info sdkutils.PluginInfo
	if err := sdkutils.JsonRead(filepath.Join(pluginpath, "plugin.json"), &info); err != nil {
		return ""
	}
	pkgCache.Store(pluginpath, info.Package)
	return info.Package
}

func levelStrToInt(level string) int {
	switch strings.ToLower(level) {
	case "info":
		return 0
	case "debug":
		return 1
	case "error":
		return 2
	}
	return -1
}

// Returns a date string with format: YYYYMMdd
// func getLogDateAsStr(log map[string]any) string {
// 	year := reflect.ValueOf(log["year"]).String()
// 	month := reflect.ValueOf(log["month"]).String()
// 	day := reflect.ValueOf(log["day"]).String()
// 	hour := reflect.ValueOf(log["hour"]).String()
// 	min := reflect.ValueOf(log["min"]).String()
// 	sec := reflect.ValueOf(log["sec"]).String()
// 	nano := reflect.ValueOf(log["nano"]).String()

// 	return year + month + day + hour + min + sec + nano
// }

// Helper function to cut an integer's digits to
// desired length
func itoa(i int, wid int) int {
	num := i
	d := 1

	for i >= 10 {
		q := i / 10
		i = q
		d++
	}

	return num / int(math.Pow10(d-wid))
}

// Returns the equivalent log level in string
func getLevelAsStr(level int) string {
	switch level {
	case 0:
		return "INFO"
	case 1:
		return "DEBUG"
	case 2:
		return "ERROR"
	}

	return "INFO"
}

// Returns a string with the desired color
func colorize(colorCode int, v string) string {
	return fmt.Sprintf("\033[%sm%s%s", strconv.Itoa(colorCode), v, reset)
}

// Returns the level in string with dedicated color
func colorizeLevel(level int) string {
	var color int
	switch level {
	case 0:
		color = lightBlue
	case 1:
		color = lightYellow
	case 2:
		color = lightRed
	}
	return colorize(color, getLevelAsStr(level))
}

// Returns the opened log file instance
func openLogFile() (*os.File, error) {
	// O_RDWR so rotateIfNeeded can read the tail before truncating; O_APPEND
	// keeps normal writes at EOF.
	logFile, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	return logFile, nil
}
