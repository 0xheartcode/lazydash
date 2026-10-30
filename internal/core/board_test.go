package core

import "testing"

func sampleBoard() *BoardData {
	return &BoardData{
		Fields: []SelectField{{
			Name:    "Status",
			Options: []FieldOption{{Name: "Todo", Color: "GRAY"}, {Name: "Done", Color: "GREEN"}},
		}},
		Items: []RawItem{
			{ID: "1", Title: "A", FieldValues: map[string]string{"Status": "Todo"}},
			{ID: "2", Title: "B", FieldValues: map[string]string{"Status": "Done"}},
			{ID: "3", Title: "C", FieldValues: map[string]string{}}, // ungrouped
			{ID: "4", Title: "arch", IsArchived: true, FieldValues: map[string]string{"Status": "Todo"}},
		},
	}
}

func TestGroupByFieldOrdersColumnsAndBucketsUngrouped(t *testing.T) {
	cols := GroupByField(sampleBoard(), "Status")

	// Option order is preserved, with the fallback bucket appended.
	if len(cols) != 3 {
		t.Fatalf("columns = %d, want 3 (Todo, Done, No Status)", len(cols))
	}
	if cols[0].Name != "Todo" || cols[1].Name != "Done" || cols[2].Name != "No Status" {
		t.Fatalf("column order = %s, %s, %s", cols[0].Name, cols[1].Name, cols[2].Name)
	}
	// Archived items are excluded, so Todo holds only A.
	if len(cols[0].Cards) != 1 || cols[0].Cards[0].Title != "A" {
		t.Errorf("Todo cards = %+v, want [A]", cols[0].Cards)
	}
	// The item with no Status lands in the fallback bucket.
	if len(cols[2].Cards) != 1 || cols[2].Cards[0].Title != "C" {
		t.Errorf("No Status cards = %+v, want [C]", cols[2].Cards)
	}
	// Card.Status is stamped with the group value.
	if cols[1].Cards[0].Status != "Done" {
		t.Errorf("card Status = %q, want Done", cols[1].Cards[0].Status)
	}
}

func TestFlatItemsExcludesArchived(t *testing.T) {
	cards := FlatItems(sampleBoard())
	if len(cards) != 3 {
		t.Fatalf("flat items = %d, want 3 (archived excluded)", len(cards))
	}
}

func TestOptionColorsMapsEnumsToTerminalCodes(t *testing.T) {
	colors := OptionColors(sampleBoard())
	if got := colors["Status"]["Done"]; got != "76" { // GREEN
		t.Errorf("Done color = %q, want 76", got)
	}
	if got := colors["Status"]["Todo"]; got != "241" { // GRAY
		t.Errorf("Todo color = %q, want 241", got)
	}
}

func TestColorToTerminalUnknownFallsBackToGray(t *testing.T) {
	if got := colorToTerminal("MAGENTA"); got != "241" {
		t.Errorf("unknown color = %q, want 241 (gray fallback)", got)
	}
	if got := colorToTerminal("red"); got != "196" { // case-insensitive
		t.Errorf("red = %q, want 196", got)
	}
}

// TestColorToTerminalAllCases asserts the exact terminal code for every switch
// branch (including the default), plus case-insensitive handling of the input.
func TestColorToTerminalAllCases(t *testing.T) {
	cases := []struct {
		name  string
		color string
		want  string
	}{
		{"gray", "GRAY", "241"},
		{"blue", "BLUE", "39"},
		{"green", "GREEN", "76"},
		{"yellow", "YELLOW", "227"},
		{"orange", "ORANGE", "214"},
		{"red", "RED", "196"},
		{"pink", "PINK", "212"},
		{"purple", "PURPLE", "99"},
		{"default unknown", "TEAL", "241"},
		{"default empty", "", "241"},
		{"lowercase blue", "blue", "39"},
		{"mixed case green", "GrEeN", "76"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := colorToTerminal(tc.color); got != tc.want {
				t.Errorf("colorToTerminal(%q) = %q, want %q", tc.color, got, tc.want)
			}
		})
	}
}

// TestOptionColorsNestedMapAllColors verifies OptionColors builds the nested
// fieldName to optionName to terminal code map across several color enums.
func TestOptionColorsNestedMapAllColors(t *testing.T) {
	data := &BoardData{
		Fields: []SelectField{{
			Name: "Priority",
			Options: []FieldOption{
				{Name: "Low", Color: "BLUE"},
				{Name: "Medium", Color: "YELLOW"},
				{Name: "High", Color: "RED"},
				{Name: "Unset", Color: "WHATEVER"},
			},
		}},
	}
	colors := OptionColors(data)
	want := map[string]string{
		"Low":    "39",
		"Medium": "227",
		"High":   "196",
		"Unset":  "241",
	}
	for opt, code := range want {
		if got := colors["Priority"][opt]; got != code {
			t.Errorf("Priority[%q] = %q, want %q", opt, got, code)
		}
	}
}
