package block

import (
	//"contentos-go/common/block"
	"encoding/binary"
	"errors"
	"os"
)

const indexSize = 8
const blockLenSize = 4

/*BLog is an external append only log of the blocks. Blocks should only be written
 * to the log after they irreverisble as the log is append only. There is a secondary
 * index file of only block positions that enables O(1) random access lookup by block number.
 *
 * A block data in the BLog is formated as len+payload, len is a uint32
 *
 * +---------+----------------+---------+----------------+-----+------------+-------------------+
 * | Block 1 | Pos of Block 1 | Block 2 | Pos of Block 2 | ... | Head Block | Pos of Head Block |
 * +---------+----------------+---------+----------------+-----+------------+-------------------+
 *
 * +----------------+----------------+-----+-------------------+
 * | Pos of Block 1 | Pos of Block 2 | ... | Pos of Head Block |
 * +----------------+----------------+-----+-------------------+
 *
 *
 * Blocks can be accessed at random via block number through the index file. Seek to 8 * (block_num - 1)
 * to find the position of the block in the main file.
 *
 * The main file is the only file that needs to persist. The index file can be reconstructed during a
 * linear scan of the main file.
 */
type BLog struct {
	logFile   *os.File
	indexFile *os.File
}

// Open opens the block log & index file
func (bl *BLog) Open(dir string) (err error) {
	bl.logFile, err = os.OpenFile(dir+"/block.bin", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return
	}
	bl.indexFile, err = os.OpenFile(dir+"/block.index", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return
	}

	logInfo, err := bl.logFile.Stat()
	if err != nil {
		return err
	}
	indexInfo, err := bl.indexFile.Stat()
	if err != nil {
		return err
	}

	if logInfo.Size() != 0 {
		if indexInfo.Size() != 0 {
			indexByte := make([]byte, indexSize)
			lastIdxFromLogFile, err := bl.readLastIndex(indexByte, true)
			if err != nil {
				return err
			}
			lastIdxFromIndexFile, err := bl.readLastIndex(indexByte, false)
			if err != nil {
				return err
			}
			if lastIdxFromIndexFile != lastIdxFromLogFile {
				bl.reindex()
			}
		} else {
			bl.reindex()
		}
	} else if indexInfo.Size() != 0 {
		bl.indexFile.Truncate(0)
	}

	return
}

func (bl *BLog) reindex() (err error) {
	if bl.indexFile != nil {
		// TODO: error log
		bl.indexFile.Truncate(0)
	} else {
		return nil
	}

	var offset, end int64
	indexByte := make([]byte, indexSize)

	end, err = bl.readLastIndex(indexByte, true)
	if err != nil {
		return err
	}

	for offset < end {
		var length int
		// read payload len
		payloadLenByte := make([]byte, blockLenSize)
		var payloadLen uint32
		length, err = bl.logFile.Read(payloadLenByte)
		if err != nil {
			return err
		}
		if length != blockLenSize {
			return errors.New("wrong blockLen size")
		}
		payloadLen = binary.LittleEndian.Uint32(payloadLenByte)

		// read payload
		payloadByte := make([]byte, payloadLen)
		length, err = bl.logFile.Read(payloadByte)
		if err != nil {
			return err
		}
		if uint32(length) != payloadLen {
			return errors.New("wrong payloadLen size")
		}

		// read index
		length, err = bl.logFile.Read(indexByte)
		if err != nil {
			return err
		}
		if uint32(length) != payloadLen {
			return errors.New("wrong index size")
		}

		// append index to indexFile
		length, err = bl.indexFile.Write(indexByte)
		if err != nil {
			return err
		}
		if length != indexSize {
			return errors.New("wrong index size")
		}

		offset = int64(binary.LittleEndian.Uint32(indexByte))
	}
	return nil
}

func (bl *BLog) readLastIndex(indexByte []byte, isLogFile bool) (int64, error) {
	var file *os.File
	if isLogFile {
		file = bl.logFile
	} else {
		file = bl.indexFile
	}
	file.Seek(-indexSize, 2)
	length, err := file.Read(indexByte)
	if err != nil {
		return 0, err
	}
	if length != indexSize {
		return 0, errors.New("wrong last index size")
	}
	return int64(binary.LittleEndian.Uint64(indexByte)), nil
}
