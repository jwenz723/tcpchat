package main

import (
	"testing"
	"io/ioutil"
	"fmt"
	"os"
)

func TestNewConfig(t *testing.T) {
	testCases := map[string]struct{
		eHTTPAddress string
		eHTTPPort int
		eLogDirectory string
		eLogJSON bool
		eLogLevel string
		eLogLevelError string
		eTCPAddress string
		eTCPPort int
	} {
		"default values": {"", 8080, "logs", false, "info", "", "", 6000},
		"bad log level": {"", 8080, "logs", false, "bad level", "not a valid logrus Level: \"bad level\"", "", 6000},
		"custom values": {"myhttp", 123, "mylogs", true, "debug", "", "mytcp", 2000},
	}

	for k, v := range testCases {
		file := "test.yml"

		yml := fmt.Sprintf(`HTTPAddress: %s
HTTPPort: %d
LogDirectory: %s
LogJSON: %v
LogLevel: %s
TCPAddress: %s
TCPPort: %d`, v.eHTTPAddress, v.eHTTPPort, v.eLogDirectory, v.eLogJSON, v.eLogLevel, v.eTCPAddress, v.eTCPPort)
		ioutil.WriteFile(file, []byte(yml), 0777)

		con, err := NewConfig(file)
		if err != nil {
			if v.eLogLevelError != "" {
				if v.eLogLevelError != err.Error() {
					t.Errorf("NewConfig expected error does not match actual error.\n\tExpected: %s\n\tActual: %s", v.eLogLevelError, err)
				}
			} else {
				t.Errorf("%s: NewConfig failed to process file %s -> %s", k, file, err)
			}
		} else if con == nil {
			t.Errorf("%s: NewConfig returned nil con", k)
		}

		if v.eLogLevelError == "" {
			if con.HTTPAddress != v.eHTTPAddress {
				t.Errorf("%s: HTTPAddress expected (%s) differed from actual (%s)", k, v.eHTTPAddress, con.HTTPAddress)
			}

			if con.HTTPPort != v.eHTTPPort {
				t.Errorf("%s: HTTPPort expected (%d) differed from actual (%d)", k, v.eHTTPPort, con.HTTPPort)
			}

			if con.LogDirectory != v.eLogDirectory {
				t.Errorf("%s: LogDirectory expected (%s) differed from actual (%s)", k, v.eLogDirectory, con.LogDirectory)
			}

			if con.LogJSON != v.eLogJSON {
				t.Errorf("%s: LogJSON expected (%v) differed from actual (%v)", k, v.eLogJSON, con.LogJSON)
			}

			if con.LogLevel != v.eLogLevel {
				t.Errorf("%s: LogLevel expected (%s) differed from actual (%s)", k, v.eLogLevel, con.LogLevel)
			}

			if con.TCPAddress != v.eTCPAddress {
				t.Errorf("%s: TCPAddress expected (%s) differed from actual (%s)", k, v.eTCPAddress, con.TCPAddress)
			}

			if con.TCPPort != v.eTCPPort {
				t.Errorf("%s: TCPPort expected (%d) differed from actual (%d)", k, v.eTCPPort, con.TCPPort)
			}
		}

		os.Remove(file)
	}
}