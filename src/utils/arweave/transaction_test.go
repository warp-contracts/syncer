package arweave

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTxMarshalUnmarshal(t *testing.T) {
	transaction := Transaction{
		Format: 2,
		ID:     Base64String("mfbcVuwxI60scpN7JPnqLVvy9DlV8ovpY4PfNbatZ54"),
		LastTx: Base64String("OBjh96vthv8wldC7FqcKduFKFtQoojr1tDLdKFqTl5ZXSXeUjLIAVAWO"),
		Owner:  Base64String("sq9JbppKLlAKtQwalfX5DagnGMlTirditXk7y4jgoeA7DEM0Z6cVPE5xMQ9kz_T9VppP6BFHtHyZCZODercEVWipzkr36tfQkR5EDGUQyLivdxUzbWgVkzw7D27PJEa4cd1Uy6r18rYLqERgbRvAZph5YJZmpSJk7r3MwnQquuktjvSpfCLFwSxP1w879-ss_JalM9ICzRi38henONio8gll6GV9-omrWwRMZer_15bspCK5txCwpY137nfKwKD5YBAuzxxcj424M7zlSHlsafBwaRwFbf8gHtW03iJER4lR4GxeY0WvnYaB3KDISHQp53a9nlbmiWO5WcHHYsR83OT2eJ0Pl3RWA-_imk_SNwGQTCjmA6tf_UVwL8HzYS2iyuu85b7iYK9ZQoh8nqbNC6qibICE4h9Fe3bN7AgitIe9XzCTOXDfMr4ahjC8kkqJ1z4zNAI6-Leei_Mgd8JtZh2vqFNZhXK0lSadFl_9Oh3AET7tUds2E7s-6zpRPd9oBZu6-kNuHDRJ6TQhZSwJ9ZO5HYsccb_G_1so72aXJymR9ggJgWr4J3bawAYYnqmvmzGklYOlE_5HVnMxf-UxpT7ztdsHbc9QEH6W2bzwxbpjTczEZs3JCCB3c-NewNHsj9PYM3b5tTlTNP9kNAwPZHWpt11t79LuNkNGt9LfOek"),
		Tags: []Tag{{Name: Base64String("QXBwLU5hbWU"), Value: Base64String("QnVuZGxy")},
			{Name: Base64String("QWN0aW9u"), Value: Base64String("QnVuZGxl")}},
		Target:   Base64String(""),
		Quantity: "0",
		Data:     Base64String(""),
		DataSize: BigInt{*big.NewInt(16284773), true},
		DataRoot: Base64String("icJ7moV2pks3J8SP529BUezuHitNJME2oZXuYVuL2GU"),
		Reward: "13211305425",
		Signature: Base64String("NICLeV8P4c2PilRUpDAeZBKLU1RPU37bDQTknsg3DtdA4QcM87E3ZA34S6_Bolp1A5OUq-Or8jcvh1uDIcaIFwSA1zdMNem_6M-OFUJmprNfZmuC3TrgmhwFTIECHCRycByDp_vq14Jp2CsVC9sfmOm_SgZT2dig-MBQprMkO6aa3KCQuK02fGpmlL-D87Ixzub_mpBy10KKIbou4SMZQ9FJ6yH6EVUX4M4OUyaeOMWLU8SI8bSOCP0b6xxlvePbVVx6FnLhNux7_8oQjMxfuGHejHqYdC3U37oRJVszNhh8ENTR6feNO7mhS657X-IXMORLj8UurYb11CkxFV7NCRhR5kZn_h747N1CikY-FlIz1LAI6B0KZsYzrveRCbcce86ft265qH3KioLm7II7PlrvE_UOtiIbeFSdh1_bvL_wNzcvG1xxmzTifYSjx7RN1CLXUipD-TYTR-kxqqprjPZsBM2qWCDPa_9PPU02g6v6TogETi87ZKqOE_-mizEpH8NroNMUVHcIZrSA_hp18Woo7ysLMFPCjPPdbnUQk-5-GewZreLaNH2WSh-_O6VUmiC0Yc9fbaFQn7-PVPuMugFD6TIQtbJHPEauLPDixTXq6oUVe5v1lib07la-pSfxVjDYuobawo755FH48vooz5JypHfLREjFDsM0KfskCbY"),
	}

	size := transaction.Size()
	buf := make([]byte, size)
	bufSize, err := transaction.MarshalTo(buf)

	require.NoError(t, err)
	require.Equal(t, size, bufSize)

	parsed := Transaction{}
	err = parsed.Unmarshal(buf)
	require.NoError(t, err)

	require.Equal(t, transaction, parsed)
}
