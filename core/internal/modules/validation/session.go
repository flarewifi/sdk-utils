package validation

import (
	"errors"
	"fmt"

	sdkapi "sdk/api"

	sdkutils "github.com/flarewifi/sdk-utils"
)

var validSessionTypes = []sdkapi.SessionType{
	sdkapi.SessionTypeTime,
	sdkapi.SessionTypeData,
	sdkapi.SessionTypeTimeOrData,
}

// ValidateSessionBandwidth checks that download and upload speeds are valid.
// When useGlobalSpeed is true the individual speed values are ignored (the LAN
// global bandwidth will be used at runtime), so no further check is needed.
func ValidateSessionBandwidth(downMbits int, upMbits int, useGlobalSpeed bool) error {
	if useGlobalSpeed {
		return nil
	}
	if downMbits <= 0 {
		return errors.New("download speed must be greater than zero")
	}
	if upMbits <= 0 {
		return errors.New("upload speed must be greater than zero")
	}
	return nil
}

// ValidateSessionType checks that the given session type is one of the known values.
func ValidateSessionType(t sdkapi.SessionType) error {
	if !sdkutils.SliceContains(validSessionTypes, t) {
		return fmt.Errorf("invalid session type %q: must be one of %v", t, validSessionTypes)
	}
	return nil
}

// ValidateSessionData validates a full set of session fields.
// This is called both when creating/updating a session record and when
// starting a session to ensure the data is safe to apply to nftables/TC.
func ValidateSessionData(t sdkapi.SessionType, downMbits int, upMbits int, useGlobalSpeed bool) error {
	if err := ValidateSessionType(t); err != nil {
		return err
	}
	if err := ValidateSessionBandwidth(downMbits, upMbits, useGlobalSpeed); err != nil {
		return err
	}
	return nil
}
