package arweave

import (
	"bytes"
	"encoding/binary"
	"reflect"
)

type Encoder struct {
	*bytes.Buffer
}

func (self Encoder) Write(val any, sizeBytes int) {
	if val == nil || reflect.ValueOf(val).IsZero() {
		for i := 0; i < sizeBytes; i++ {
			self.WriteByte(0)
		}
		return
	}

	switch x := val.(type) {
	case []byte:
		self.WriteBuffer(x, sizeBytes)
	case RewardAddr:
		self.WriteBuffer(x.Bytes(), sizeBytes)
	case Base64String:
		self.WriteBuffer([]byte(x), sizeBytes)
	case BigInt:
		self.WriteBuffer(x.Bytes(), sizeBytes)
	case int:
		self.WriteUint(uint(x), sizeBytes)
	case int64:
		self.WriteUint64(uint64(x), sizeBytes)
	case uint64:
		self.WriteUint64(x, sizeBytes)
	default:
		panic("unsupported encoder type")
	}
}

func (self Encoder) RawWrite(val any) {
	switch x := val.(type) {
	case byte:
		self.Buffer.WriteByte(x)
	case []byte:
		self.Buffer.Write(x)
	case Base64String:
		self.Buffer.Write(x.Bytes())
	case BigInt:
		self.Buffer.Write(x.Bytes())
	case uint16:
		self.RawWriteUint16(x)
	case uint64:
		self.RawWriteUint64(x)
	case []Base64String:
		self.RawWriteBase64StringSlice(x)
	default:
		panic("unsupported encoder raw type")
	}
}

func (self Encoder) WriteBuffer(val []byte, sizeBytes int) {
	size := len(val)
	for i := 0; i < sizeBytes; i++ {
		self.WriteByte(byte(size >> uint((sizeBytes-i-1)*8)))
	}
	self.Buffer.Write(val)
}

func (self Encoder) WriteUint(val uint, sizeBytes int) {
	if sizeBytes == 1 {
		self.WriteUint8(uint8(val), sizeBytes)
		return
	} else if sizeBytes == 2 {
		self.WriteUint16(uint16(val), sizeBytes)
		return
	}

	panic("Unsupported size")
}

func (self Encoder) WriteUint8(val uint8, sizeBytes int) {
	self.WriteBuffer([]byte{byte(val)}, sizeBytes)
}

func (self Encoder) WriteUint16(val uint16, sizeBytes int) {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, val)
	self.WriteBuffer(buf, sizeBytes)
}

func (self Encoder) WriteUint64(val uint64, sizeBytes int) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, val)
	self.WriteBuffer(buf, sizeBytes)
}

func (self Encoder) WriteSlice(val []any, lenBytes, elemSizeBytes int) {
	self.WriteUint(uint(len(val)), lenBytes)
	for _, v := range val {
		self.Write(v, elemSizeBytes)
	}
}

func (self Encoder) WriteBase64StringSlice(val []Base64String, lenBytes, elemSizeBytes int) {
	self.WriteUint(uint(len(val)), lenBytes)
	for _, v := range val {
		self.Write(v, elemSizeBytes)
	}
}

func (self Encoder) RawWriteUint64(val uint64) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, val)
	self.Buffer.Write(buf)
}

func (self Encoder) RawWriteUint16(val uint16) {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, val)
	self.Buffer.Write(buf)
}

func (self Encoder) RawWriteBigInt(val BigInt, sizeBytes int) {
	if !val.IsUint64() {
		panic("bad bigint value")
	}
	value := val.Uint64()
	for i := 0; i < sizeBytes; i++ {
		self.WriteByte(byte(value >> uint((sizeBytes-i-1)*8)))
	}
}

func (self Encoder) RawWriteBase64StringSlice(val []Base64String) {
	for _, v := range val {
		self.Buffer.Write(v.Bytes())
	}
}
