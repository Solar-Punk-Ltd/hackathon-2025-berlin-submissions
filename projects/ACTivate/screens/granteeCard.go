package screens

import (
	"context"
	"encoding/hex"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/ethersphere/bee/v2/pkg/postage"
	"github.com/ethersphere/bee/v2/pkg/swarm"
)

func (i *index) showGranteeCard() fyne.CanvasObject {
	eglrefPref := i.getPreferenceString(eglrefPrefKey)
	grantees := []string{}
	eglref := swarm.ZeroAddress
	var (
		err         error
		eglrefBytes []byte
	)

	if eglrefPref != "" {
		i.logger.Log(fmt.Sprintf("Using eglref from preferences: %s", eglrefPref))

		eglrefBytes, err = hex.DecodeString(eglrefPref)
		if err != nil {
			return container.NewVBox(widget.NewLabel("Error decoding eglrefBytes"))
		}
		eglref = swarm.NewAddress(eglrefBytes)

		grantees, err = i.bl.GetGranteeList(context.TODO(), eglref, false)
		if err != nil {
			i.showError(err)
			return container.NewVBox(widget.NewLabel("Error fetching grantee list"))
		}
	}

	granteeList := widget.NewList(
		func() int {
			return len(grantees)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("grantee list")
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			item.(*widget.Label).SetText(grantees[id])
		},
	)

	granteeScroll := container.NewScroll(granteeList)
	granteeScroll.SetMinSize(fyne.NewSize(200, 150))

	granteeEntry := widget.NewEntry()
	granteeEntry.SetPlaceHolder("eglref")

	historyEntry := widget.NewEntry()
	savedHref := i.getPreferenceString(historyRefPrefKey)
	if savedHref != "" {
		historyEntry.SetText(savedHref)
		historyEntry.SetPlaceHolder(savedHref)
	} else {
		historyEntry.SetPlaceHolder("history")
	}

	submitButton := widget.NewButton("Add grantee", func() {
		addlist := []string{granteeEntry.Text}
		i.logger.Log(fmt.Sprintf("Submitted grantee: %s", addlist))

		batchHex := hex.EncodeToString(i.getStamp().ID())
		i.logger.Log(fmt.Sprintf("using stamp: %s", batchHex))

		historyBytes, err := hex.DecodeString(historyEntry.Text)
		if err != nil {
			i.logger.Log(fmt.Sprintf("Invalid history reference: %s", err.Error()))
			return
		}
		historyRef := swarm.NewAddress(historyBytes)

		revokelist := []string{}
		encryptedGlist, newHref, err := i.bl.AddRevokeGrantees(context.TODO(), batchHex, eglref, historyRef, addlist, revokelist)
		if err != nil {
			i.logger.Log(fmt.Sprintf("error in AddRevokeGrantees: %v", err))
			return
		}

		i.logger.Log(fmt.Sprintf("Encrypted grantees: %s, New history ref: %s", encryptedGlist, newHref))

		i.setPreference(eglrefPrefKey, hex.EncodeToString(encryptedGlist.Bytes()))
		i.setPreference(historyRefPrefKey, hex.EncodeToString(newHref.Bytes()))

		granteeEntry.SetText("")
		historyEntry.SetText("")
	})

	content := container.NewVBox(
		granteeScroll,
		granteeEntry,
		historyEntry,
		submitButton,
	)

	return content
}

func (i *index) getStamp() *postage.StampIssuer {
	stamps := i.bl.GetUsableBatches()

	if len(stamps) != 0 {
		return stamps[0]
	}
	return nil
}
