// Package osutils provides enrichments to types in the os package
package osutils

import (
	"fmt"
	"os"
	"syscall"

	humanize "github.com/dustin/go-humanize"
)

// HumanizedFileInfo - builds on os.FileInfo to provide a humanized description
// Example of use:
//	_info, err := os.Stat("/tmp/f.txt")
//	if err != nil {
//		log.Fatal(err)
//	}
//	info := osutils.HumanizedFileInfo{FileInfo: _info}
//	fmt.Println(info)
type HumanizedFileInfo struct {
	os.FileInfo
}

type humanizedStatT struct {
	*syscall.Stat_t
}

func (s humanizedStatT) String() string {
	return fmt.Sprintf("Size: %d, Blocks: %d, Flags: %d", s.Size, s.Blocks, s.Flags)
}

func (f HumanizedFileInfo) String() string {
	fileOrDir := "File"
	if f.IsDir() {
		fileOrDir = "Dir"
	}
	humanizedBytes := humanize.Bytes(uint64(f.Size()))
	humanizedModifiedTime := humanize.Time(f.ModTime())
	humanizedStat := ""
	if f.Sys() != nil {
		_stat, ok := f.Sys().(*syscall.Stat_t)
		if ok {
			stat := humanizedStatT{_stat}
			humanizedStat = fmt.Sprintf("[%s]", stat)
		}
	}
	return fmt.Sprintf("%s (%s): %s. Modified: %s. %s", f.Name(), fileOrDir, humanizedBytes, humanizedModifiedTime, humanizedStat)
}
