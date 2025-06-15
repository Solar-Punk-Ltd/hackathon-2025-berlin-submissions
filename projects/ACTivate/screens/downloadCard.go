package screens

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"io"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethersphere/bee/v2/pkg/swarm"
)

func (i *index) showDownloadCard() *widget.Card {
	dlForm := i.downloadForm()
	return widget.NewCard("Download", "download content from swarm", dlForm)
}

func (i *index) downloadForm() *widget.Form {
	hash := widget.NewEntry()
	hash.SetPlaceHolder("Swarm Hash")
	dlForm := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Swarm Hash", Widget: hash, HintText: "Swarm Hash"},
		},
		OnSubmit: func() {
			// dlAddr, err := swarm.ParseHexAddress(hash.Text)
			// if err != nil {
			// 	i.showError(err)
			// 	return
			// }
			if hash.Text == "" {
				i.showError(fmt.Errorf("please enter a hash"))
				return
			}
			go func() {
				i.showProgressWithMessage(fmt.Sprintf("Downloading %s", shortenHashOrAddress(hash.Text)))
				//ref, fileName, err := i.bl.GetBzz(context.Background(), dlAddr, nil, nil, nil)
				bytehash, _ := swarm.ParseHexAddress(i.getPreferenceString("event32ByteHex"))
				acthash, _ := swarm.ParseHexAddress(i.getPreferenceString("eventActRef"))
				publisherHex := i.getPreferenceString("eventPublicKey")

				// Parse the public key as ECDSA public key from hex encoded string
				var publisher *ecdsa.PublicKey
				if publisherHex != "" {
					// Remove 0x prefix if present
					if len(publisherHex) > 2 && publisherHex[:2] == "0x" {
						publisherHex = publisherHex[2:]
					}

					// Decode hex string to bytes
					publicKeyBytes, err := hex.DecodeString(publisherHex)
					if err != nil {
						i.hideProgress()
						i.showError(fmt.Errorf("failed to decode public key hex: %w", err))
						return
					}

					// Parse ECDSA public key
					publisher, err = crypto.UnmarshalPubkey(publicKeyBytes)
					if err != nil {
						i.hideProgress()
						i.showError(fmt.Errorf("failed to parse ECDSA public key: %w", err))
						return
					}

					fmt.Printf("Successfully parsed ECDSA public key: %x\n", crypto.FromECDSAPub(publisher))
				}

				fmt.Println("bytehash", i.getPreferenceString("event32ByteHex"))
				fmt.Println("acthash", i.getPreferenceString("eventActRef"))
				fmt.Println("publisher", publisherHex)
				ref, err := i.bl.GetBytes(context.Background(), bytehash, publisher, &acthash, nil)
				if err != nil {
					i.hideProgress()
					i.showError(err)
					return
				}
				hash.SetText("")
				data, err := io.ReadAll(ref)
				if err != nil {
					i.hideProgress()
					i.showError(err)
					return
				}
				i.hideProgress()
				saveFile := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
					if err != nil {
						i.showError(err)
						return
					}
					if writer == nil {
						return
					}
					_, err = writer.Write(data)
					if err != nil {
						i.showError(err)
						return
					}
					writer.Close()
				}, i.Window)
				//saveFile.SetFileName(fileName)
				saveFile.Show()
			}()
		},
	}

	return dlForm
}
