package util

import (
	"fmt"
	"io"
	"log"
	"os"
)

type PoolLogger struct {
	l        *log.Logger
	logLevel int
}

var (
	DEBUG = 10
	INFO  = 20
	WARN  = 30
	ERROR = 40
	SHARE = 100
	BLOCK = 101

	logSetLevel = 10

	Debug *PoolLogger
	Info  *PoolLogger
	Warn  *PoolLogger
	Error *PoolLogger

	ShareLog *PoolLogger
	BlockLog *PoolLogger
)

func InitLog(infoFile, errorFile, shareFile, blockFile string, setLevel int) {
	logSetLevel = setLevel
	log.Println("logSetLevel:", setLevel)

	log.Println("infoFile:", infoFile)
	log.Println("errorFile:", errorFile)
	log.Println("shareFile:", shareFile)
	log.Println("blockFile:", blockFile)
	infoFd, err := os.OpenFile(infoFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open info log file:", err)
	}

	errorFd, err := os.OpenFile(errorFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open error log file:", err)
	}

	shareFd, err := os.OpenFile(shareFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open share log file:", err)
	}

	blockFd, err := os.OpenFile(blockFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open block log file:", err)
	}

	Debug = &PoolLogger{log.New(io.MultiWriter(os.Stdout, infoFd), "[D] ", log.Ldate|log.Lmicroseconds|log.Lshortfile), DEBUG}
	Info = &PoolLogger{log.New(io.MultiWriter(os.Stdout, infoFd), "[I] ", log.Ldate|log.Lmicroseconds|log.Lshortfile), INFO}
	Warn = &PoolLogger{log.New(io.MultiWriter(os.Stderr, infoFd, errorFd), "[W] ", log.Ldate|log.Lmicroseconds|log.Lshortfile), WARN}
	Error = &PoolLogger{log.New(io.MultiWriter(os.Stderr, infoFd, errorFd), "[E] ", log.Ldate|log.Lmicroseconds|log.Lshortfile), ERROR}

	ShareLog = &PoolLogger{log.New(io.MultiWriter(shareFd, os.Stdout), "[S]", log.Ldate|log.Lmicroseconds), SHARE}
	BlockLog = &PoolLogger{log.New(io.MultiWriter(blockFd, os.Stdout), "[B]", log.Ldate|log.Lmicroseconds), BLOCK}
}

func (l *PoolLogger) Print(v ...interface{}) {
	if logSetLevel <= l.logLevel {
		_ = l.l.Output(2, fmt.Sprint(v...))
	}
}

func (l *PoolLogger) Println(v ...interface{}) {
	if logSetLevel <= l.logLevel {
		_ = l.l.Output(2, fmt.Sprintln(v...))
	}
}

func (l *PoolLogger) Printf(format string, v ...interface{}) {
	if logSetLevel <= l.logLevel {
		_ = l.l.Output(2, fmt.Sprintf(format, v...))
	}
}

func (l *PoolLogger) Fatal(v ...interface{}) {
	_ = l.l.Output(2, fmt.Sprint(v...))
	os.Exit(1)
}

func (l *PoolLogger) Fatalln(v ...interface{}) {
	_ = l.l.Output(2, fmt.Sprintln(v...))
	os.Exit(1)
}

func (l *PoolLogger) Fatalf(format string, v ...interface{}) {
	_ = l.l.Output(2, fmt.Sprintf(format, v...))
	os.Exit(1)
}

func (l *PoolLogger) Panic(v ...interface{}) {
	s := fmt.Sprint(v...)
	_ = l.l.Output(2, s)
	panic(s)
}

func (l *PoolLogger) Panicln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.l.Output(2, s)
	panic(s)
}

func (l *PoolLogger) Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	_ = l.l.Output(2, s)
	panic(s)
}
