package reports

import (
	"encoding/json"
	"fmt"
	"html/template"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Renderer provides common utilities for rendering report data
type Renderer struct{}

// NewRenderer creates a new renderer instance
func NewRenderer() *Renderer {
	return &Renderer{}
}

// FormatCurrency formats a numeric value as currency
func (r *Renderer) FormatCurrency(value interface{}, currency string) string {
	var amount float64
	
	switch v := value.(type) {
	case float64:
		amount = v
	case float32:
		amount = float64(v)
	case int:
		amount = float64(v)
	case int64:
		amount = float64(v)
	case string:
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			amount = parsed
		}
	default:
		return fmt.Sprintf("%v", value)
	}

	symbol := getCurrencySymbol(currency)
	
	if amount >= 1000000 {
		return fmt.Sprintf("%s%.1fM", symbol, amount/1000000)
	} else if amount >= 1000 {
		return fmt.Sprintf("%s%.1fK", symbol, amount/1000)
	}
	
	return fmt.Sprintf("%s%.2f", symbol, amount)
}

// FormatPercentage formats a numeric value as a percentage
func (r *Renderer) FormatPercentage(value interface{}, decimals int) string {
	var amount float64
	
	switch v := value.(type) {
	case float64:
		amount = v
	case float32:
		amount = float64(v)
	case int:
		amount = float64(v)
	case int64:
		amount = float64(v)
	case string:
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			amount = parsed
		}
	default:
		return fmt.Sprintf("%v%%", value)
	}

	format := fmt.Sprintf("%%.%df%%%%", decimals)
	return fmt.Sprintf(format, amount)
}

// FormatDuration formats a duration in a human-readable way
func (r *Renderer) FormatDuration(duration time.Duration) string {
	if duration < time.Minute {
		return fmt.Sprintf("%.0fs", duration.Seconds())
	} else if duration < time.Hour {
		return fmt.Sprintf("%.0fm", duration.Minutes())
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%.1fh", duration.Hours())
	}
	return fmt.Sprintf("%.1fd", duration.Hours()/24)
}

// FormatNumber formats a large number with appropriate units
func (r *Renderer) FormatNumber(value interface{}) string {
	var amount float64
	
	switch v := value.(type) {
	case float64:
		amount = v
	case float32:
		amount = float64(v)
	case int:
		amount = float64(v)
	case int64:
		amount = float64(v)
	case string:
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			amount = parsed
		} else {
			return v
		}
	default:
		return fmt.Sprintf("%v", value)
	}

	if amount >= 1000000000 {
		return fmt.Sprintf("%.1fB", amount/1000000000)
	} else if amount >= 1000000 {
		return fmt.Sprintf("%.1fM", amount/1000000)
	} else if amount >= 1000 {
		return fmt.Sprintf("%.1fK", amount/1000)
	}
	
	return fmt.Sprintf("%.0f", amount)
}

// FormatTrend creates trend data from current and previous values
func (r *Renderer) FormatTrend(current, previous float64, period string) *TrendData {
	if previous == 0 {
		return &TrendData{
			Direction: TrendFlat,
			Value:     "N/A",
			Period:    period,
		}
	}

	change := ((current - previous) / previous) * 100
	var direction TrendDirection
	
	if change > 0.1 {
		direction = TrendUp
	} else if change < -0.1 {
		direction = TrendDown
	} else {
		direction = TrendFlat
	}

	value := fmt.Sprintf("%.1f%%", change)
	if change > 0 {
		value = "+" + value
	}

	return &TrendData{
		Direction: direction,
		Value:     value,
		Period:    period,
	}
}

// GenerateChartData converts data points into chart format
func (r *Renderer) GenerateChartData(title, chartType string, dataPoints []DataPoint, xField, yField string) ChartData {
	chart := ChartData{
		Title: title,
		Type:  chartType,
		XAxis: xField,
		YAxis: yField,
	}

	// Group data points by series if there are multiple y values
	seriesMap := make(map[string][]ChartPoint)
	
	for _, point := range dataPoints {
		xValue := r.extractValue(point, xField)
		
		if yField == "" {
			// Use all numeric values as separate series
			for key, val := range point.Values {
				if r.isNumeric(val) {
					seriesMap[key] = append(seriesMap[key], ChartPoint{
						X: xValue,
						Y: val,
					})
				}
			}
		} else {
			// Use specific y field
			yValue := r.extractValue(point, yField)
			seriesMap["data"] = append(seriesMap["data"], ChartPoint{
				X: xValue,
				Y: yValue,
			})
		}
	}

	// Convert map to slice and sort
	var seriesNames []string
	for name := range seriesMap {
		seriesNames = append(seriesNames, name)
	}
	sort.Strings(seriesNames)

	for _, name := range seriesNames {
		chart.Series = append(chart.Series, ChartSeries{
			Name: name,
			Data: seriesMap[name],
		})
	}

	return chart
}

// GenerateTableData converts data points into table format
func (r *Renderer) GenerateTableData(title string, dataPoints []DataPoint, columns []string) TableData {
	table := TableData{
		Title: title,
	}

	// Generate headers
	if len(columns) == 0 && len(dataPoints) > 0 {
		// Auto-detect columns from first data point
		for key := range dataPoints[0].Values {
			columns = append(columns, key)
		}
		for key := range dataPoints[0].Labels {
			columns = append(columns, key)
		}
		columns = append(columns, "timestamp")
	}

	sort.Strings(columns)

	for _, col := range columns {
		header := TableHeader{
			Key:        col,
			Label:      r.formatColumnName(col),
			Type:       "string",
			Sortable:   true,
			Filterable: true,
		}
		table.Headers = append(table.Headers, header)
	}

	// Generate rows
	for _, point := range dataPoints {
		row := make(map[string]interface{})
		
		for _, col := range columns {
			if col == "timestamp" {
				row[col] = point.Timestamp.Format("2006-01-02 15:04:05")
			} else if val, exists := point.Values[col]; exists {
				row[col] = val
			} else if val, exists := point.Labels[col]; exists {
				row[col] = val
			} else {
				row[col] = ""
			}
		}
		
		table.Rows = append(table.Rows, row)
	}

	return table
}

// CreateSummaryCard creates a basic summary implementation
func (r *Renderer) CreateSummaryCard(title, value, subtitle string, summaryType SummaryType, trend *TrendData) Summary {
	return &BasicSummary{
		title:       title,
		value:       value,
		subtitle:    subtitle,
		summaryType: summaryType,
		trend:       trend,
		healthy:     true,
	}
}

// ToJSON converts data to JSON string
func (r *Renderer) ToJSON(data interface{}) (string, error) {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ToHTML renders data as HTML table
func (r *Renderer) ToHTML(data TableData) (template.HTML, error) {
	var html strings.Builder
	
	html.WriteString(fmt.Sprintf("<table class=\"govuk-table\"><caption class=\"govuk-table__caption govuk-table__caption--m\">%s</caption>", data.Title))
	
	// Headers
	html.WriteString("<thead class=\"govuk-table__head\"><tr class=\"govuk-table__row\">")
	for _, header := range data.Headers {
		html.WriteString(fmt.Sprintf("<th scope=\"col\" class=\"govuk-table__header\">%s</th>", header.Label))
	}
	html.WriteString("</tr></thead>")
	
	// Rows
	html.WriteString("<tbody class=\"govuk-table__body\">")
	for _, row := range data.Rows {
		html.WriteString("<tr class=\"govuk-table__row\">")
		for _, header := range data.Headers {
			value := row[header.Key]
			html.WriteString(fmt.Sprintf("<td class=\"govuk-table__cell\">%v</td>", value))
		}
		html.WriteString("</tr>")
	}
	html.WriteString("</tbody></table>")
	
	return template.HTML(html.String()), nil
}

// Helper functions

func (r *Renderer) extractValue(point DataPoint, field string) interface{} {
	if field == "timestamp" {
		return point.Timestamp
	}
	if val, exists := point.Values[field]; exists {
		return val
	}
	if val, exists := point.Labels[field]; exists {
		return val
	}
	return nil
}

func (r *Renderer) isNumeric(value interface{}) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float32, float64:
		return true
	case string:
		_, err := strconv.ParseFloat(value.(string), 64)
		return err == nil
	}
	return false
}

func (r *Renderer) formatColumnName(name string) string {
	// Convert snake_case to Title Case
	words := strings.Split(name, "_")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

func getCurrencySymbol(currency string) string {
	switch strings.ToUpper(currency) {
	case "USD":
		return "$"
	case "EUR":
		return "€"
	case "GBP":
		return "£"
	case "JPY":
		return "¥"
	default:
		return currency + " "
	}
}

// BasicSummary provides a simple implementation of the Summary interface
type BasicSummary struct {
	title       string
	value       string
	subtitle    string
	summaryType SummaryType
	trend       *TrendData
	healthy     bool
}

func (s *BasicSummary) GetTitle() string       { return s.title }
func (s *BasicSummary) GetValue() string       { return s.value }
func (s *BasicSummary) GetSubtitle() string    { return s.subtitle }
func (s *BasicSummary) GetTrend() *TrendData   { return s.trend }
func (s *BasicSummary) GetType() SummaryType   { return s.summaryType }
func (s *BasicSummary) IsHealthy() bool        { return s.healthy }

// SetHealthy allows updating the health status
func (s *BasicSummary) SetHealthy(healthy bool) {
	s.healthy = healthy
}