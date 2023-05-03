package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

var logger *log.Logger

func init() {
	path, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	f, err := os.OpenFile(path+"/shlev_net.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}

	logger = log.New(f, "", 0)
	Init("logger init success!")
	fmt.Println("logger init success!")
}

func setPrefix(level string) {
	_, file, line, ok := runtime.Caller(2)
	total := ""
	if ok {
		total = fmt.Sprintf("[%s][%s:%d]", level, filepath.Base(file), line)
	} else {
		total = fmt.Sprintf("[%s]", level)
	}
	logger.SetPrefix(total)
}

func Init(v ...any) {
	setPrefix("INIT")
	logger.Println(v...)
}

func DebugF(fmt string, v ...any) {
	setPrefix("DEBUG")
	logger.Printf(fmt, v...)
}

func Debug(v ...any) {
	setPrefix("DEBUG")
	logger.Println(v...)
}

func WarnF(fmt string, v ...any) {
	setPrefix("WARN")
	logger.Printf(fmt, v...)
}

func Warn(v ...any) {
	setPrefix("WARN")
	logger.Println(v...)
}

func ErrorF(fmt string, v ...any) {
	setPrefix("ERROR")
	logger.Printf(fmt, v...)
}

func Error(v ...any) {
	setPrefix("ERROR")
	logger.Println(v...)
}

func InfoF(fmt string, v ...any) {
	setPrefix("INFO")
	logger.Printf(fmt, v...)
}

func Info(v ...any) {
	setPrefix("INFO")
	logger.Println(v...)
}

func FatalF(fmt string, v ...any) {
	setPrefix("FATAL")
	logger.Printf(fmt, v...)
}

func Fatal(v ...any) {
	setPrefix("FATAL")
	logger.Fatalln(v...)
}
