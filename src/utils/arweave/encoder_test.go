package arweave

import (
	"encoding/base64"
	"math/big"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"testing"
)

func TestEncoderTestSuite(t *testing.T) {
	suite.Run(t, new(EncoderTestSuite))
}

type EncoderTestSuite struct {
	suite.Suite
}

func (s *EncoderTestSuite) TestTrim() {
	env := NewEncoder()
	a := []byte{0, 0, 0, 3}
	require.Equal(s.T(), []byte{3}, env.Trim(a))
}

func (s *EncoderTestSuite) TestUint64() {
	env := NewEncoder()
	env.RawWrite(uint64(1234))
	require.Equal(s.T(), "BNI", env.Base64())
}

func (s *EncoderTestSuite) TestEncodeInt() {
	env := NewEncoder()
	env.WriteUint64(1234, 2)
	require.Equal(s.T(), "AAIE0g", env.Base64())
}

func (s *EncoderTestSuite) TestEncodeBinEmpty() {
	env := NewEncoder()
	env.WriteBuffer([]byte{}, 2)
	require.Equal(s.T(), []byte{0, 0}, env.Bytes())
}

func (s *EncoderTestSuite) TestEncodeBin() {
	env := NewEncoder()
	env.WriteBuffer([]byte{9, 7}, 2)
	require.Equal(s.T(), []byte{0, 2, 9, 7}, env.Bytes())
}

func (s *EncoderTestSuite) TestEncodeListFlat() {
	env := NewEncoder()
	l := make([][]byte, 0)
	l = append(l, []byte{9, 7})
	env.WriteSliceByte(l, 1, 2)
	require.Equal(s.T(), []byte{1, 0, 2, 9, 7}, env.Bytes())
}

func (s *EncoderTestSuite) TestEncodeList() {
	env := NewEncoder()
	l := make([][]byte, 0)
	l = append(l, []byte{9, 7})
	l = append(l, []byte{6, 5})
	env.WriteSliceByte(l, 1, 2)
	// fmt.Printf("%X\n", env.Bytes())
	require.Equal(s.T(), []byte{2, 0, 2, 6, 5, 0, 2, 9, 7}, env.Bytes())
}
func (s *EncoderTestSuite) TestEncodeListEmpty() {
	env := NewEncoder()
	a, _ := base64.RawURLEncoding.DecodeString("ZXktLIUuYEaTrN6l2grYO49JegaOp0sbz_itn6hVKLao47L_kdFkMlJ3VWSI8fLB")
	env.WriteBuffer(a, 1)
	// fmt.Printf("%v\n", env.Bytes())
	require.Equal(s.T(), []byte{48, 101, 121, 45, 44, 133, 46, 96, 70, 147, 172, 222, 165, 218, 10, 216, 59, 143, 73, 122, 6, 142, 167, 75, 27, 207, 248, 173, 159, 168, 85, 40, 182, 168, 227, 178, 255, 145, 209, 100, 50, 82, 119, 85, 100, 136, 241, 242, 193}, env.Bytes())

	// 65 79 2D 2C 85 2E 60 46 93 AC DE A5 DA 0A D8 3B 8F 49 7A 06 8E A7 4B 1B CF F8 AD 9F A8 55 28 B6 A8 E3 B2 FF 91 D1 64 32 52 77 55 64 88 F1 F2 C1
	// 3065792D2C852E604693ACDEA5DA0AD83B8F497A068EA74B1BCFF8AD9FA85528B6A8E3B2FF91D164325277556488F1F2C1
	//   65792D2C852E604693ACDEA5DA0AD83B8F497A068EA74B1BCFF8AD9FA85528B6A8E3B2FF91D164325277556488F1F2C1
}

func (s *EncoderTestSuite) TestIolistToBinary() {
	env := NewEncoder()
	l := make([]Base64String, 0)
	l = append(l, Base64String{1, 2})
	l = append(l, Base64String{3, 4})
	env.RawWriteBase64StringSlice(l)
	require.Equal(s.T(), []byte{1, 2, 3, 4}, env.Bytes())
}

func (s *EncoderTestSuite) TestRawWriteSize() {
	env := NewEncoder()
	l := uint64(1234)
	env.RawWriteSize(l, 8)
	// fmt.Printf("%X\n", env.Bytes())
	// 00000000000004D2
	// 00000000000004D2
	require.Equal(s.T(), []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x4, 0xd2}, env.Bytes())
}

func (s *EncoderTestSuite) TestRawWriteSizeBigger() {
	env := NewEncoder()
	l := uint64(1234)
	env.RawWriteSize(l, 32)
	// fmt.Printf("%v\n", env.Bytes())
	// 00000000000000000000000000000000000000000000000000000000000004D2
	// 00000000000000000000000000000000000000000000000000000000000004D2
	require.Equal(s.T(), []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x4, 0xd2}, env.Bytes())
}

func (s *EncoderTestSuite) TestOutput() {
	env := NewEncoder()
	a, _ := base64.RawURLEncoding.DecodeString("7NRtHpg2JEPn37XcRUNa2iRqBAIE5GNIwBbov_OX_9g")
	buf := Base64String(a)
	env.RawWrite(buf.Head(32))
	require.Equal(s.T(), a, env.Bytes())

	// ECD46D1E98362443E7DFB5DC45435ADA246A040204E46348C016E8BFF397FFD8
	// ECD46D1E98362443E7DFB5DC45435ADA246A040204E46348C016E8BFF397FFD8
}
func (s *EncoderTestSuite) TestRawWriteBigInt() {
	env := NewEncoder()
	v := BigInt{*big.NewInt(9)}
	env.RawWriteBigInt(v, 3)
	// fmt.Printf("%X\n", env.Bytes())

	require.Equal(s.T(), []byte{0, 0, 9}, env.Bytes())

	// ECD46D1E98362443E7DFB5DC45435ADA246A040204E46348C016E8BFF397FFD8
	// ECD46D1E98362443E7DFB5DC45435ADA246A040204E46348C016E8BFF397FFD8
}
