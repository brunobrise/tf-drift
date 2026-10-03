package drift

import (
	"testing"
)

func TestExtractAttributeDiffs(t *testing.T) {
	change := Change{
		Actions: []string{"update"},
		Before: map[string]interface{}{
			"instance_type": "t2.micro",
			"tags": map[string]interface{}{
				"Environment": "dev",
			},
			"monitoring": false,
		},
		After: map[string]interface{}{
			"instance_type": "t3.micro",
			"tags": map[string]interface{}{
				"Environment": "prod",
			},
			"monitoring": false, // unchanged
		},
		AfterUnknown: map[string]interface{}{
			"ami": true, // unknown after apply
		},
	}

	diffs := ExtractAttributeDiffs(change)
	if len(diffs) != 3 {
		t.Fatalf("expected 3 attribute diffs (ami, instance_type, tags), got %d: %+v", len(diffs), diffs)
	}

	diffMap := make(map[string]AttributeDiff)
	for _, d := range diffs {
		diffMap[d.Attribute] = d
	}

	if d, ok := diffMap["instance_type"]; !ok {
		t.Errorf("missing instance_type diff")
	} else {
		if d.Before != "t2.micro" || d.After != "t3.micro" {
			t.Errorf("instance_type diff mismatch: got before=%v, after=%v", d.Before, d.After)
		}
	}

	if d, ok := diffMap["ami"]; !ok {
		t.Errorf("missing ami diff")
	} else {
		if d.After != "(known after apply)" {
			t.Errorf("expected ami after to be '(known after apply)', got %v", d.After)
		}
	}
}

func TestFormatAttributeValue(t *testing.T) {
	cases := []struct {
		input    interface{}
		expected string
	}{
		{nil, "null"},
		{"hello", "\"hello\""},
		{123, "123"},
		{true, "true"},
		{[]interface{}{"a", "b"}, "[\"a\",\"b\"]"},
	}

	for _, tc := range cases {
		got := FormatAttributeValue(tc.input)
		if got != tc.expected {
			t.Errorf("FormatAttributeValue(%v) = %q; want %q", tc.input, got, tc.expected)
		}
	}
}
