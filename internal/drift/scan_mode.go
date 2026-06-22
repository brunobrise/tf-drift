package drift

import (
	"fmt"
	"strings"
)

type ScanMode string

const (
	ScanModeBoth  ScanMode = "both"
	ScanModeDrift ScanMode = "drift"
	ScanModePlan  ScanMode = "plan"
)

func ParseScanMode(value string) (ScanMode, error) {
	switch ScanMode(strings.ToLower(strings.TrimSpace(value))) {
	case "", ScanModeBoth:
		return ScanModeBoth, nil
	case ScanModeDrift:
		return ScanModeDrift, nil
	case ScanModePlan:
		return ScanModePlan, nil
	default:
		return "", fmt.Errorf("invalid scan mode %q: expected both, drift, or plan", value)
	}
}

func (m ScanMode) includes(classification ChangeClassification) bool {
	if m == "" {
		m = ScanModeBoth
	}
	switch m {
	case ScanModeDrift:
		return classification == ChangeClassificationExternalDrift
	case ScanModePlan:
		return classification == ChangeClassificationPlannedChange
	default:
		return true
	}
}
