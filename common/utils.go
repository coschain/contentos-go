package common

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/coschain/contentos-go/common/constants"
	"io"
	"os"
	"runtime"
	"unsafe"
)

func Int2Bytes(n uint32) []byte {
	var b []byte
	var i int
	for i = 0; i < 4; i++ {
		b = append(b, 0)
	}
	i = 4
	for n > 0 && i > 0 {
		i--
		b[i] = byte(n & 0xff)
		n /= 256
	}
	return b
}

// Fatalf formats a message to standard error and exits the program.
// The message is also printed to standard output if standard error
// is redirected to a different file.
func Fatalf(format string, args ...interface{}) {
	w := io.MultiWriter(os.Stdout, os.Stderr)
	if runtime.GOOS == "windows" {
		// The SameFile check below doesn't work on Windows.
		// stdout is unlikely to get redirected though, so just print there.
		w = os.Stdout
	} else {
		outf, _ := os.Stdout.Stat()
		errf, _ := os.Stderr.Stat()
		if outf != nil && errf != nil && os.SameFile(outf, errf) {
			w = os.Stderr
		}
	}
	_, _ = fmt.Fprintf(w, "Fatal: "+format+"\n", args...)
	os.Exit(1)
}

func GetBucket(timestamp uint32) uint32 {
	return timestamp / uint32(constants.BlockInterval)
}

const Is32bitPlatform = ^uint(0)>>32 == 0

var (
	endianTesting = int(1)
	isLittleEndianPlatform = *(*byte)(unsafe.Pointer(&endianTesting)) != 0
)

func IsLittleEndianPlatform() bool {
	return isLittleEndianPlatform
}

func HostByteOrder() binary.ByteOrder {
	if IsLittleEndianPlatform() {
		return binary.LittleEndian
	} else {
		return binary.BigEndian
	}
}

func JsonNumber(jn json.Number) (r int64) {
	r, _ = jn.Int64()
	return
}
