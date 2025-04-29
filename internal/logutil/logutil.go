package logutil

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	// File Loggers
	infoFileLogger    *log.Logger
	warnFileLogger    *log.Logger
	errorFileLogger   *log.Logger
	debugFileLogger   *log.Logger
	fatalFileLogger   *log.Logger
	successFileLogger *log.Logger

	// Console Loggers
	infoConsoleLogger    *log.Logger
	warnConsoleLogger    *log.Logger
	errorConsoleLogger   *log.Logger
	debugConsoleLogger   *log.Logger
	fatalConsoleLogger   *log.Logger
	successConsoleLogger *log.Logger

	logFileHandle io.WriteCloser // Keep handle to close later if needed

	verboseMode bool

	// Color functions
	colorInfo    = color.New(color.FgBlue).SprintfFunc()
	colorWarn    = color.New(color.FgYellow).SprintfFunc()
	colorError   = color.New(color.FgRed).SprintfFunc()
	colorDebug   = color.New(color.FgMagenta).SprintfFunc()
	colorSuccess = color.New(color.FgGreen).SprintfFunc()
	colorFatal   = color.New(color.FgRed, color.Bold).SprintfFunc()
)

// Init initializes the logging system based on the provided configuration parameters.
func Init(logFile string, verbose bool, maxSizeMB, maxBackups, maxAgeDays int, compress bool) {
	verboseMode = verbose

	// --- Setup File Writer (Lumberjack) ---
	fileWriter := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    maxSizeMB, // megabytes
		MaxBackups: maxBackups,
		MaxAge:     maxAgeDays, // days
		Compress:   compress,
		LocalTime:  true, // Use local time for timestamps in backups
	}
	logFileHandle = fileWriter // Store for potential Close() later

	// --- Setup Console Writer (with Color detection) ---
	consoleWriter := io.Writer(os.Stdout)
	isTerm := isatty.IsTerminal(os.Stdout.Fd())
	color.NoColor = !isTerm // Disable color if not a TTY

	// --- Combine Writers ---
	// File output always gets standard log prefixes
	// Console output gets color prefixes, no standard flags initially
	consoleFlags := log.Ldate | log.Ltime // Keep standard timestamp for console

	// --- Create File Loggers (No flags, manual timestamp) ---
	infoFileLogger = log.New(fileWriter, "", 0)
	warnFileLogger = log.New(fileWriter, "", 0)
	errorFileLogger = log.New(fileWriter, "", 0)
	successFileLogger = log.New(fileWriter, "", 0)
	debugFileLogger = log.New(fileWriter, "", 0)
	fatalFileLogger = log.New(fileWriter, "", 0)

	// --- Create Console Loggers (Standard flags, manual color prefix) ---
	infoConsoleLogger = log.New(consoleWriter, "", consoleFlags)
	warnConsoleLogger = log.New(consoleWriter, "", consoleFlags)
	errorConsoleLogger = log.New(consoleWriter, "", consoleFlags)
	successConsoleLogger = log.New(consoleWriter, "", consoleFlags)
	debugConsoleLogger = log.New(consoleWriter, "", consoleFlags)
	fatalConsoleLogger = log.New(consoleWriter, "", consoleFlags)

	// Initial message
	Info("Logging initialized. File: %s, Verbose: %v, TTY: %v", logFile, verboseMode, isTerm)
	// Redirect standard log output to our logger if possible?
	// log.SetOutput(infoLogger.Writer()) // This might mess up prefixes/colors
}

// Info logs an informational message.
func Info(format string, v ...interface{}) {
	logToFile(infoFileLogger, format, v...)
	logToConsole(infoConsoleLogger, colorInfo, "[INFO] ", format, v...)
}

// Warn logs a warning message.
func Warn(format string, v ...interface{}) {
	logToFile(warnFileLogger, format, v...)
	logToConsole(warnConsoleLogger, colorWarn, "[WARN] ", format, v...)
}

// Error logs an error message.
func Error(format string, v ...interface{}) {
	logToFile(errorFileLogger, format, v...)
	logToConsole(errorConsoleLogger, colorError, "[ERROR] ", format, v...)
}

// Success logs a success message.
func Success(format string, v ...interface{}) {
	logToFile(successFileLogger, format, v...)
	logToConsole(successConsoleLogger, colorSuccess, "[SUCCESS] ", format, v...)
}

// Debug logs a debug message only if verbose mode is enabled.
func Debug(format string, v ...interface{}) {
	if verboseMode {
		logToFile(debugFileLogger, format, v...)
		logToConsole(debugConsoleLogger, colorDebug, "[DEBUG] ", format, v...)
	}
}

// Fatal logs an error message and exits the application.
func Fatal(format string, v ...interface{}) {
	logToFile(fatalFileLogger, format, v...)
	logToConsole(fatalConsoleLogger, colorFatal, "[FATAL] ", format, v...)
	os.Exit(1)
}

// Close closes the log file handle if it exists.
func Close() {
	if logFileHandle != nil {
		Info("Closing log file.") // Log before closing
		logFileHandle.Close()
	}
}

// -- Helper Functions --

// logToFile handles formatting for the file logger
func logToFile(logger *log.Logger, format string, v ...interface{}) {
	now := time.Now()
	timestamp := now.Format("[2006/01/02 03:04:05 PM]") // Custom 12-hour format with brackets
	logger.Printf("%s %s", timestamp, fmt.Sprintf(format, v...))
}

// logToConsole handles formatting for the console logger
func logToConsole(logger *log.Logger, colorFunc func(format string, a ...interface{}) string, prefix string, format string, v ...interface{}) {
	// Console logger already has Ldate|Ltime flags set
	logger.Printf("%s %s", colorFunc(prefix), fmt.Sprintf(format, v...))
}
