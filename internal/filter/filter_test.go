package filter

import "testing"

func TestFilterAllowed(t *testing.T) {
	tests := []struct {
		name       string
		include    []string
		exclude    []string
		categories []string
		want       bool
	}{
		{
			name:       "disabled filter allows everything",
			categories: []string{"wishlist"},
			want:       true,
		},
		{
			name: "disabled filter allows entries without categories",
			want: true,
		},
		{
			name:       "exact exclusion match",
			exclude:    []string{"wishlist"},
			categories: []string{"wishlist"},
			want:       false,
		},
		{
			name:       "exclusion match is case-insensitive",
			exclude:    []string{"wishlist"},
			categories: []string{"Wishlist"},
			want:       false,
		},
		{
			name:       "exclusion does not match substrings",
			exclude:    []string{"wishlist"},
			categories: []string{"wishlist-adjacent"},
			want:       true,
		},
		{
			name:       "exclusion does not match when the entry category is longer",
			exclude:    []string{"wish"},
			categories: []string{"wishlist"},
			want:       true,
		},
		{
			name:       "entry without categories is allowed when only exclude is set",
			exclude:    []string{"wishlist"},
			categories: nil,
			want:       true,
		},
		{
			name:       "entry with unrelated categories is allowed",
			exclude:    []string{"wishlist"},
			categories: []string{"golang", "links"},
			want:       true,
		},
		{
			name:       "include requires a match",
			include:    []string{"golang"},
			categories: []string{"links"},
			want:       false,
		},
		{
			name:       "include matches case-insensitively",
			include:    []string{"golang"},
			categories: []string{"GoLang"},
			want:       true,
		},
		{
			name:    "entry without categories is not allowed when include is set",
			include: []string{"golang"},
			want:    false,
		},
		{
			name:       "exclude wins over include",
			include:    []string{"wishlist"},
			exclude:    []string{"wishlist"},
			categories: []string{"wishlist"},
			want:       false,
		},
		{
			name:       "include matches while exclude does not",
			include:    []string{"golang"},
			exclude:    []string{"wishlist"},
			categories: []string{"golang", "links"},
			want:       true,
		},
		{
			name:       "category whitespace is ignored",
			exclude:    []string{"wishlist"},
			categories: []string{"  wishlist  "},
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := New(tt.include, tt.exclude)
			if got := f.Allowed(tt.categories); got != tt.want {
				t.Errorf("Allowed(%v) = %v, want %v", tt.categories, got, tt.want)
			}
		})
	}
}

func TestFilterEnabled(t *testing.T) {
	if New(nil, nil).Enabled() {
		t.Error("empty filter should be disabled")
	}

	if New([]string{"  "}, []string{""}).Enabled() {
		t.Error("filter with only blank values should be disabled")
	}

	if !New(nil, []string{"wishlist"}).Enabled() {
		t.Error("exclude-only filter should be enabled")
	}

	if !New([]string{"links"}, nil).Enabled() {
		t.Error("include-only filter should be enabled")
	}
}

func TestFilterNormalizesValues(t *testing.T) {
	f := New(
		[]string{" Golang ", "golang", "", "News"},
		[]string{"Wishlist", " wishlist ", " "},
	)

	if got, want := len(f.include), 2; got != want {
		t.Fatalf("include length = %d, want %d (%v)", got, want, f.include)
	}
	if f.include[0] != "Golang" || f.include[1] != "News" {
		t.Errorf("include = %v, want [Golang News]", f.include)
	}

	if got, want := len(f.exclude), 1; got != want {
		t.Fatalf("exclude length = %d, want %d (%v)", got, want, f.exclude)
	}
	if f.exclude[0] != "Wishlist" {
		t.Errorf("exclude = %v, want [Wishlist]", f.exclude)
	}
}
