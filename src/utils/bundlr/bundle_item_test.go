package bundlr

import (
	"bytes"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/tool"

	"testing"
)

func TestBundleItemTestSuite(t *testing.T) {
	suite.Run(t, new(BundleItemTestSuite))
}

type BundleItemTestSuite struct {
	suite.Suite
	signer *Signer
}

func (s *BundleItemTestSuite) SetupSuite() {
	var err error
	s.signer, err = NewSigner(EMPTY_ARWEAVE_WALLET)
	require.Nil(s.T(), err)
}

func (s *BundleItemTestSuite) TestSerialization() {
	item := BundleItem{
		SignatureType: SignatureTypeArweave,
		Target:        arweave.Base64String(tool.RandomString(32)),
		// Anchor: arweave.Base64String(tool.RandomString(32)),
		Tags: Tags{Tag{Name: "1", Value: "2"}, Tag{Name: "3", Value: "4"}},
		Data: arweave.Base64String(tool.RandomString(100)),
	}

	buf, err := item.Reader(s.signer)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), buf)

	reader := bytes.NewReader(buf.Bytes())
	parsed := BundleItem{}

	err = parsed.UnmarshalFromReader(reader)
	require.Nil(s.T(), err)
	require.Equal(s.T(), item.SignatureType, parsed.SignatureType)
	require.Equal(s.T(), item.Data, parsed.Data)
	require.Equal(s.T(), item.Tags, parsed.Tags)
}
