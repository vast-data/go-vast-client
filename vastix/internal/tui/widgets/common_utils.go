package widgets

import (
	"vastix/internal/colors"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"vastix/internal/database"

	"github.com/charmbracelet/lipgloss"
	vast_client "github.com/vast-data/go-vast-client"
)

func getActiveRest(db *database.Service) (*vast_client.VMSRest, error) {
	profile, err := db.GetActiveProfile()
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, fmt.Errorf("no active profile found")
	}
	return profile.RestClientFromProfile()
}

// formatRecordAsJSON converts a map[string]any record into JSON-style formatted string with syntax highlighting
func formatRecordAsJSON(record map[string]any) string {
	delete(record, "@resourceType")
	var details strings.Builder

	// Define colors for syntax highlighting (balanced brightness)
	keyColor := lipgloss.NewStyle().Foreground(colors.MediumCyan)      // Medium cyan for keys
	stringColor := lipgloss.NewStyle().Foreground(colors.MediumGreen)   // Medium green for strings
	numberColor := lipgloss.NewStyle().Foreground(colors.MutedOrange)  // Muted orange for numbers
	boolColor := lipgloss.NewStyle().Foreground(colors.MediumPurple)    // Medium purple for booleans
	nullColor := lipgloss.NewStyle().Foreground(colors.MediumGrey)    // Gray for null values
	bracketColor := lipgloss.NewStyle().Foreground(colors.VeryLightGrey) // Light white for brackets/punctuation

	// Left margin (2 spaces)
	leftMargin := "  "

	// Start JSON object
	details.WriteString(leftMargin + bracketColor.Render("{\n"))

	// Helper function to check if a string is a JSON array
	isJSONArray := func(s string) bool {
		s = strings.TrimSpace(s)
		return strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]")
	}

	// Helper function to check if a string is a JSON object
	isJSONObject := func(s string) bool {
		s = strings.TrimSpace(s)
		return strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")
	}

	// Helper function to format JSON object with proper indentation
	formatObject := func(objStr string, nestLevel int) string {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(objStr), &obj); err != nil {
			// If parsing fails, treat as regular string
			return stringColor.Render(fmt.Sprintf("\"%s\"", objStr))
		}

		if len(obj) == 0 {
			return bracketColor.Render("{}")
		}

		// Calculate indentation for nested objects
		baseIndent := strings.Repeat("  ", nestLevel+1) // Base indentation
		fieldIndent := baseIndent + "  "                // Field indentation (extra 2 spaces)

		result := bracketColor.Render("{\n")

		// Get keys and sort them for consistent output
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// Format each field
		for i, key := range keys {
			isLast := i == len(keys)-1
			keyPart := keyColor.Render(fmt.Sprintf("\"%s\"", key))
			var valuePart string

			switch v := obj[key].(type) {
			case string:
				valuePart = stringColor.Render(fmt.Sprintf("\"%s\"", v))
			case float64:
				if math.Mod(v, 1) == 0 {
					valuePart = numberColor.Render(fmt.Sprintf("%.0f", v))
				} else {
					valuePart = numberColor.Render(fmt.Sprintf("%.2f", v))
				}
			case bool:
				valuePart = boolColor.Render(fmt.Sprintf("%t", v))
			case nil:
				valuePart = nullColor.Render("null")
			case map[string]interface{}:
				// Format nested objects with proper indentation
				// Get keys and sort them for consistent output
				nestedKeys := make([]string, 0, len(v))
				for k := range v {
					nestedKeys = append(nestedKeys, k)
				}
				sort.Strings(nestedKeys)

				result := bracketColor.Render("{\n")
				for i, nestedKey := range nestedKeys {
					nestedValue := v[nestedKey]
					nestedKeyPart := keyColor.Render(fmt.Sprintf("\"%s\"", nestedKey))
					var nestedValuePart string
					switch nv := nestedValue.(type) {
					case string:
						nestedValuePart = stringColor.Render(fmt.Sprintf("\"%s\"", nv))
					case float64:
						if math.Mod(nv, 1) == 0 {
							nestedValuePart = numberColor.Render(fmt.Sprintf("%.0f", nv))
						} else {
							nestedValuePart = numberColor.Render(fmt.Sprintf("%.2f", nv))
						}
					case bool:
						nestedValuePart = boolColor.Render(fmt.Sprintf("%t", nv))
					case nil:
						nestedValuePart = nullColor.Render("null")
					default:
						nestedValuePart = stringColor.Render(fmt.Sprintf("\"%v\"", nv))
					}

					comma := ""
					if i < len(nestedKeys)-1 {
						comma = bracketColor.Render(",")
					}

					result += fieldIndent + nestedKeyPart + bracketColor.Render(": ") + nestedValuePart + comma + "\n"
				}
				result += fieldIndent + bracketColor.Render("}")
				valuePart = result
			case []interface{}:
				// Handle nested arrays - format inline for simplicity
				if len(v) == 0 {
					valuePart = bracketColor.Render("[]")
				} else {
					var arrayItems []string
					for _, item := range v {
						switch av := item.(type) {
						case string:
							arrayItems = append(arrayItems, stringColor.Render(fmt.Sprintf("\"%s\"", av)))
						case float64:
							if math.Mod(av, 1) == 0 {
								arrayItems = append(arrayItems, numberColor.Render(fmt.Sprintf("%.0f", av)))
							} else {
								arrayItems = append(arrayItems, numberColor.Render(fmt.Sprintf("%.2f", av)))
							}
						case bool:
							arrayItems = append(arrayItems, boolColor.Render(fmt.Sprintf("%t", av)))
						case nil:
							arrayItems = append(arrayItems, nullColor.Render("null"))
						default:
							arrayItems = append(arrayItems, stringColor.Render(fmt.Sprintf("\"%v\"", av)))
						}
					}
					valuePart = bracketColor.Render("[") + strings.Join(arrayItems, bracketColor.Render(", ")) + bracketColor.Render("]")
				}
			default:
				valuePart = stringColor.Render(fmt.Sprintf("\"%v\"", v))
			}

			comma := ""
			if !isLast {
				comma = bracketColor.Render(",")
			}

			result += fieldIndent + keyPart + bracketColor.Render(": ") + valuePart + comma + "\n"
		}

		result += baseIndent + bracketColor.Render("}")
		return result
	}

	// Helper function to format JSON array
	formatArray := func(arrayStr string) string {
		var items []interface{}
		if err := json.Unmarshal([]byte(arrayStr), &items); err != nil {
			// If parsing fails, treat as regular string
			return stringColor.Render(fmt.Sprintf("\"%s\"", arrayStr))
		}

		if len(items) == 0 {
			return bracketColor.Render("[]")
		}

		// Format as inline array for short arrays, multiline for long ones
		if len(items) <= 3 {
			// Inline format: ["item1", "item2"]
			var formattedItems []string
			for _, item := range items {
				switch v := item.(type) {
				case string:
					formattedItems = append(formattedItems, stringColor.Render(fmt.Sprintf("\"%s\"", v)))
				case float64:
					if math.Mod(v, 1) == 0 {
						formattedItems = append(formattedItems, numberColor.Render(fmt.Sprintf("%.0f", v)))
					} else {
						formattedItems = append(formattedItems, numberColor.Render(fmt.Sprintf("%.2f", v)))
					}
				case bool:
					formattedItems = append(formattedItems, boolColor.Render(fmt.Sprintf("%t", v)))
				case nil:
					formattedItems = append(formattedItems, nullColor.Render("null"))
				default:
					formattedItems = append(formattedItems, stringColor.Render(fmt.Sprintf("\"%v\"", v)))
				}
			}
			return bracketColor.Render("[") + strings.Join(formattedItems, bracketColor.Render(", ")) + bracketColor.Render("]")
		} else {
			// Multiline format for long arrays
			result := bracketColor.Render("[\n")
			for i, item := range items {
				isLast := i == len(items)-1
				indent := leftMargin + "    "

				var itemStr string
				switch v := item.(type) {
				case string:
					itemStr = stringColor.Render(fmt.Sprintf("\"%s\"", v))
				case float64:
					if math.Mod(v, 1) == 0 {
						itemStr = numberColor.Render(fmt.Sprintf("%.0f", v))
					} else {
						itemStr = numberColor.Render(fmt.Sprintf("%.2f", v))
					}
				case bool:
					itemStr = boolColor.Render(fmt.Sprintf("%t", v))
				case nil:
					itemStr = nullColor.Render("null")
				default:
					itemStr = stringColor.Render(fmt.Sprintf("\"%v\"", v))
				}

				comma := ""
				if !isLast {
					comma = bracketColor.Render(",")
				}
				result += indent + itemStr + comma + "\n"
			}
			result += leftMargin + "  " + bracketColor.Render("]")
			return result
		}
	}

	// Helper function to add a field
	addField := func(key string, value interface{}, isLast bool) {
		indent := leftMargin
		keyStr := keyColor.Render(fmt.Sprintf("\"%s\"", key))
		colon := bracketColor.Render(": ")

		var valueStr string
		switch v := value.(type) {
		case string:
			if v == "" {
				valueStr = nullColor.Render("null")
			} else if isJSONArray(v) {
				// Handle JSON array strings
				valueStr = formatArray(v)
			} else if isJSONObject(v) {
				// Handle JSON object strings
				valueStr = formatObject(v, 1)
			} else {
				valueStr = stringColor.Render(fmt.Sprintf("\"%s\"", v))
			}
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			valueStr = numberColor.Render(fmt.Sprintf("%v", v))
		case float32:
			// Check if it's actually an integer value
			if math.Mod(float64(v), 1) == 0 {
				valueStr = numberColor.Render(fmt.Sprintf("%.0f", v))
			} else {
				valueStr = numberColor.Render(fmt.Sprintf("%.2f", v))
			}
		case float64:
			// Check if it's actually an integer value
			if math.Mod(v, 1) == 0 {
				valueStr = numberColor.Render(fmt.Sprintf("%.0f", v))
			} else {
				valueStr = numberColor.Render(fmt.Sprintf("%.2f", v))
			}
		case bool:
			valueStr = boolColor.Render(fmt.Sprintf("%t", v))
		case nil:
			valueStr = nullColor.Render("null")
		case map[string]interface{}:
			// Handle nested maps properly
			if len(v) == 0 {
				valueStr = bracketColor.Render("{}")
			} else {
				// Convert map to JSON string and format it
				if jsonBytes, err := json.Marshal(v); err == nil {
					valueStr = formatObject(string(jsonBytes), 1)
				} else {
					valueStr = stringColor.Render(fmt.Sprintf("\"%v\"", v))
				}
			}
		case []interface{}:
			// Handle nested arrays properly
			if len(v) == 0 {
				valueStr = bracketColor.Render("[]")
			} else {
				// Convert array to JSON string and format it
				if jsonBytes, err := json.Marshal(v); err == nil {
					valueStr = formatArray(string(jsonBytes))
				} else {
					valueStr = stringColor.Render(fmt.Sprintf("\"%v\"", v))
				}
			}
		default:
			// For any other type, convert to string
			valueStr = stringColor.Render(fmt.Sprintf("\"%v\"", v))
		}

		comma := ""
		if !isLast {
			comma = bracketColor.Render(",")
		}

		details.WriteString(fmt.Sprintf("%s%s%s%s%s\n", indent, keyStr, colon, valueStr, comma))
	}

	// Get all keys and sort them for consistent output
	keys := make([]string, 0, len(record))
	for k := range record {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Render all fields
	for i, key := range keys {
		isLast := i == len(keys)-1
		addField(key, record[key], isLast)
	}

	// Close JSON object with left margin
	details.WriteString(leftMargin + bracketColor.Render("}"))

	return details.String()
}
