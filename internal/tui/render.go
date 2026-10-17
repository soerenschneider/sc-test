package tui

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
)

var (
	keyStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))  // Blue
	warnStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("202")) // Orange/red
	valueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))            // Gray
)

func PrintMapOutput(data map[string]any, problematicColumns ...string) {
	// Get and sort keys
	keys := make([]string, 0, len(data))
	maxKeyLen := 0
	for k := range data {
		keys = append(keys, k)
		if len(k) > maxKeyLen {
			maxKeyLen = len(k)
		}
	}
	sort.Strings(keys)

	// Build output
	indent := 2
	var builder strings.Builder
	for _, k := range keys {
		v := data[k]
		paddedKey := fmt.Sprintf("%-*s", maxKeyLen, k)

		var key string
		if slices.Contains(problematicColumns, k) {
			key = warnStyle.Render(paddedKey)
		} else {
			key = keyStyle.Render(paddedKey)
		}

		val := valueStyle.Render(fmt.Sprintf("%v", v))
		spaces := strings.Repeat(" ", indent)
		_, _ = fmt.Fprintf(&builder, "%s%s%s\n", key, spaces, val)
	}

	rendered := builder.String()
	fmt.Println(rendered)
	//rendered = strings.ReplaceAll(rendered, "_", "\\_")
}

func WriteListOutput[T any](items []T) {
	itemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")) // Gray

	var builder strings.Builder
	for _, item := range items {
		builder.WriteString(itemStyle.Render(fmt.Sprintf("- %v", item)) + "\n")
	}
	fmt.Println(builder.String())
}
