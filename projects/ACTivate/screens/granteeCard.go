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
	var granteesData []string

	eglrefString := i.getPreferenceString(eglrefPrefKey)
	currentEglRef := swarm.ZeroAddress

	statusLabel := widget.NewLabel("")
	shorten := func(s string) string {
		if len(s) > 12 {
			return s[:6] + "..." + s[len(s)-4:]
		}
		return s
	}

	var granteeList *widget.List
	refreshGranteeDisplay := func() {
		if !currentEglRef.IsZero() {
			fetchedGrantees, err := i.bl.GetGranteeList(context.Background(), currentEglRef, false)
			if err != nil {
				i.logger.Log(fmt.Sprintf("Error fetching grantee list for EGL %s: %v", currentEglRef.String(), err))
				statusLabel.SetText("Error fetching grantee list.")
				granteesData = []string{}
			} else {
				granteesData = fetchedGrantees
				if len(granteesData) == 0 {
					statusLabel.SetText(fmt.Sprintf("No grantees in EGL: %s", shorten(currentEglRef.String())))
				} else {
					statusLabel.SetText(fmt.Sprintf("Displaying %d grantees for EGL: %s", len(granteesData), shorten(currentEglRef.String())))
				}
			}
		} else {
			statusLabel.SetText("No grantee list loaded. Add a grantee to create one.")
			granteesData = []string{}
		}
		if granteeList != nil {
			granteeList.Refresh()
		}
	}

	if eglrefString != "" {
		eglrefBytes, err := hex.DecodeString(eglrefString)
		if err == nil && len(eglrefBytes) == 64 {
			currentEglRef = swarm.NewAddress(eglrefBytes)
		} else {
			i.logger.Log(fmt.Sprintf("Error decoding stored eglref '%s' or invalid length: %v", eglrefString, err))
		}
	}

	granteeList = widget.NewList(
		func() int {
			return len(granteesData)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template grantee")
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id < len(granteesData) {
				item.(*widget.Label).SetText(granteesData[id])
			}
		},
	)
	refreshGranteeDisplay()
	granteeScroll := container.NewScroll(granteeList)
	granteeScroll.SetMinSize(fyne.NewSize(350, 150))

	newGranteeEntry := widget.NewEntry()
	newGranteeEntry.SetPlaceHolder("New grantee public key")

	historyEntry := widget.NewEntry()
	savedHistoryRef := i.getPreferenceString(historyRefPrefKey)
	if savedHistoryRef != "" {
		historyEntry.SetText(savedHistoryRef)
		historyEntry.SetPlaceHolder("History Ref (from preferences)")
	} else {
		historyEntry.SetPlaceHolder("History Ref")
	}

	submitButton := widget.NewButton("Add Grantee / Update List", func() {
		newGranteeStr := newGranteeEntry.Text
		if newGranteeStr == "" {
			i.showError(fmt.Errorf("new grantee public key cannot be empty"))
			return
		}

		stamp := i.getStamp()
		if stamp == nil {
			i.showError(fmt.Errorf("no usable postage stamp found"))
			statusLabel.SetText("Error: No usable postage stamp.")
			return
		}
		batchHex := hex.EncodeToString(stamp.ID())
		i.logger.Log(fmt.Sprintf("Using stamp: %s for AddRevokeGrantees", batchHex))

		historyRefString := historyEntry.Text
		var resolvedHistoryRef swarm.Address
		if historyRefString == "" {
			resolvedHistoryRef = swarm.ZeroAddress
			i.logger.Log("History reference field is empty, using ZeroAddress for API call.")
		} else {
			historyBytes, err := hex.DecodeString(historyRefString)
			if err != nil || (len(historyBytes) != 0 && len(historyBytes) != swarm.HashSize) {
				i.showError(fmt.Errorf("invalid history reference hex string or length: %v", err))
				statusLabel.SetText("Error: Invalid history reference format.")
				return
			}
			resolvedHistoryRef = swarm.NewAddress(historyBytes)
			i.logger.Log(fmt.Sprintf("Using history reference from input: %s", resolvedHistoryRef.String()))
		}

		i.logger.Log(fmt.Sprintf("Calling AddRevokeGrantees with EGL: %s, History: %s", currentEglRef.String(), resolvedHistoryRef.String()))

		var newEglAddressFromAPI, newHistoryAddressFromAPI swarm.Address

		if currentEglRef.IsZero() {
			i.logger.Log("creating new grantee list as current EGL is zero address")

			var err error
			newEglAddressFromAPI, newHistoryAddressFromAPI, err = i.bl.CreateGrantees(context.Background(), batchHex, resolvedHistoryRef, []string{newGranteeStr})
			if err != nil {
				errMsg := fmt.Sprintf("Error in CreateGrantees: %v", err)
				i.logger.Log(errMsg)
				i.showError(fmt.Errorf(errMsg))
				statusLabel.SetText("Failed to create grantee list.")
				return
			}

			i.logger.Log(fmt.Sprintf("New eglref: %s, New history ref: %s", newEglAddressFromAPI.String(), newHistoryAddressFromAPI.String()))
		} else {
			i.logger.Log("adding new grantee")

			var err error
			newEglAddressFromAPI, newHistoryAddressFromAPI, err = i.bl.AddRevokeGrantees(
				context.Background(),
				batchHex,
				currentEglRef,
				resolvedHistoryRef,
				[]string{newGranteeStr},
				[]string{},
			)
			if err != nil {
				errMsg := fmt.Sprintf("Error in AddRevokeGrantees: %v", err)
				i.logger.Log(errMsg)
				i.showError(fmt.Errorf(errMsg))
				statusLabel.SetText("Failed to update grantee list.")
				return
			}
		}

		newEglRefString := newEglAddressFromAPI.String()
		newHistoryRefString := newHistoryAddressFromAPI.String()

		i.logger.Log(fmt.Sprintf("Successfully updated EGL. New EGL Ref: %s, New History Ref: %s", newEglRefString, newHistoryRefString))

		i.setPreference(eglrefPrefKey, newEglRefString)
		i.setPreference(historyRefPrefKey, newHistoryRefString)

		newGranteeEntry.SetText("")
		historyEntry.SetText(newHistoryRefString)

		refreshGranteeDisplay()
	})

	layout := container.NewVBox(
		statusLabel,
		granteeScroll,
		widget.NewLabel("New Grantee public key:"),
		newGranteeEntry,
		widget.NewLabel("History Reference:"),
		historyEntry,
		submitButton,
	)

	return layout
}

func (i *index) getStamp() *postage.StampIssuer {
	stamps := i.bl.GetUsableBatches()

	if len(stamps) != 0 {
		return stamps[0]
	}
	return nil
}
