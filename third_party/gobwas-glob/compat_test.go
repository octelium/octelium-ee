package glob

import "testing"

func TestLegacyGlobAndPattern(t *testing.T) {
	pattern, err := Compile("service::**::receivers")
	if err != nil {
		t.Fatal(err)
	}
	var current *Pattern = pattern
	var legacy Glob = current
	for _, tc := range []struct {
		name string
		want bool
	}{
		{name: "service::pipelines::logs::receivers", want: true},
		{name: "service::pipelines::metrics::receivers", want: true},
		{name: "service::pipelines::logs::exporters"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := legacy.Match(tc.name); got != tc.want {
				t.Errorf("Glob.Match(%q) = %v, want %v", tc.name, got, tc.want)
			}
			if got := current.Match(tc.name); got != tc.want {
				t.Errorf("Pattern.Match(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}
