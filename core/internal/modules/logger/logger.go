package logger

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	jobque "core/utils/job-que"

	sdkutils "github.com/flarehotspot/sdk-utils"
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
	queId       = jobque.NewJobQue[any]()
	LineCount   atomic.Int64
	logFilePath = filepath.Join(sdkutils.PathTmpDir, "logs", logFilename)
)

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

	_, file, line, ok := runtime.Caller(calldepth)
	if !ok {
		log.Println("Core Logger: Cannot retrieve caller")
	}

	return
}

// Returns the total number of lines of the current log file
func GetLogLines(logFile string) int {
	result, err := queId.Exec(func() (any, error) {
		logFilePathToRead := filepath.Join(sdkutils.PathTmpDir, "logs", logFile)

		file, err := os.Open(logFilePathToRead)
		if err != nil {
			log.Println(err)
			return 0, err
		}
		defer file.Close()

		// get log's lines count
		logLines, err := lineCounter(file)
		if err != nil {
			log.Println(err)
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
	result, err := queId.Exec(func() (any, error) {
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
				log.Println("Core Logger: error inside readlogs for loop ", err)
				return nil, err
			}

			// read of line successful
			dataInLine, err := sdkutils.ParseLineAsArray(l)
			if err != nil {
				log.Println("Core Logger: error parsing raw log file to wsv: ", err)
				return nil, err
			}

			parsedlog, err := parseLog(dataInLine)
			if err != nil {
				log.Println("Core Logger: error parsing log file to flarelog format: ", err)
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
	_, err := queId.Exec(func() (any, error) {
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
	// log.Println("Logline: ", logLine)
	logLength := len(logLine)

	// check if valid flare log file
	if logLength < flarelogBaseMetadataCount {
		return nil, errors.New("Core Logger: invalid flarelog format")
	}

	pathRaw := logLine[9] // raw file path
	relativePath := sdkutils.StripRootPath(pathRaw)
	subpaths := strings.Split(relativePath, "/")

	var plugin, filename, filepluginpath, pluginpath string

	if subpaths[0] == "core" {
		plugin = "core"
		filename = subpaths[len(subpaths)-1]
		// TODO: show only details for core when in dev mode
		filepluginpath = "******"
		pluginpath = plugin
	} else {
		plugin = subpaths[1]
		filename = subpaths[len(subpaths)-1]
		filepluginpath = strings.Join(subpaths[2:], "/")
		pluginpath = strings.Join(subpaths[:1], "/")
	}

	var pluginInfo sdkutils.PluginInfo
	err := sdkutils.JsonRead(filepath.Join(pluginpath, "plugin.json"), &pluginInfo)
	if err != nil {
		return nil, err
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
		Plugin:     pluginInfo.Package,
		PluginPath: filepluginpath,
		Filename:   filename,
		Line:       line,
		Body:       body,
	}, nil
}

// Logs the log info to the console with colored texts
func LogToConsole(file string, line int, level int, title string, body ...any) {
	// date and time data
	now := time.Now()
	hour, min, sec := now.Clock()
	year, month, day := now.Date()
	nano := itoa(now.Nanosecond(), 3)

	metadata := fmt.Sprintf("[%d/%d/%d %d:%d:%d.%d %s:%d]", year, month, day, hour, min, sec, nano, file, line)
	content := colorize(darkGray, metadata)
	content = fmt.Sprintf("%s\n%s %s", content, colorizeLevel(level), title)

	// adding all body key-value pairs if any
	for i, arg := range body {
		value := reflect.ValueOf(arg)
		var str string

		switch value.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			str = fmt.Sprintf("%d", value.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			str = fmt.Sprintf("%d", value.Uint())
		case reflect.Float32, reflect.Float64:
			str = fmt.Sprintf("%f", value.Float())
		case reflect.String:
			str = value.String()
		// Add cases for other types as needed
		default:
			str = fmt.Sprintf("%v", arg)
		}

		// if i is last and is even,
		// means that the value is not given
		if i == len(body)-1 && i%2 == 0 {
			content = fmt.Sprintf("%s\n  \"%s\": -", content, str)
			break
		}

		// if i is key
		if i%2 == 0 {
			content = fmt.Sprintf("%s\n  \"%v\": ", content, str)
			continue
		}

		// if i is value
		content = fmt.Sprintf("%s\"%s\"", content, str)
	}

	fmt.Println(content)
}

// Logs the log info to the specified file path
func LogToFile(file string, line int, level int, title string, body ...any) error {
	_, err := queId.Exec(func() (any, error) {
		logFile, err := openLogFile()
		if err != nil {
			log.Println(err)
			return nil, err
		}

		defer logFile.Close()

		var content [][]string

		// date and time data
		now := time.Now()
		hour, min, sec := now.Clock()
		year, month, day := now.Date()
		nano := itoa(now.Nanosecond(), 3)

		var logInfo []string
		logInfo = append(logInfo, strconv.Itoa(level), title, strconv.Itoa(year), strconv.Itoa(int(month)), strconv.Itoa(day), strconv.Itoa(hour), strconv.Itoa(min), strconv.Itoa(sec), strconv.Itoa(nano), file, strconv.Itoa(line))

		// append body content
		for _, arg := range body {
			value := reflect.ValueOf(arg)
			var str string

			// body content string conversion
			switch value.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				str = fmt.Sprintf("%d", value.Int())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				str = fmt.Sprintf("%d", value.Uint())
			case reflect.Float32, reflect.Float64:
				str = fmt.Sprintf("%f", value.Float())
			case reflect.String:
				str = value.String()
			// Add cases for other types as needed
			default:
				str = fmt.Sprintf("%v", arg)
			}

			logInfo = append(logInfo, str)
		}
		content = append(content, logInfo)

		// serialize
		serialized := sdkutils.Serialize(content)

		// actual logging to file
		_, err = logFile.WriteString(serialized + "\n")
		if err != nil {
			return nil, err
		}

		// increment log file lines by 1
		LineCount.Add(1)

		// nils for the job que
		return nil, nil
	})

	return err
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
	// opening/creating log file
	logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Println("Core Logger: Error creating log file: ", err)
		return nil, err
	}

	return logFile, nil
}
