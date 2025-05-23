/*
Copyright 2019 The Vitess Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// You can modify this file to hook up a different logging library instead of glog.
// If you adapt to a different logging framework, you may need to use that
// framework's equivalent of *Depth() functions so the file and line number printed
// point to the real caller instead of your adapter function.

package log

import (
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/golang/glog"
	"github.com/spf13/pflag"

	"vitess.io/vitess/go/vt/utils"
)

// Level is used with V() to test log verbosity.
type Level = glog.Level

var (
	// V quickly checks if the logging verbosity meets a threshold.
	V = glog.V

	// Flush ensures any pending I/O is written.
	Flush = glog.Flush

	// Info formats arguments like fmt.Print.
	Info = glog.Info
	// Infof formats arguments like fmt.Printf.
	Infof = glog.Infof
	// InfoDepth formats arguments like fmt.Print and uses depth to choose which call frame to log.
	InfoDepth = glog.InfoDepth

	// Warning formats arguments like fmt.Print.
	Warning = glog.Warning
	// Warningf formats arguments like fmt.Printf.
	Warningf = glog.Warningf
	// WarningDepth formats arguments like fmt.Print and uses depth to choose which call frame to log.
	WarningDepth = glog.WarningDepth

	// Error formats arguments like fmt.Print.
	Error = glog.Error
	// Errorf formats arguments like fmt.Printf.
	Errorf = glog.Errorf
	// ErrorDepth formats arguments like fmt.Print and uses depth to choose which call frame to log.
	ErrorDepth = glog.ErrorDepth

	// Exit formats arguments like fmt.Print.
	Exit = glog.Exit
	// Exitf formats arguments like fmt.Printf.
	Exitf = glog.Exitf
	// ExitDepth formats arguments like fmt.Print and uses depth to choose which call frame to log.
	ExitDepth = glog.ExitDepth

	// Fatal formats arguments like fmt.Print.
	Fatal = glog.Fatal
	// Fatalf formats arguments like fmt.Printf
	Fatalf = glog.Fatalf
	// FatalDepth formats arguments like fmt.Print and uses depth to choose which call frame to log.
	FatalDepth = glog.FatalDepth
)

// RegisterFlags installs log flags on the given FlagSet.
//
// `go/cmd/*` entrypoints should either use servenv.ParseFlags(WithArgs)? which
// calls this function, or call this function directly before parsing
// command-line arguments.
func RegisterFlags(fs *pflag.FlagSet) {
	flagVal := logRotateMaxSize{
		val: fmt.Sprintf("%d", atomic.LoadUint64(&glog.MaxSize)),
	}
	utils.SetFlagVar(fs, &flagVal, "log-rotate-max-size", "size in bytes at which logs are rotated (glog.MaxSize)")
}

// logRotateMaxSize implements pflag.Value and is used to
// try and provide thread-safe access to glog.MaxSize.
type logRotateMaxSize struct {
	val string
}

func (lrms *logRotateMaxSize) Set(s string) error {
	maxSize, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return err
	}
	atomic.StoreUint64(&glog.MaxSize, maxSize)
	lrms.val = s
	return nil
}

func (lrms *logRotateMaxSize) String() string {
	return lrms.val
}

func (lrms *logRotateMaxSize) Type() string {
	return "uint64"
}

type PrefixedLogger struct {
	prefix string
}

func NewPrefixedLogger(prefix string) *PrefixedLogger {
	return &PrefixedLogger{prefix: prefix + ": "}
}

func (pl *PrefixedLogger) V(level glog.Level) glog.Verbose {
	return V(level)
}

func (pl *PrefixedLogger) Flush() {
	Flush()
}

func (pl *PrefixedLogger) Info(args ...any) {
	args = append([]interface{}{pl.prefix}, args...)
	Info(args...)
}

func (pl *PrefixedLogger) Infof(format string, args ...any) {
	args = append([]interface{}{pl.prefix}, args...)
	Infof("%s"+format, args...)
}

func (pl *PrefixedLogger) InfoDepth(depth int, args ...any) {
	args = append([]interface{}{pl.prefix}, args...)
	InfoDepth(depth, args...)
}

func (pl *PrefixedLogger) Warning(args ...any) {
	args = append([]interface{}{pl.prefix}, args...)
	Warning(args...)
}

func (pl *PrefixedLogger) Warningf(format string, args ...any) {
	args = append([]interface{}{pl.prefix}, args...)
	Warningf("%s"+format, args...)
}

func (pl *PrefixedLogger) WarningDepth(depth int, args ...any) {
	args = append([]interface{}{pl.prefix}, args...)
	WarningDepth(depth, args...)
}

func (pl *PrefixedLogger) Error(args ...any) {
	args = append([]interface{}{pl.prefix}, args...)
	Error(args...)
}

func (pl *PrefixedLogger) Errorf(format string, args ...any) {
	args = append([]interface{}{pl.prefix}, args...)
	Errorf("%s"+format, args...)
}

func (pl *PrefixedLogger) ErrorDepth(depth int, args ...any) {
	args = append([]interface{}{pl.prefix}, args...)
	ErrorDepth(depth, args...)
}

func (pl *PrefixedLogger) Exit(args ...any) {
	args = append([]interface{}{pl.prefix}, args...)
	Exit(args...)
}

func (pl *PrefixedLogger) Exitf(format string, args ...any) {
	args = append([]interface{}{pl.prefix}, args...)
	Exitf("%s"+format, args...)
}

func (pl *PrefixedLogger) ExitDepth(depth int, args ...any) {
	args = append([]interface{}{pl.prefix}, args...)
	ExitDepth(depth, args...)
}

func (pl *PrefixedLogger) Fatal(args ...any) {
	args = append([]interface{}{pl.prefix}, args...)
	Fatal(args...)
}

func (pl *PrefixedLogger) Fatalf(format string, args ...any) {
	args = append([]interface{}{pl.prefix}, args...)
	Fatalf("%s"+format, args...)
}

func (pl *PrefixedLogger) FatalDepth(depth int, args ...any) {
	args = append([]interface{}{pl.prefix}, args...)
	FatalDepth(depth, args...)
}
