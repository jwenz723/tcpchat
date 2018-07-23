package main

import (
	"testing"
	"github.com/sirupsen/logrus"
	"os"
	"fmt"
	"path/filepath"
	"time"
	"regexp"
	"io/ioutil"
	"reflect"
)

var LogDirectory string

func TestMain(m *testing.M) {
	setup(m)
	code := m.Run()
	shutdown(m)
	os.Exit(code)
}

func setup(m *testing.M) {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Printf("setup() -> error while getting directory -> %s\n", err)
		panic(m)
	}
	LogDirectory = filepath.Join(dir, "tests")

	// create a directory for test logs to go in
	if _, err := os.Stat(LogDirectory); os.IsNotExist(err) {
		err = os.MkdirAll(LogDirectory, 0777)
		if err != nil {
			fmt.Printf("setup() -> error while creating test log directory -> %s\n", err)
			panic(m)
		}
		err = os.Chmod(LogDirectory, 0777)
		if err != nil {
			fmt.Printf("setup() -> error while performing chmod on test log directory -> %s", err)
			panic(m)
		}
	}
}

func shutdown(m *testing.M) {
	err := os.RemoveAll(LogDirectory)
	if err != nil {
		fmt.Printf("shutdown() -> error deleting test log directory -> %s\n", err)
		panic(m)
	}
}

func TestInitLogging(t *testing.T) {
	jsonFormatterType := reflect.TypeOf((*logrus.JSONFormatter)(nil)).Elem()
	textFormatterType := reflect.TypeOf((*logrus.TextFormatter)(nil)).Elem()

	var testTable = map[string]struct {
		expectedFormatter	   reflect.Type
		expectedLogLevel       logrus.Level
		expectedLogOutputRegex string
		logDirectory           string
		logLevel               string
		jsonOutput             bool
	}{
		"debug-level json test":     {jsonFormatterType,logrus.DebugLevel, "{.*this is an error.*\\n{.*this is a warn.*}\\n{.*this is an info.*}\\n{.*this is a debug.*}", "inlog", "debug", true},
		"error-level json test":     {jsonFormatterType,logrus.ErrorLevel, "this is an error", "inlog", "error", true},
		"debug-level non-json test": {textFormatterType,logrus.DebugLevel, "time.*this is an error\"\ntime.*this is a warn\"\ntime.*this is an info\"\ntime.*this is a debug\"", "inlog", "debug", false},
		"info-level non-json test":  {textFormatterType,logrus.InfoLevel, "time.*this is an error\"\ntime.*this is a warn\"\ntime.*this is an info\"", "inlog", "info", false},
		"warn-level non-json test":  {textFormatterType,logrus.WarnLevel, "time.*this is an error\"\ntime.*this is a warn\"", "inlog", "warn", false},
		"error-level non-json test": {textFormatterType,logrus.ErrorLevel, "time.*this is an error\"", "inlog", "error", false},
		"new log dir test":          {textFormatterType,logrus.InfoLevel, "time.*this is an error\"\ntime.*this is a warn\"\ntime.*this is an info\"", "inlog\\sub", "info", false},
		"bad log level test":        {textFormatterType,logrus.InfoLevel, "time.*this is an error\"\ntime.*this is a warn\"\ntime.*this is an info\"", "inlog", "invalid level", false},
		"stdout test":               {textFormatterType,logrus.InfoLevel, "", "", "info", false},
	}

	for testName, v := range testTable {
		ld := v.logDirectory
		if v.logDirectory != "" {
			ld = filepath.Join(LogDirectory, v.logDirectory)
		}

		logger, teardown, err := InitLogging(ld, v.logLevel, v.jsonOutput)
		if err != nil {
			t.Errorf("%s[%s] -> InitLogging() experienced error an error -> %s", t.Name(), testName, err)
		}
		defer func() {
			if err := teardown(); err != nil {
				t.Errorf("%s[%s] -> %s", t.Name(), testName, err)
			}
		}()

		if logger == nil {
			t.Errorf("%s[%s] -> logger failed to initialize", t.Name(), testName)
			continue
		}

		// test setting of level
		if logger.Level != v.expectedLogLevel {
			t.Errorf("%s[%s] -> expectedFile result (%s) does not equal actual result (%s)", t.Name(), testName, v.expectedLogLevel, logger.Level)
		}

		// test JSON setting
		actualFormatter := reflect.ValueOf(logger.Formatter).Elem().Type()
		if actualFormatter != v.expectedFormatter {
			t.Errorf("%s[%s] -> expected formatter (%s) does not equal actual formatter (%s)", t.Name(), testName, v.expectedFormatter, actualFormatter)
		}

		if v.logDirectory == "" {
			if logger.Out != os.Stdout {
				t.Errorf("%s[%s] -> Unexpected logger.Out value, got %s. Expected os.Stdout.", t.Name(), testName, reflect.ValueOf(logger.Out).Elem().Type())
			}
		} else if logger.Out == os.Stdout && v.logDirectory != "" {
			t.Errorf("%s[%s] -> Unexpected logger.Out value, got %v", t.Name(), testName, logger.Out)
		} else {
			// write an entry into the file with each log level
			logger.Error("this is an error")
			logger.Warn("this is a warn")
			logger.Info("this is an info")
			logger.Debug("this is a debug")
			teardown()

			// Read the contents of the logfile that was just written to
			b, err := ioutil.ReadFile(filepath.Join(ld,fmt.Sprintf("%s%s", time.Now().Local().Format("20060102"), ".log")))
			if err != nil {
				t.Errorf("%s[%s] -> Failed to read contents of log file", t.Name(), testName)
			}

			// test if the lines written were what was expected
			var re = regexp.MustCompile(v.expectedLogOutputRegex)
			loc := re.FindAllIndex(b, -1)
			if len(loc) > 1 {
				t.Errorf("%s[%s] -> expected file contents does not equal actual file contents.\n\t\tExpected Pattern: %s\n\t\tActual: %s\n", t.Name(), testName, v.expectedLogOutputRegex, b)
			} else if v.expectedLogOutputRegex == "" {
				t.Errorf("%s[%s] -> expectedLogOutputRegex cannot be set to \"\"", t.Name(), testName)
			}

			// delete the generated directory and files
			err = os.RemoveAll(ld)
			if err != nil {
				t.Errorf("%s[%s] -> error deleting files in logDirectory (%s) -> %s", t.Name(), testName, ld, err)
			}
		}
	}
}