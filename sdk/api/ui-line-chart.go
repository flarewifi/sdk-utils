/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

// LineChartDataPoint represents a single data point on the chart
type LineChartDataPoint struct {
	Label  string             `json:"label"`
	Values map[string]float64 `json:"values"`
}

// LineChartSeries defines a data series in the chart
type LineChartSeries struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Color string `json:"color"`
}

// LineChartYAxis configures the Y-axis
type LineChartYAxis struct {
	Min      *float64 `json:"min,omitempty"`
	Max      *float64 `json:"max,omitempty"`
	StepSize *float64 `json:"stepSize,omitempty"`
}

// LineChartPadding configures chart padding
type LineChartPadding struct {
	Top    int `json:"top,omitempty"`
	Right  int `json:"right,omitempty"`
	Bottom int `json:"bottom,omitempty"`
	Left   int `json:"left,omitempty"`
}

// LineChartOpts configures the line chart component
type LineChartOpts struct {
	ID              string               // HTML element ID (auto-generated if empty)
	Data            []LineChartDataPoint // Chart data points
	Series          []LineChartSeries    // Series definitions
	Tension         *float64             // Curve smoothness (0-1, default 0.4)
	FillOpacity     []float64            // Gradient opacity [top, bottom]
	Padding         *LineChartPadding    // Chart padding
	YAxis           *LineChartYAxis      // Y-axis configuration
	Height          string               // Container height (default "350px")
	Class           string               // Additional CSS classes
	TooltipTemplate string               // Tooltip format, e.g., "{label}: {value} MB"
	TooltipPrefix   string               // Prefix before value, e.g., "₱"
	TooltipDecimals *int                 // Decimal places (default: 2)
	IsStacked       *bool
}
