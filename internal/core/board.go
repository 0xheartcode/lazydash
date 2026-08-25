package core

import "strings"

// GroupByField buckets RawItems into Columns using the named single-select field.
// Column order follows the field's option order. Falls back to "No <field>" bucket.
func GroupByField(data *BoardData, fieldName string) []Column {
	// Find the field's ordered options.
	var options []string
	for _, f := range data.Fields {
		if f.Name == fieldName {
			for _, o := range f.Options {
				options = append(options, o.Name)
			}
			break
		}
	}
	noGroup := "No " + fieldName
	if len(options) == 0 {
		options = []string{noGroup}
	}

	buckets := make(map[string][]Card, len(options)+1)
	for _, opt := range options {
		buckets[opt] = nil
	}

	for _, item := range data.Items {
		if item.IsArchived {
			continue
		}
		card := Card{
			ID:          item.ID,
			Type:        item.Type,
			Number:      item.Number,
			Title:       item.Title,
			State:       item.State,
			URL:         item.URL,
			Repo:        item.Repo,
			Assignees:   item.Assignees,
			Labels:      item.Labels,
			Body:        item.Body,
			FieldValues: item.FieldValues,
		}

		val := item.FieldValues[fieldName]
		if val == "" {
			val = noGroup
		}
		card.Status = val

		if _, ok := buckets[val]; ok {
			buckets[val] = append(buckets[val], card)
		} else {
			buckets[noGroup] = append(buckets[noGroup], card)
		}
	}

	columns := make([]Column, 0, len(options))
	for _, opt := range options {
		columns = append(columns, Column{Name: opt, Cards: buckets[opt]})
	}
	// Append the no-group bucket only if it has cards.
	if cards := buckets[noGroup]; len(cards) > 0 {
		// Avoid duplicate if noGroup was already an option.
		found := false
		for _, opt := range options {
			if opt == noGroup {
				found = true
				break
			}
		}
		if !found {
			columns = append(columns, Column{Name: noGroup, Cards: cards})
		}
	}
	return columns
}

// FlatItems returns all non-archived items in source order, for table views.
func FlatItems(data *BoardData) []Card {
	var cards []Card
	for _, item := range data.Items {
		if item.IsArchived {
			continue
		}
		cards = append(cards, Card{
			ID:          item.ID,
			Type:        item.Type,
			Number:      item.Number,
			Title:       item.Title,
			State:       item.State,
			URL:         item.URL,
			Repo:        item.Repo,
			Assignees:   item.Assignees,
			Labels:      item.Labels,
			Body:        item.Body,
			Status:      item.FieldValues["Status"],
			FieldValues: item.FieldValues,
		})
	}
	return cards
}

// OptionColors builds a lookup map of fieldName → optionName → terminal color string
// derived from each option's color enum. Used by the board renderer.
func OptionColors(data *BoardData) map[string]map[string]string {
	result := make(map[string]map[string]string, len(data.Fields))
	for _, f := range data.Fields {
		m := make(map[string]string, len(f.Options))
		for _, opt := range f.Options {
			m[opt.Name] = colorToTerminal(opt.Color)
		}
		result[f.Name] = m
	}
	return result
}

// colorToTerminal maps a single-select color enum to a 256-color terminal code.
func colorToTerminal(color string) string {
	switch strings.ToUpper(color) {
	case "GRAY":
		return "241"
	case "BLUE":
		return "39"
	case "GREEN":
		return "76"
	case "YELLOW":
		return "227"
	case "ORANGE":
		return "214"
	case "RED":
		return "196"
	case "PINK":
		return "212"
	case "PURPLE":
		return "99"
	default:
		return "241"
	}
}
