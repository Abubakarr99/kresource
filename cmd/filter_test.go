package cmd

import "testing"

func TestParseFilterEmpty(t *testing.T) {
	f, err := parseFilter("")
	if err != nil {
		t.Fatal(err)
	}
	if !f.matches(ResourceRow{Kind: "Deployment"}) {
		t.Fatal("expected an empty filter to match everything")
	}
}

func TestParseFilterTypeAndName(t *testing.T) {
	f, err := parseFilter("type=deployment,name=web")
	if err != nil {
		t.Fatal(err)
	}

	if !f.matches(ResourceRow{Kind: "Deployment", Name: "web-frontend"}) {
		t.Fatal("expected type=deployment,name=web to match a Deployment named web-frontend (substring, case-insensitive)")
	}
	if f.matches(ResourceRow{Kind: "StatefulSet", Name: "web-frontend"}) {
		t.Fatal("expected a StatefulSet to be excluded by type=deployment")
	}
	if f.matches(ResourceRow{Kind: "Deployment", Name: "db"}) {
		t.Fatal("expected name=web to exclude a row named db")
	}
}

func TestParseFilterInit(t *testing.T) {
	f, err := parseFilter("init=true")
	if err != nil {
		t.Fatal(err)
	}
	if !f.matches(ResourceRow{Init: true}) {
		t.Fatal("expected init=true to match an init container row")
	}
	if f.matches(ResourceRow{Init: false}) {
		t.Fatal("expected init=true to exclude a non-init container row")
	}
}

func TestParseFilterErrors(t *testing.T) {
	cases := []string{"bogus=x", "type", "init=notabool"}
	for _, expr := range cases {
		if _, err := parseFilter(expr); err == nil {
			t.Fatalf("expected an error for --filter %q", expr)
		}
	}
}

func TestFilterRows(t *testing.T) {
	rows := []ResourceRow{
		{Kind: "Deployment", Name: "web"},
		{Kind: "Pod", Name: "bare-pod"},
	}
	f, err := parseFilter("type=pod")
	if err != nil {
		t.Fatal(err)
	}
	out := filterRows(rows, f)
	if len(out) != 1 || out[0].Name != "bare-pod" {
		t.Fatalf("expected only the Pod row to survive, got %+v", out)
	}
}
