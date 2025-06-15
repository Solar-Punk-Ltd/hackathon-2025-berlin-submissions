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

	statusLabel := widget.NewLabel("Loading grantees...")
	shorten := func(s string) string {
		if len(s) > 12 {
			return s[:6] + "..." + s[len(s)-4:]
		}
		return s
	}

	var granteeList *widget.List // Declare here to be captured

	// loadAndRefreshGrantees fetches the list asynchronously and updates UI
	loadAndRefreshGrantees := func() {
		statusLabel.SetText("Loading grantees...")
		if granteeList != nil {
			granteesData = []string{}
			granteeList.Refresh()
		}

		go func(eglRefToLoad swarm.Address) {
			var fetchedGrantees []string
			var err error
			var newStatusText string

			if !eglRefToLoad.IsZero() {
				fetchedGrantees, err = i.bl.GetGranteeList(context.Background(), eglRefToLoad, false)
				if err != nil {
					i.logger.Log(fmt.Sprintf("Error fetching grantee list for EGL %s: %v", eglRefToLoad.String(), err))
					newStatusText = "Error fetching grantee list."
					fetchedGrantees = []string{}
				} else {
					if len(fetchedGrantees) == 0 {
						newStatusText = fmt.Sprintf("No grantees in EGL: %s", shorten(eglRefToLoad.String()))
					} else {
						newStatusText = fmt.Sprintf("Displaying %d grantees for EGL: %s", len(fetchedGrantees), shorten(eglRefToLoad.String()))
					}
				}
			} else {
				newStatusText = "No grantee list (EGL) loaded. Add a grantee to create one."
				fetchedGrantees = []string{}
			}

			// Directly update UI components and refresh from the goroutine.
			// This is generally acceptable in Fyne v2 for many cases.
			granteesData = fetchedGrantees
			statusLabel.SetText(newStatusText)
			if granteeList != nil {
				granteeList.Refresh()
			}
		}(currentEglRef)
	}

	if eglrefString != "" {
		eglrefBytes, err := hex.DecodeString(eglrefString)
		if err == nil && len(eglrefBytes) == 64 {
			currentEglRef = swarm.NewAddress(eglrefBytes)
		} else {
			i.logger.Log(fmt.Sprintf("Error decoding stored eglref '%s' (len %d) or invalid length: %v. Expected %d bytes.", eglrefString, len(eglrefBytes), err, 64))
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

	loadAndRefreshGrantees() // Initial asynchronous load

	granteeScroll := container.NewScroll(granteeList)
	granteeScroll.SetMinSize(fyne.NewSize(350, 150))

	newGranteeEntry := widget.NewEntry()
	newGranteeEntry.SetPlaceHolder("New grantee public key (hex)")

	historyEntry := widget.NewEntry()
	savedHistoryRef := i.getPreferenceString(historyRefPrefKey)
	if savedHistoryRef != "" {
		historyEntry.SetText(savedHistoryRef)
		historyEntry.SetPlaceHolder("History Ref (from preferences)")
	} else {
		historyEntry.SetPlaceHolder("History Ref (hex, or empty for default)")
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
		i.logger.Log(fmt.Sprintf("Using stamp: %s", batchHex))

		historyRefString := historyEntry.Text
		var resolvedHistoryRef swarm.Address
		if historyRefString == "" {
			resolvedHistoryRef = swarm.ZeroAddress
			i.logger.Log("History reference field empty, using ZeroAddress.")
		} else {
			historyBytes, err := hex.DecodeString(historyRefString)
			if err != nil || (len(historyBytes) != 0 && len(historyBytes) != swarm.HashSize) {
				i.showError(fmt.Errorf("invalid history reference hex string or length: %v", err))
				statusLabel.SetText("Error: Invalid history ref format.")
				return
			}
			resolvedHistoryRef = swarm.NewAddress(historyBytes)
			i.logger.Log(fmt.Sprintf("Using history reference from input: %s", resolvedHistoryRef.String()))
		}

		statusLabel.SetText("Processing request...")

		go func(currentEGLForOp swarm.Address, histRefForOp swarm.Address, granteeToAdd string) {
			var newEglAddressFromAPI, newHistoryAddressFromAPI swarm.Address
			var err error
			var opDesc string

			if currentEGLForOp.IsZero() {
				opDesc = "CreateGrantees"
				i.logger.Log(fmt.Sprintf("Calling %s with History: %s, Grantee: %s", opDesc, histRefForOp.String(), granteeToAdd))
				newEglAddressFromAPI, newHistoryAddressFromAPI, err = i.bl.CreateGrantees(context.Background(), batchHex, histRefForOp, []string{granteeToAdd})
			} else {
				opDesc = "AddRevokeGrantees"
				i.logger.Log(fmt.Sprintf("Calling %s with EGL: %s, History: %s, Grantee: %s", opDesc, currentEGLForOp.String(), histRefForOp.String(), granteeToAdd))
				newEglAddressFromAPI, newHistoryAddressFromAPI, err = i.bl.AddRevokeGrantees(
					context.Background(),
					batchHex,
					currentEGLForOp,
					histRefForOp,
					[]string{granteeToAdd},
					[]string{},
				)
			}

			if err != nil {
				errMsg := fmt.Sprintf("Error in %s: %v", opDesc, err)
				i.logger.Log(errMsg)
				// Directly update UI from goroutine
				i.showError(fmt.Errorf(errMsg)) // This might need to be scheduled if it manipulates UI directly beyond simple dialogs
				statusLabel.SetText(fmt.Sprintf("Failed to %s.", opDesc))
				statusLabel.Refresh() // Refresh the label
				return
			}

			newEglRefString := newEglAddressFromAPI.String()
			newHistoryRefString := newHistoryAddressFromAPI.String()

			i.logger.Log(fmt.Sprintf("Successfully %s. New EGL Ref: %s, New History Ref: %s", opDesc, newEglRefString, newHistoryRefString))

			// Directly update UI components and preferences from goroutine
			i.setPreference(eglrefPrefKey, newEglRefString)
			i.setPreference(historyRefPrefKey, newHistoryRefString)

			currentEglRef = newEglAddressFromAPI

			newGranteeEntry.SetText("")
			historyEntry.SetText(newHistoryRefString)

			// Refresh relevant widgets
			newGranteeEntry.Refresh()
			historyEntry.Refresh()

			loadAndRefreshGrantees() // Reload the list with the new EGL

		}(currentEglRef, resolvedHistoryRef, newGranteeStr)
	})

	layout := container.NewVBox(
		statusLabel,
		granteeScroll,
		widget.NewLabel("New Grantee Public Key:"),
		newGranteeEntry,
		widget.NewLabel("History Reference:"),
		historyEntry,
		submitButton,
	)

	return layout
}

func (i *index) createGranteeCard() fyne.CanvasObject {
	statusLabel := widget.NewLabel("")

	newGranteeEntry := widget.NewEntry()
	newGranteeEntry.SetPlaceHolder("New grantee public key")

	submitButton := widget.NewButton("Create grantee list", func() {
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

		i.logger.Log("creating new grantee list as current EGL is zero address")

		newEglAddressFromAPI, newHistoryAddressFromAPI, err := i.bl.CreateGrantees(context.Background(), batchHex, swarm.ZeroAddress, []string{newGranteeStr})
		if err != nil {
			errMsg := fmt.Sprintf("Error in CreateGrantees: %v", err)
			i.logger.Log(errMsg)
			i.showError(fmt.Errorf(errMsg))
			statusLabel.SetText("Failed to create grantee list.")
			return
		}

		i.logger.Log(fmt.Sprintf("New eglref: %s, New history ref: %s", newEglAddressFromAPI.String(), newHistoryAddressFromAPI.String()))
		newEglRefString := newEglAddressFromAPI.String()
		newHistoryRefString := newHistoryAddressFromAPI.String()

		i.logger.Log(fmt.Sprintf("Successfully updated EGL. New EGL Ref: %s, New History Ref: %s", newEglRefString, newHistoryRefString))

		i.setPreference(eglrefPrefKey, newEglRefString)
		i.setPreference(historyRefPrefKey, newHistoryRefString)

		newGranteeEntry.SetText("")
	})

	layout := container.NewVBox(
		statusLabel,
		widget.NewLabel("New Grantee public key:"),
		newGranteeEntry,
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
