package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"core/internal/api"
	"core/internal/modules/activation"
	machineuid "core/internal/modules/machine-uid"
	activationview "core/resources/views/activation"
	cmd "core/utils/shell"
)

const (
	ActivationURL = "/activation"
)

func NewActivationCtrl(g *api.CoreGlobals) ActivationCtrl {
	return ActivationCtrl{g}
}

type ActivationCtrl struct {
	g *api.CoreGlobals
}

func (ctrl *ActivationCtrl) ActivationPage(w http.ResponseWriter, r *http.Request) {
	_, machineID := machineuid.GetMachineUID()

	page := activationview.ActivationPage(&activationview.ActivationPageData{
		API:       ctrl.g.CoreAPI,
		MachineID: machineID,
	})

	if err := page.Render(r.Context(), w); err != nil {
		w.Write([]byte("\n\nTemplate Error:" + err.Error()))
	}
}

// GetActivationStatus returns the current activation status without triggering validation.
// This is a lightweight endpoint for polling.
func (ctrl *ActivationCtrl) GetActivationStatus(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"activated": activation.IsActivated.Load(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ValidateActivation triggers a re-validation of the activation status
func (ctrl *ActivationCtrl) ValidateActivation(w http.ResponseWriter, r *http.Request) {
	// Trigger validation synchronously (blocking) to wait for RPC response
	activation.Validate()

	// If activation succeeded, trigger system reboot to ensure proper initialization
	if activation.IsActivated.Load() {
		// Schedule reboot in a goroutine to allow response to be sent first
		go func() {
			time.Sleep(2 * time.Second) // Give time for response to reach client
			cmd.Exec("reboot", nil)
		}()
	}

	// Return the actual validation result
	response := map[string]interface{}{
		"activated":  activation.IsActivated.Load(),
		"validating": activation.IsValidating.Load(),
	}

	// If there was an error during validation, include it
	if errVal := activation.ActivationError.Load(); errVal != nil {
		if err, ok := errVal.(error); ok {
			response["error"] = err.Error()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
