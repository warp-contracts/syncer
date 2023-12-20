package turbo

import (
	"context"

	"github.com/warp-contracts/syncer/src/utils/bundlr"
	"github.com/warp-contracts/syncer/src/utils/config"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"testing"
)

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

type ClientTestSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc
	signer *bundlr.ArweaveSigner
	client *Client
}

const EMPTY_ARWEAVE_WALLET = `{
    "d": "IVv3IzUPbj2yJP9qqJcH3cVI86jWdhZCpNoomLeJaH0rpKnujzlDSADC2yuFNBnS_sIthk1-w83_bkTwwOOCAn_9LZbkKYEd2onZ7iWAh--tMB5ijNHv0acn64TZjS-5aH6WgfsxwCjrXj57ejnh7GaterucVpTX_RlGtpp5IWY5ISM-5JLBm2wLLnXjhsJD51a03eClxy0MAclG6suOkm2pRF7yl1sJjQ23kZ7xExpO-Lb_j8o1JEGao5xI1TPWdJyovuhPrWK14l3JXU9URz6IKFH9xuvbWjqWhyVQVjUBBWg5B5DbzQhI_6tPVHb8eUBP9L9BNkRyr5cWU1SCYynzEa9_1cXjLuYNtTUB9358bkveYiZRlvSjCYoNd6lSFtESbyMfvmU2FF7gnduVqzdTPuisfHHNYQKCall-emCt9Oiy26OJ2uMX-dfqutcZd65OlJN5KG65h6D8cp7xjDlwHx4VeK2qI-dyzOS6ufZlG0nrNEfzRDekmRsFCgZxJUjc0JjCMde5LRKZhsmltntizeaURw69dnNTrtrLFQLlo6X3wEHzyjFNqaqJDQmB6UnpdOjZp6FeotV02FpeqhJZ8pA1kYywO9LFB-iciy7h-bufHoK5Owti-CwOMADdwzYPPaKrbhc7ZhAuogQTMfFSHJtL5_le_Y-k8FTtu4E",
    "dp": "phZwSYPUvAO-231R-_IuLMHB294qzoiHeg1GUBEAvf4PqA95dgQAXUQUTEVUBuOvJ89g4Zubz3QcRabzEeySGDHLhF0x5BdUCmZugiQJ_MphBPTa82PPDWWohPTdztt8L-2mXWAJRHQqesT4zix7cKYao9wbWvG-9i0sDzk9hfFT9HNM8yr5-Sp089so-5jro-48ZWa97nhsOKDvNamHX9BdOX-TSl97txlSf5IjgXeGUImgIcIgZAdnp7cWjo2rYodyaeJ_yh_dGEnVL1XauVJ5gochLIKcIIZWaO0ENqvPJdly_TT7FUHG-uLUicSGRJuloBooZzLUzMuasSZwoQ",
    "dq": "X_oppBgiMcI6fyuvlTI9YaveiJmLWI_B2T1IsdU0xPS1PvPdjLq5ArK7NpqlkWsaF3Y4eR96uPniNPGrnvl7Z4A383G7zOXtlFzuYZxvXMGs9G46VNVXxT0vvO9Htm4Zp8W11eW9MneKXdeJ-uMUcTw3vlCgXG8x9C2CcTqRN_J3PNiWmkHT2FE5Tbqwj36MPPOOInI-22k3UG2OX2qOrQoFD6SPgRoRLJmRLDl_ktJ1rQus187FfNgmB77-qeg_p772jwLxnzIvay4WmehJdI1wdp_JlKmQkEqknAq_ab0ltLcofqCR4-_2MkFMLksqVDilUtQkH3Od0QYIlbM9kw",
    "e": "AQAB",
    "ext": true,
    "kty": "RSA",
    "n": "xEDoW3dIO93QcmK3G1bgNrguKoI1eSsgtBd5IERwJOtpqM2cBDlqkMbMhcy3dzL-0YPSPAB78HudvhnmNlTRWas9zqPX7nj0CtcDlbntAWIyjUXUUbqdRHUkvOpUzEcdU-x9ZLFPOJfAMAZ5Wh0kdASjptyWzQLRErBkX_4nzIJm79SdLkYvkr5toJxPtdxlVXRgcEU1ZuythSGRPKH_CNRsJVMqJxqWBGU4JgVks1LeVZ-sUvQSWVGCMCRRqPdaAEFjFLTeNknLuMDvngc00mE9GeESISENSNiVUc5Zy7pOX0I9NuuUOFl8XjnjIbJBoxX_MnJNhj4pFu3X-l20_ejlKlYrkSFeWHcw0u2_wsCrGuwsNQrrL1iUHSe7ohhB7HLmJ-DQd1BaatUMsRTxLpGR1n_fgq_3xbtm0xsZ83dLJkr8ewNtp63v18LBzJIJmaYW1rICBnmEK8IChDIWjZOk5tQ7ghMNO10bgrnI0Ba0l_arZM3lPISv74kRG_BuS3MiDUqZ5bYD_S5QYknWf6LzBWlSd0aOVScA1ZFBtnuLu4DETCDNivAXqGYbsvDHJsytXgeVWiRog44E1hHR2Xd2W2ax5KsZaxRGwl4KxUF-WnMu8kVgPZFUkIUPQpy7nQNFkyb-F6wemYRZeaPkKy96HD3Zfy_yvEVH4r_LJZs",
    "p": "-2r7Ncw3A6IqNvgGrWtPmGcdljQlNYhtGXFCyj8Juhm-Tn8jyGb45mYpy6rOcCIwiAn8PsCVvJ1DGZlUdJp5DoKPA6KEGviDzO0ANFV0z71h4X_sLk3CZJ7uQ7NuLqxrToZDf2q_ENA6Xg_MFAqC2dKVYCCKdGAiS5flZMEf_B0-0aw1WbNfnXGUKNMNyzIgXH3I10EBFVYfNBnTySGUmmZ3twmeimfYfgyFf56SKyLNj91IUCWqxSPj8XhYHUJYGxMs-4wE8m7ysk7RZnGpQyro-wBXWHhMjqM3wXvWiSjSm_1zVQqcGCdt_6fqaLb5Uy82FFDkxcB4VyMh4uQKsQ",
    "q": "x9SNAr0sk186_9z8WwGGis5_HxOXfiiiqqNO_OaKbHTW1iYdbgQpdPlF-nft8gh4dAKzGQ6hPz0H64lcjL22LWUYjPDkGeByubHuFFbFGlnZpWBXNbceHvYxBrfLBRC2vug1QE21-c8Hww0VnNX0macM0E2sxruEDJXcvdz3jdf-42lPCNPlX73HVmmJACWzubKEsl_VK1MdwWZb_cNL7w6AdwOcug-_YZfMlPv9I8sTMqNwNKppWcrqV1bz0Or04ds1ifA-WR52eaodU8jSMa7j92GShKxtjJ6yaMutLaNtMxsuk1QTAKyAGGUH3HhW_BiS8P2LIGhW5binojWwCw",
    "qi": "XqpyET1rXxpqflIE_5fpVYzpJy316JgBcoFoaQwJXBV2S-AkiOgSHVP_OClZXj2ondHHpShvNbSmFZ8NDunbZhNqDWpXYWFJsdq8-Hcid-c0kipCfh75i799EdLs2HS8zAbbJiVhl5I0QeTE0n3mEUsNWDSMC0pIbZtKuc1Ij849rIxIDhMOKjEMCNUQJVn-FcajTttoamnUHzb4whFmgnMm8JWVDwdFK0Yt4TbchrHg4gpmGHzn1LD4mUPeqstd_JKgZQYMzZawAupN9C3SXDCYjAI6Glskjm-M5eC3yTEFnOE74cHymtI61rU-4-n2aPzMMPsJsLm7U8hzKkHEZg"
}`

func (s *ClientTestSuite) SetupSuite() {
	var err error
	s.signer, err = bundlr.NewArweaveSigner(EMPTY_ARWEAVE_WALLET)
	require.Nil(s.T(), err)
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.client = NewClient(s.ctx, &config.Default().Bundlr)
}

func (s *ClientTestSuite) TearDownSuite() {
	s.cancel()
}

func (s *ClientTestSuite) TestUpload() {
	item := &bundlr.BundleItem{
		Data: []byte("asdf"),
		Tags: bundlr.Tags{bundlr.Tag{Name: "name", Value: "value"}},
	}
	err := item.Sign(s.signer)
	require.Nil(s.T(), err)

	resp, r, err := s.client.Upload(s.ctx, item)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), r)
	require.NotNil(s.T(), resp)

	status, err := s.client.GetStatus(s.ctx, resp.Id)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), status)
}

func (s *ClientTestSuite) TestNestedBundleUpload() {
	// Nested item
	nested := &bundlr.BundleItem{
		Data: []byte("asdf"),
		Tags: bundlr.Tags{{Name: "name", Value: "value"}},
	}

	err := nested.Sign(s.signer)
	require.Nil(s.T(), err)

	// Bundle item
	item := &bundlr.BundleItem{
		Tags: bundlr.Tags{
			{
				Name:  "Bundle-Format",
				Value: "binary",
			},
			{
				Name:  "Bundle-Version",
				Value: "2.0.0",
			},
		},
	}
	err = item.NestBundles([]*bundlr.BundleItem{nested})
	require.Nil(s.T(), err)

	err = item.Sign(s.signer)
	require.Nil(s.T(), err)

	resp, r, err := s.client.Upload(s.ctx, item)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), r)
	require.NotNil(s.T(), resp)
	// fmt.Println(string(r.Body()))
	// fmt.Println(r.Header())

	status, err := s.client.GetStatus(s.ctx, resp.Id)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), status)
}
