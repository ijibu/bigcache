package bigcache

import (
	"encoding/binary"
)

const (
	timestampSizeInBytes = 8                                                       // Number of bytes used for timestamp 用8个byte存储timestamp
	hashSizeInBytes      = 8                                                       // Number of bytes used for hash 用8个byte存储hashKey
	keySizeInBytes       = 2                                                       // Number of bytes used for size of entry key 用两个byte来储存key的长度，所以key的长度最多为65535。
	headersSizeInBytes   = timestampSizeInBytes + hashSizeInBytes + keySizeInBytes // Number of bytes used for all headers 所有头部信息的长度
)

//包裹实体
func wrapEntry(timestamp uint64, hash uint64, key string, entry []byte, buffer *[]byte) []byte {
	keyLength := len(key)
	blobLength := len(entry) + headersSizeInBytes + keyLength

	if blobLength > len(*buffer) {
		//重新分配内存
		*buffer = make([]byte, blobLength)
	}
	blob := *buffer

	//uint64正好8个字节
	binary.LittleEndian.PutUint64(blob, timestamp)
	binary.LittleEndian.PutUint64(blob[timestampSizeInBytes:], hash)
	//uint16正好2个字节
	binary.LittleEndian.PutUint16(blob[timestampSizeInBytes+hashSizeInBytes:], uint16(keyLength))
	copy(blob[headersSizeInBytes:], []byte(key))
	copy(blob[headersSizeInBytes+keyLength:], entry)

	return blob[:blobLength]
}

//读取实体
func readEntry(data []byte) []byte {
	length := binary.LittleEndian.Uint16(data[timestampSizeInBytes+hashSizeInBytes:])
	return data[headersSizeInBytes+length:]
}

//从实体字节数组中读取Timestamp
func readTimestampFromEntry(data []byte) uint64 {
	return binary.LittleEndian.Uint64(data)
}

//从实体字节数组中读取key
func readKeyFromEntry(data []byte) string {
	length := binary.LittleEndian.Uint16(data[timestampSizeInBytes+hashSizeInBytes:])
	return string(data[headersSizeInBytes : headersSizeInBytes+length])
}

//从实体字节数组中读取Hash
func readHashFromEntry(data []byte) uint64 {
	return binary.LittleEndian.Uint64(data[timestampSizeInBytes:])
}

//重置实体字节数组中的key
func resetKeyFromEntry(data []byte) {
	binary.LittleEndian.PutUint64(data[timestampSizeInBytes:], 0)
}
