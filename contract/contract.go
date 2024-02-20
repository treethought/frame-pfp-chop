package contract

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"log"
	"os"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/thirdweb-dev/go-sdk/v2/thirdweb"
	fc "github.com/treethought/impression-frame/farcaster"

	"github.com/wabarc/ipfs-pinner/pkg/pinata"
)

var (
	RPC_ENDPOINT     = os.Getenv("RPC_URL")
	PRIVATE_KEY      = os.Getenv("PRIVATE_KEY")
	SECRET_KEY       = os.Getenv("SECRET_KEY")
	CLIENT_ID        = os.Getenv("CLIENT_ID")
	CONTRACT_ADDRESS = os.Getenv("CONTRACT_ADDRESS")

	PINATA_API_KEY    = os.Getenv("PINATA_API_KEY")
	PINATA_SECRET_KEY = os.Getenv("PINATA_SECRET_KEY")
	TW_GATEWAY        = os.Getenv("TW_GATEWAY")

	description = "PFP chopped & screwed"
)

type Contract struct {
	sdk      *thirdweb.ThirdwebSDK
	contract *thirdweb.SmartContract
}

func NewContract() (*Contract, error) {
	sdk, err := thirdweb.NewThirdwebSDK(RPC_ENDPOINT, &thirdweb.SDKOptions{
		SecretKey:  SECRET_KEY,
		PrivateKey: PRIVATE_KEY,
		GatewayUrl: TW_GATEWAY,
	})
	if err != nil {
		return nil, err
	}

	contract, err := sdk.GetContract(context.Background(), CONTRACT_ADDRESS)
	if err != nil {
		return nil, err
	}
	return &Contract{sdk, contract}, nil
}

func (c *Contract) Mint(ctx context.Context, img image.Image, user *fc.User) (*types.Transaction, error) {
	if len(user.Verfications) == 0 {
		return nil, fmt.Errorf("user has no verifications")
	}
	to := user.Verfications[0]
	fmt.Println("MintTo: ", to)

	ctx = context.Background()
	// Encode the image to JPEG format
	var buf bytes.Buffer
	err := png.Encode(&buf, img) // Change "jpeg" to "png" or "gif" if needed
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Pinning image to IPFS")
	pnt := pinata.Pinata{Apikey: PINATA_API_KEY, Secret: PINATA_SECRET_KEY}
	cid, err := pnt.PinWithBytes(buf.Bytes())
	if err != nil {
		log.Println("Error pinning image to IPFS: ", err)
		return nil, err
	}
	log.Println("CID: ", cid)

	imgUrl := fmt.Sprintf("ipfs://%s", cid)

	md := &thirdweb.EditionMetadataInput{
		Supply: 1,
		Metadata: &thirdweb.NFTMetadataInput{
			Name:        user.Username,
			Description: description,
			Image:       imgUrl,
		},
	}

	return c.contract.ERC1155.MintTo(context.Background(), to, md)
}
