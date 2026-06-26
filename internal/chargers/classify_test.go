package chargers

import "testing"

func TestClassifyCurrent(t *testing.T) {
	cases := []struct {
		name  string
		plugs []string
		power float64
		want  Current
	}{
		{"ccs is dc", []string{"CCS"}, 300, CurrentDC},
		{"chademo is dc", []string{"CHADEMO"}, 50, CurrentDC},
		{"type2 is ac", []string{"TYPE_2"}, 22, CurrentAC},
		{"mixed is both", []string{"CCS", "TYPE_2"}, 300, CurrentBoth},
		{"unknown low power falls back to ac", nil, 11, CurrentAC},
		{"unknown high power falls back to dc", nil, 150, CurrentDC},
		{"lowercase normalises", []string{"ccs"}, 50, CurrentDC},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := classifyCurrent(c.plugs, c.power); got != c.want {
				t.Fatalf("classifyCurrent(%v,%v) = %q, want %q", c.plugs, c.power, got, c.want)
			}
		})
	}
}

func TestClassifyConnector(t *testing.T) {
	if got := classifyConnector("DC_CHARGER", "CCS", 300); got != CurrentDC {
		t.Fatalf("tariff DC_CHARGER -> %q, want dc", got)
	}
	if got := classifyConnector("AC_CHARGER", "TYPE_2", 22); got != CurrentAC {
		t.Fatalf("tariff AC_CHARGER -> %q, want ac", got)
	}
	// Falls back to plug group when tariff group is absent.
	if got := classifyConnector("", "CHADEMO", 50); got != CurrentDC {
		t.Fatalf("empty tariff, CHADEMO plug -> %q, want dc", got)
	}
}

func TestMatchesCurrent(t *testing.T) {
	cases := []struct {
		station Current
		filter  string
		want    bool
	}{
		{CurrentDC, "dc", true},
		{CurrentDC, "ac", false},
		{CurrentAC, "ac", true},
		{CurrentBoth, "ac", true},
		{CurrentBoth, "dc", true},
		{CurrentAC, "", true},
		{CurrentAC, "all", true},
	}
	for _, c := range cases {
		if got := matchesCurrent(c.station, c.filter); got != c.want {
			t.Fatalf("matchesCurrent(%q,%q) = %v, want %v", c.station, c.filter, got, c.want)
		}
	}
}
