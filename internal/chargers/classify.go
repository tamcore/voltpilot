package chargers

import "strings"

// dcPowerThresholdKw is the power above which a station is treated as DC even
// when plug-type strings are missing or ambiguous.
const dcPowerThresholdKw = 43.0

// classifyCurrent infers AC vs DC from plug types and max power. CCS and
// CHAdeMO are always DC; Type 2 (and similar) are AC. A station offering both
// returns CurrentBoth.
func classifyCurrent(plugTypes []string, maxPowerKw float64) Current {
	hasDC, hasAC := false, false
	for _, p := range plugTypes {
		switch normalizePlug(p) {
		case "CCS", "CHADEMO":
			hasDC = true
		case "TYPE_2", "TYPE_1", "TYPE_3", "SCHUKO", "DOMESTIC", "AC":
			hasAC = true
		}
	}
	// Fall back to power when plug types are unknown.
	if !hasDC && !hasAC {
		if maxPowerKw >= dcPowerThresholdKw {
			return CurrentDC
		}
		return CurrentAC
	}
	switch {
	case hasDC && hasAC:
		return CurrentBoth
	case hasDC:
		return CurrentDC
	default:
		return CurrentAC
	}
}

// classifyConnector maps a connector's tariff group / plug group to a Current.
// tariffGroup (AC_CHARGER/DC_CHARGER) is authoritative when present.
func classifyConnector(tariffGroup, plugGroup string, maxPowerKw float64) Current {
	switch strings.ToUpper(strings.TrimSpace(tariffGroup)) {
	case "DC_CHARGER":
		return CurrentDC
	case "AC_CHARGER":
		return CurrentAC
	}
	return classifyCurrent([]string{plugGroup}, maxPowerKw)
}

func normalizePlug(p string) string {
	return strings.ToUpper(strings.TrimSpace(p))
}

// matchesCurrent reports whether a station of the given Current satisfies a
// requested filter ("ac", "dc", or "all"/"").
func matchesCurrent(stationCurrent Current, filter string) bool {
	switch strings.ToLower(strings.TrimSpace(filter)) {
	case "", "all":
		return true
	case "dc":
		return stationCurrent == CurrentDC || stationCurrent == CurrentBoth
	case "ac":
		return stationCurrent == CurrentAC || stationCurrent == CurrentBoth
	default:
		return true
	}
}
