package main

import (
	"encoding/binary"
	"fmt"
)

const (
	headerEntrySize      = 4
	timestampSizeInBytes = 8                                                       // Number of bytes used for timestamp
	hashSizeInBytes      = 8                                                       // Number of bytes used for hash
	keySizeInBytes       = 2                                                       // Number of bytes used for size of entry key
	headersSizeInBytes   = timestampSizeInBytes + hashSizeInBytes + keySizeInBytes // Number of bytes used for all headers
)

func main() {
	index := 1
	str := []uint8{0, 32, 0, 0, 0, 90, 171, 41, 87, 0, 0, 0, 0, 103, 151, 171, 167, 104, 24, 29, 206, 13, 0, 109, 121, 45, 117, 110, 105, 113, 117, 101, 45, 107, 101, 121, 97, 25, 0, 0, 0, 90, 171, 41, 87, 0, 0, 0, 0, 151, 106, 139, 152, 155, 205, 212, 76, 6, 0, 105, 106, 105, 98, 117, 49, 99, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	blockSize := int(binary.LittleEndian.Uint32(str[index : index+headerEntrySize]))
	fmt.Println(blockSize)
	data := str[index+headerEntrySize : index+headerEntrySize+blockSize]

	fmt.Println(string(readEntry(data)))
	fmt.Println(readKeyFromEntry(data))
	fmt.Println(readHashFromEntry(data))
	fmt.Println(readTimestampFromEntry(data))
}

func readEntry(data []byte) []byte {
	length := binary.LittleEndian.Uint16(data[timestampSizeInBytes+hashSizeInBytes:])
	return data[headersSizeInBytes+length:]
}

func readTimestampFromEntry(data []byte) uint64 {
	return binary.LittleEndian.Uint64(data)
}

func readKeyFromEntry(data []byte) string {
	length := binary.LittleEndian.Uint16(data[timestampSizeInBytes+hashSizeInBytes:])
	return string(data[headersSizeInBytes : headersSizeInBytes+length])
}

func readHashFromEntry(data []byte) uint64 {
	return binary.LittleEndian.Uint64(data[timestampSizeInBytes:])
}
