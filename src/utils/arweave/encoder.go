package arweave

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
)

type Encoder struct {
	*bytes.Buffer
}

func NewEncoder() Encoder {
	return Encoder{Buffer: &bytes.Buffer{}}
}

func (self Encoder) Trim(val []byte) []byte {
	for i := 0; i < len(val); i++ {
		if val[i] != 0 {
			return val[i:]
		}
	}
	return []byte{}
}

// Erlang's ar_serialize:encode_int
func (self Encoder) WriteUint64(val uint64, sizeBytes int) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, val)
	buf = self.Trim(buf)
	self.WriteBuffer(buf, sizeBytes)
}

// Erlang's ar_serialize:encode_bin
func (self Encoder) WriteBuffer(val []byte, sizeBytes int) {
	buf := make([]byte, 8)
	size := uint64(len(val))
	binary.BigEndian.PutUint64(buf, size)
	self.Buffer.Write(buf[8-uint64(sizeBytes):])
	self.Buffer.Write(val)
}

func (self Encoder) encodeBin(val []byte, sizeBytes int) []byte {
	buf := make([]byte, 8)
	size := uint64(len(val))
	binary.BigEndian.PutUint64(buf, size)
	return append(buf[8-sizeBytes:], val...)
}

// -------------------------------------------------------------------------------------

func (self Encoder) Write(val any, sizeBytes int) {
	// if val == nil || reflect.ValueOf(val).IsZero() {
	// 	for i := 0; i < sizeBytes; i++ {
	// 		self.WriteByte(0)
	// 	}
	// 	return
	// }

	switch x := val.(type) {
	case []byte:
		self.WriteBuffer(x, sizeBytes)
	case RewardAddr:
		self.WriteBuffer(x.Bytes(), sizeBytes)
	case Base64String:
		self.WriteBuffer([]byte(x), sizeBytes)
	case BigInt:
		self.WriteBuffer(x.Bytes(), sizeBytes)
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
	case uint64:
		self.RawWriteUint64(x)
	case []Base64String:
		self.RawWriteBase64StringSlice(x)
	default:
		panic("unsupported encoder raw type")
	}
}

func (self Encoder) RawWriteSize(val any, sizeBytes int) {
	out := make([]byte, sizeBytes)
	switch x := val.(type) {
	case uint16:
		binary.BigEndian.PutUint16(out, x)
	case uint64:
		binary.BigEndian.PutUint64(out[sizeBytes-8:], x)

	default:
		panic("unsupported encoder raw type")
	}

	self.Buffer.Write(out)
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

func (self Encoder) WriteSlice(val []any, lenBytes, elemSizeBytes int) {
	self.WriteUint(uint(len(val)), lenBytes)
	for _, v := range val {
		self.Write(v, elemSizeBytes)
	}
}

func (self Encoder) WriteSliceAny(bins []any, lenBytes, elemSizeBytes int) {
	var encodeBinList func(bins []any, encoded []byte, n int, lenBytes, elemSizeBytes int) []byte
	encodeBinList = func(bins []any, encoded []byte, n int, lenBytes, elemSizeBytes int) []byte {
		if len(bins) > 0 {
			// Still bins to encode
			b, ok := bins[0].([]byte)
			if !ok {
				panic("unsupported encoder type")
			}

			elem := self.encodeBin(b, elemSizeBytes)
			return encodeBinList(bins[1:], append(elem, encoded...), n+1, lenBytes, elemSizeBytes)
		}

		// No more bins to encode
		buf := make([]byte, 8)
		size := uint64(n)
		binary.BigEndian.PutUint64(buf, size)
		return append(buf[8-uint64(lenBytes):], encoded...)
	}
	buf := encodeBinList(bins, []byte{}, 0, lenBytes, elemSizeBytes)
	self.Buffer.Write(buf)
}

func (self Encoder) WriteSliceByte(bins [][]byte, lenBytes, elemSizeBytes int) {
	var encodeBinList func(bins [][]byte, encoded []byte, n int, lenBytes, elemSizeBytes int) []byte
	encodeBinList = func(bins [][]byte, encoded []byte, n int, lenBytes, elemSizeBytes int) []byte {
		if len(bins) > 0 {
			// Still bins to encode
			elem := self.encodeBin(bins[0], elemSizeBytes)
			return encodeBinList(bins[1:], append(elem, encoded...), n+1, lenBytes, elemSizeBytes)
		}

		// No more bins to encode
		buf := make([]byte, 8)
		size := uint64(n)
		binary.BigEndian.PutUint64(buf, size)
		return append(buf[8-uint64(lenBytes):], encoded...)
	}
	buf := encodeBinList(bins, []byte{}, 0, lenBytes, elemSizeBytes)
	self.Buffer.Write(buf)
}

func (self Encoder) WriteBase64StringSlice(bins []Base64String, lenBytes, elemSizeBytes int) {
	var encodeBinList func(bins []Base64String, encoded []byte, n int, lenBytes, elemSizeBytes int) []byte
	encodeBinList = func(bins []Base64String, encoded []byte, n int, lenBytes, elemSizeBytes int) []byte {
		if len(bins) > 0 {
			// Still bins to encode
			elem := self.encodeBin(bins[0], elemSizeBytes)
			return encodeBinList(bins[1:], append(elem, encoded...), n+1, lenBytes, elemSizeBytes)
		}

		// No more bins to encode
		buf := make([]byte, 8)
		size := uint64(n)
		binary.BigEndian.PutUint64(buf, size)
		return append(buf[8-uint64(lenBytes):], encoded...)
	}
	buf := encodeBinList(bins, []byte{}, 0, lenBytes, elemSizeBytes)
	self.Buffer.Write(buf)
}

// encode_bin_list([], Encoded, N, LenBits, _ElemSizeBits) ->
// 		<< N:LenBits, (iolist_to_binary(Encoded))/binary >>;
// encode_bin_list([Bin | Bins], Encoded, N, LenBits, ElemSizeBits) ->
// 		Elem = encode_bin(Bin, ElemSizeBits),
// 		encode_bin_list(Bins, [Elem | Encoded], N + 1, LenBits, ElemSizeBits).

// Erlang's binary:encode_unsigned
func (self Encoder) RawWriteUint64(val uint64) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, val)
	buf = self.Trim(buf)
	self.Buffer.Write(buf)
}

func (self Encoder) RawWriteUint16(val uint16) {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, val)
	buf = self.Trim(buf)
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

func (self Encoder) Base64() string {
	return base64.RawURLEncoding.EncodeToString(self.Bytes())
}
