// Copyright (c) The OpenTofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2023 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package funcs

import (
	"fmt"
	"testing"

	"github.com/opentofu/opentofu/internal/lang/marks"
	"github.com/zclconf/go-cty/cty"
)

func TestSensitive(t *testing.T) {
	tests := []struct {
		Input   cty.Value
		WantErr string
	}{
		{
			cty.NumberIntVal(1),
			``,
		},
		{
			// Unknown values stay unknown while becoming sensitive
			cty.UnknownVal(cty.String),
			``,
		},
		{
			// Null values stay unknown while becoming sensitive
			cty.NullVal(cty.String),
			``,
		},
		{
			// DynamicVal can be marked as sensitive
			cty.DynamicVal,
			``,
		},
		{
			// The marking is shallow only
			cty.ListVal([]cty.Value{cty.NumberIntVal(1)}),
			``,
		},
		{
			// A value already marked is allowed and stays marked
			cty.NumberIntVal(1).Mark(marks.Sensitive),
			``,
		},
		{
			// A value with some non-standard mark gets "fixed" to be marked
			// with the standard "sensitive" mark. (This situation occurring
			// would imply an inconsistency/bug elsewhere, so we're just
			// being robust about it here.)
			cty.NumberIntVal(1).Mark("bloop"),
			``,
		},
		{
			// A value deep already marked is allowed and stays marked,
			// _and_ we'll also mark the outer collection as sensitive.
			cty.ListVal([]cty.Value{cty.NumberIntVal(1).Mark(marks.Sensitive)}),
			``,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("sensitive(%#v)", test.Input), func(t *testing.T) {
			got, err := Sensitive(test.Input)

			if test.WantErr != "" {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				if got, want := err.Error(), test.WantErr; got != want {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", got, want)
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.HasMark(marks.Sensitive) {
				t.Errorf("result is not marked sensitive")
			}

			gotRaw, gotMarks := got.Unmark()
			if len(gotMarks) != 1 {
				// We're only expecting to have the "sensitive" mark we checked
				// above. Any others are an error, even if they happen to
				// appear alongside "sensitive". (We might change this rule
				// if someday we decide to use marks for some additional
				// unrelated thing in OpenTofu, but currently we assume that
				// _all_ marks imply sensitive, and so returning any other
				// marks would be confusing.)
				t.Errorf("extraneous marks %#v", gotMarks)
			}

			// Disregarding shallow marks, the result should have the same
			// effective value as the input.
			wantRaw, _ := test.Input.Unmark()
			if !gotRaw.RawEquals(wantRaw) {
				t.Errorf("wrong unmarked result\ngot:  %#v\nwant: %#v", got, wantRaw)
			}
		})
	}
}

func TestNonsensitive(t *testing.T) {
	tests := []struct {
		Input   cty.Value
		WantErr string
	}{
		{
			cty.NumberIntVal(1).Mark(marks.Sensitive),
			``,
		},
		{
			cty.DynamicVal.Mark(marks.Sensitive),
			``,
		},
		{
			cty.UnknownVal(cty.String).Mark(marks.Sensitive),
			``,
		},
		{
			cty.NullVal(cty.EmptyObject).Mark(marks.Sensitive),
			``,
		},
		{
			// The inner sensitive remains afterwards
			cty.ListVal([]cty.Value{cty.NumberIntVal(1).Mark(marks.Sensitive)}).Mark(marks.Sensitive),
			``,
		},

		// Passing a value that is already non-sensitive is not an error,
		// as this function may be used with specific to ensure that all
		// values are indeed non-sensitive
		{
			cty.NumberIntVal(1),
			``,
		},
		{
			cty.NullVal(cty.String),
			``,
		},

		// Unknown values may become sensitive once they are known, so we
		// permit them to be marked nonsensitive.
		{
			cty.DynamicVal,
			``,
		},
		{
			cty.UnknownVal(cty.String),
			``,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("nonsensitive(%#v)", test.Input), func(t *testing.T) {
			got, err := Nonsensitive(test.Input)

			if test.WantErr != "" {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				if got, want := err.Error(), test.WantErr; got != want {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", got, want)
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if got.HasMark(marks.Sensitive) {
				t.Errorf("result is still marked sensitive")
			}
			wantRaw, _ := test.Input.Unmark()
			if !got.RawEquals(wantRaw) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Input)
			}
		})
	}
}

func TestIsSensitive(t *testing.T) {
	tests := []struct {
		Input       cty.Value
		IsSensitive bool
	}{
		{
			cty.NumberIntVal(1).Mark(marks.Sensitive),
			true,
		},
		{
			cty.NumberIntVal(1),
			false,
		},
		{
			cty.DynamicVal.Mark(marks.Sensitive),
			true,
		},
		{
			cty.DynamicVal,
			false,
		},
		{
			cty.UnknownVal(cty.String).Mark(marks.Sensitive),
			true,
		},
		{
			cty.UnknownVal(cty.String),
			false,
		},
		{
			cty.NullVal(cty.EmptyObject).Mark(marks.Sensitive),
			true,
		},
		{
			cty.NullVal(cty.EmptyObject),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("issensitive(%#v)", test.Input), func(t *testing.T) {
			got, err := IsSensitive(test.Input)

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if got.Equals(cty.BoolVal(test.IsSensitive)).False() {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, cty.BoolVal(test.IsSensitive))
			}
		})
	}
}

func TestFlipSensitive(t *testing.T) {
	tests := []struct {
		name     string
		input    cty.Value
		expected cty.Value
	}{
		{
			name:     "non-sensitive to sensitive string",
			input:    cty.StringVal("hello"),
			expected: cty.StringVal("hello").Mark(marks.Sensitive),
		},
		{
			name:     "sensitive to non-sensitive string",
			input:    cty.StringVal("secret").Mark(marks.Sensitive),
			expected: cty.StringVal("secret"),
		},
		{
			name:     "non-sensitive to sensitive number",
			input:    cty.NumberIntVal(42),
			expected: cty.NumberIntVal(42).Mark(marks.Sensitive),
		},
		{
			name:     "sensitive to non-sensitive number",
			input:    cty.NumberIntVal(42).Mark(marks.Sensitive),
			expected: cty.NumberIntVal(42),
		},
		{
			name:     "non-sensitive to sensitive bool",
			input:    cty.BoolVal(true),
			expected: cty.BoolVal(true).Mark(marks.Sensitive),
		},
		{
			name:     "sensitive to non-sensitive bool",
			input:    cty.BoolVal(false).Mark(marks.Sensitive),
			expected: cty.BoolVal(false),
		},
		{
			name:     "non-sensitive to sensitive list",
			input:    cty.ListVal([]cty.Value{cty.StringVal("a"), cty.StringVal("b")}),
			expected: cty.ListVal([]cty.Value{cty.StringVal("a"), cty.StringVal("b")}).Mark(marks.Sensitive),
		},
		{
			name:     "sensitive to non-sensitive list",
			input:    cty.ListVal([]cty.Value{cty.StringVal("a"), cty.StringVal("b")}).Mark(marks.Sensitive),
			expected: cty.ListVal([]cty.Value{cty.StringVal("a"), cty.StringVal("b")}),
		},
		{
			name:     "non-sensitive to sensitive map",
			input:    cty.MapVal(map[string]cty.Value{"key": cty.StringVal("value")}),
			expected: cty.MapVal(map[string]cty.Value{"key": cty.StringVal("value")}).Mark(marks.Sensitive),
		},
		{
			name:     "sensitive to non-sensitive map",
			input:    cty.MapVal(map[string]cty.Value{"key": cty.StringVal("value")}).Mark(marks.Sensitive),
			expected: cty.MapVal(map[string]cty.Value{"key": cty.StringVal("value")}),
		},
		{
			name:     "non-sensitive to sensitive empty list",
			input:    cty.ListValEmpty(cty.String),
			expected: cty.ListValEmpty(cty.String).Mark(marks.Sensitive),
		},
		{
			name:     "sensitive to non-sensitive empty map",
			input:    cty.MapValEmpty(cty.String).Mark(marks.Sensitive),
			expected: cty.MapValEmpty(cty.String),
		},
		{
			name:     "non-sensitive to sensitive null",
			input:    cty.NullVal(cty.String),
			expected: cty.NullVal(cty.String).Mark(marks.Sensitive),
		},
		{
			name:     "sensitive to non-sensitive null",
			input:    cty.NullVal(cty.String).Mark(marks.Sensitive),
			expected: cty.NullVal(cty.String),
		},
		{
			name:     "non-sensitive to sensitive unknown",
			input:    cty.UnknownVal(cty.String),
			expected: cty.UnknownVal(cty.String).Mark(marks.Sensitive),
		},
		{
			name:     "sensitive to non-sensitive unknown",
			input:    cty.UnknownVal(cty.String).Mark(marks.Sensitive),
			expected: cty.UnknownVal(cty.String),
		},
		{
			name:     "nested structure non-sensitive to sensitive",
			input:    cty.ObjectVal(map[string]cty.Value{"a": cty.ListVal([]cty.Value{cty.StringVal("nested")})}),
			expected: cty.ObjectVal(map[string]cty.Value{"a": cty.ListVal([]cty.Value{cty.StringVal("nested")})}).Mark(marks.Sensitive),
		},
		{
			name:     "nested structure sensitive to non-sensitive",
			input:    cty.ObjectVal(map[string]cty.Value{"a": cty.ListVal([]cty.Value{cty.StringVal("nested")})}).Mark(marks.Sensitive),
			expected: cty.ObjectVal(map[string]cty.Value{"a": cty.ListVal([]cty.Value{cty.StringVal("nested")})}),
		},
		{
			name:     "extreme number value non-sensitive to sensitive",
			input:    cty.NumberFloatVal(1e100),
			expected: cty.NumberFloatVal(1e100).Mark(marks.Sensitive),
		},
		{
			name:     "empty string non-sensitive to sensitive",
			input:    cty.StringVal(""),
			expected: cty.StringVal("").Mark(marks.Sensitive),
		},
		{
			name:     "preserve other marks when making sensitive",
			input:    cty.StringVal("marked").Mark("custom"),
			expected: cty.StringVal("marked").Mark("custom").Mark(marks.Sensitive),
		},
		{
			name:     "preserve other marks when making non-sensitive",
			input:    cty.StringVal("marked").Mark("custom").Mark(marks.Sensitive),
			expected: cty.StringVal("marked").Mark("custom"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FlipSensitive(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(tt.expected) {
				t.Errorf("FlipSensitive() = %#v, want %#v", got, tt.expected)
			}

			// Check sensitivity
			if marks.Has(got, marks.Sensitive) != marks.Has(tt.expected, marks.Sensitive) {
				t.Errorf("FlipSensitive() sensitivity mismatch: got %v, want %v", 
					marks.Has(got, marks.Sensitive), marks.Has(tt.expected, marks.Sensitive))
			}
		})
	}
}

func TestFlipSensitiveWithMarks(t *testing.T) {
	const customMark = "custom"

	tests := []struct {
		name     string
		input    cty.Value
		expected cty.Value
		checkFn  func(t *testing.T, got cty.Value)
	}{
		{
			name:     "non-sensitive to sensitive",
			input:    cty.StringVal("hello"),
			expected: cty.StringVal("hello").Mark(marks.Sensitive),
			checkFn: func(t *testing.T, got cty.Value) {
				if !marks.Has(got, marks.Sensitive) {
					t.Errorf("FlipSensitive() result is not sensitive")
				}
			},
		},
		{
			name:     "sensitive to non-sensitive",
			input:    cty.StringVal("secret").Mark(marks.Sensitive),
			expected: cty.StringVal("secret"),
			checkFn: func(t *testing.T, got cty.Value) {
				if marks.Has(got, marks.Sensitive) {
					t.Errorf("FlipSensitive() result is unexpectedly sensitive")
				}
			},
		},
		{
			name:     "preserve custom mark when flipping to non-sensitive",
			input:    cty.StringVal("multi-marked").Mark(customMark).Mark(marks.Sensitive),
			expected: cty.StringVal("multi-marked").Mark(customMark),
			checkFn: func(t *testing.T, got cty.Value) {
				if marks.Has(got, marks.Sensitive) {
					t.Errorf("FlipSensitive() result is unexpectedly sensitive")
				}
				if !got.HasMark(customMark) {
					t.Errorf("FlipSensitive() did not preserve custom mark")
				}
			},
		},
		{
			name:     "preserve custom mark when flipping to sensitive",
			input:    cty.StringVal("multi-marked").Mark(customMark),
			expected: cty.StringVal("multi-marked").Mark(customMark).Mark(marks.Sensitive),
			checkFn: func(t *testing.T, got cty.Value) {
				if !marks.Has(got, marks.Sensitive) {
					t.Errorf("FlipSensitive() result is not sensitive")
				}
				if !got.HasMark(customMark) {
					t.Errorf("FlipSensitive() did not preserve custom mark")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FlipSensitive(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(tt.expected) {
				t.Errorf("FlipSensitive() = %#v, want %#v", got, tt.expected)
			}

			if tt.checkFn != nil {
				tt.checkFn(t, got)
			}
		})
	}
}

