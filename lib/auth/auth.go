package auth

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/binary"
	"golang.org/x/crypto/md4"
	"unsafe"
)

func MakeKeepAlive1Packet1(cnt byte) []byte {
	return []byte{0x07, cnt, 0x08, 0x00, 0x01, 0x00, 0x00, 0x00}
}

func MakeKeepAlive1Packet2(seed, hostIP []byte, keepAlive1Flag byte, cnt byte, crypt, first bool) []byte {
	data := []byte{0x07}                                    // code
	data = append(data, cnt)                                // id
	data = append(data, 0x60, 0x00)                         // length
	data = append(data, 0x03)                               // type
	data = append(data, 0x00)                               // uid length
	data = append(data, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00) // mac
	data = append(data, hostIP...)                          // sip
	if first {
		data = append(data, 0x00, 0x62, 0x00)
		data = append(data, keepAlive1Flag)
	} else {
		data = append(data, 0x00, 0x63, 0x00)
		data = append(data, keepAlive1Flag)
	}
	data = append(data, seed...)
	if crypt {
		data = append(data, generateChecksum1(int(seed[0]&0x03), seed)...)
	} else {
		data = append(data, generateChecksum1(0, seed)...)
	}
	for i := 0; i < 16*4; i++ {
		data = append(data, 0x00)
	}
	return data
}

func generateChecksum1(mode int, seed []byte) []byte {
	switch mode {
	case 0:
		n1 := 20000711
		buf1 := make([]byte, 4)
		binary.BigEndian.PutUint32(buf1, uint32(n1))
		n2 := 126
		buf2 := make([]byte, 4)
		binary.BigEndian.PutUint32(buf2, uint32(n2))
		return append(buf1, buf2...)
	case 1:
		h := md5.New()
		h.Write(seed)
		m := h.Sum(nil)
		r := make([]byte, 8)
		r[0] = m[2]
		r[1] = m[3]
		r[2] = m[8]
		r[3] = m[9]
		r[4] = m[5]
		r[5] = m[6]
		r[6] = m[13]
		r[7] = m[14]
		return r
	case 2:
		h := md4.New()
		h.Write(seed)
		m := h.Sum(nil)
		r := make([]byte, 8)
		r[0] = m[1]
		r[1] = m[2]
		r[2] = m[8]
		r[3] = m[9]
		r[4] = m[4]
		r[5] = m[5]
		r[6] = m[11]
		r[7] = m[12]
		return r
	case 3:
		h := sha1.New()
		h.Write(seed)
		m := h.Sum(nil)
		r := make([]byte, 8)
		r[0] = m[2]
		r[1] = m[3]
		r[2] = m[9]
		r[3] = m[10]
		r[4] = m[5]
		r[5] = m[6]
		r[6] = m[15]
		r[7] = m[16]
		return r
	default:
		return nil
	}
}

func MakeKeepAlive2Packet1(cnt byte, keepAlive2Flag []byte, random []byte, keepAlive2Key []byte) []byte {
	data := []byte{0x07, cnt, 0x28, 0x00, 0x0b, 0x01}
	data = append(data, keepAlive2Flag...)
	data = append(data, random...)
	for i := 0; i < 6; i++ {
		data = append(data, 0x00)
	}
	data = append(data, keepAlive2Key...)
	for i := 0; i < 20; i++ {
		data = append(data, 0x00)
	}
	return data
}

func MakeKeepAlive2Packet2(cnt byte, keepAlive2Flag []byte, random []byte, keepAlive2Key []byte, hostIP []byte) []byte {
	data := []byte{0x07, cnt, 0x28, 0x00, 0x0b, 0x03}
	data = append(data, keepAlive2Flag...)
	data = append(data, random...)
	for i := 0; i < 6; i++ {
		data = append(data, 0x00)
	}
	data = append(data, keepAlive2Key...)
	for i := 0; i < 4; i++ {
		data = append(data, 0x00)
	}
	checksumP := len(data)
	for i := 0; i < 4; i++ {
		data = append(data, 0x00)
	}
	data = append(data, hostIP...)
	for i := 0; i < 8; i++ {
		data = append(data, 0x00)
	}
	checkSum := generateChecksum2(data)
	af := append(data[:checksumP], checkSum...)
	data = append(af, data[checksumP+4:]...)
	return data
}

func generateChecksum2(data []byte) []byte {
	p := (*[]int16)(unsafe.Pointer(&data))
	checkSumTemp := int32(0)
	for i := 0; i < len(*p)/2; i++ {
		checkSumTemp ^= int32((*p)[i])
	}
	checkSumTemp &= 0xFFFF
	checkSumTemp *= 0x2C7
	checkSum := make([]byte, 4)
	for i := 0; i < 4; i++ {
		checkSum[i] = uint8(checkSumTemp >> (8 * uint(i)))
	}
	return checkSum
}
