// Package gohistogram contains implementations of weighted and exponential histograms.
package gohistogram

// Copyright (c) 2013 VividCortex, Inc. All rights reserved.
// Please see the LICENSE file for applicable license terms.

import "fmt"

// A WeightedHistogram implements Histogram. A WeightedHistogram has bins that have values
// which are exponentially weighted moving averages. This allows you keep inserting large
// amounts of data into the histogram and approximate quantiles with recency factored in.
type WeightedHistogram struct {
	Bins    []Bin
	Maxbins int
	Total   float64
	Alpha   float64
}

// NewWeightedHistogram returns a new WeightedHistogram with a maximum of n bins with a decay factor
// of alpha.
//
// There is no "optimal" bin count, but somewhere between 20 and 80 bins should be
// sufficient.
//
// Alpha should be set to 2 / (N+1), where N represents the average age of the moving window.
// For example, a 60-second window with an average age of 30 seconds would yield an
// alpha of 0.064516129.
func NewWeightedHistogram(n int, alpha float64) *WeightedHistogram {
	return &WeightedHistogram{
		Bins:    make([]Bin, 0),
		Maxbins: n,
		Total:   0,
		Alpha:   alpha,
	}
}

func ewma(existingVal float64, newVal float64, alpha float64) (result float64) {
	result = newVal*(1-alpha) + existingVal*alpha
	return
}

func (h *WeightedHistogram) scaleDown(except int) {
	for i := range h.Bins {
		if i != except {
			h.Bins[i].Count = ewma(h.Bins[i].Count, 0, h.Alpha)
		}
	}
}

func (h *WeightedHistogram) Add(n float64) {
	defer h.trim()
	for i := range h.Bins {
		if h.Bins[i].Value == n {
			h.Bins[i].Count++

			defer h.scaleDown(i)
			return
		}

		if h.Bins[i].Value > n {

			newbin := Bin{Value: n, Count: 1}
			head := append(make([]Bin, 0), h.Bins[0:i]...)

			head = append(head, newbin)
			tail := h.Bins[i:]
			h.Bins = append(head, tail...)

			defer h.scaleDown(i)
			return
		}
	}

	h.Bins = append(h.Bins, Bin{Count: 1, Value: n})
}

func (h *WeightedHistogram) Quantile(q float64) float64 {
	count := q * h.Total
	for i := range h.Bins {
		count -= float64(h.Bins[i].Count)

		if count <= 0 {
			return h.Bins[i].Value
		}
	}

	return -1
}

// CDF returns the value of the cumulative distribution function
// at x
func (h *WeightedHistogram) CDF(x float64) float64 {
	count := 0.0
	for i := range h.Bins {
		if h.Bins[i].Value <= x {
			count += float64(h.Bins[i].Count)
		}
	}

	return count / h.Total
}

// Mean returns the sample mean of the distribution
func (h *WeightedHistogram) Mean() float64 {
	if h.Total == 0 {
		return 0
	}

	sum := 0.0

	for i := range h.Bins {
		sum += h.Bins[i].Value * h.Bins[i].Count
	}

	return sum / h.Total
}

// Variance returns the variance of the distribution
func (h *WeightedHistogram) Variance() float64 {
	if h.Total == 0 {
		return 0
	}

	sum := 0.0
	mean := h.Mean()

	for i := range h.Bins {
		sum += (h.Bins[i].Count * (h.Bins[i].Value - mean) * (h.Bins[i].Value - mean))
	}

	return sum / h.Total
}

func (h *WeightedHistogram) Count() float64 {
	return h.Total
}

func (h *WeightedHistogram) trim() {
	total := 0.0
	for i := range h.Bins {
		total += h.Bins[i].Count
	}
	h.Total = total
	for len(h.Bins) > h.Maxbins {

		// Find closest bins in terms of value
		minDelta := 1e99
		minDeltaIndex := 0
		for i := range h.Bins {
			if i == 0 {
				continue
			}

			if delta := h.Bins[i].Value - h.Bins[i-1].Value; delta < minDelta {
				minDelta = delta
				minDeltaIndex = i
			}
		}

		// We need to merge bins minDeltaIndex-1 and minDeltaIndex
		totalCount := h.Bins[minDeltaIndex-1].Count + h.Bins[minDeltaIndex].Count
		mergedbin := Bin{
			Value: (h.Bins[minDeltaIndex-1].Value*
				h.Bins[minDeltaIndex-1].Count +
				h.Bins[minDeltaIndex].Value*
					h.Bins[minDeltaIndex].Count) /
				totalCount, // weighted average
			Count: totalCount, // summed heights
		}
		head := append(make([]Bin, 0), h.Bins[0:minDeltaIndex-1]...)
		tail := append([]Bin{mergedbin}, h.Bins[minDeltaIndex+1:]...)
		h.Bins = append(head, tail...)
	}
}

// String returns a string reprentation of the histogram,
// which is useful for printing to a terminal.
func (h *WeightedHistogram) String() (str string) {
	str += fmt.Sprintln("Total:", h.Total)

	for i := range h.Bins {
		var bar string
		for j := 0; j < int(float64(h.Bins[i].Count)/float64(h.Total)*200); j++ {
			bar += "."
		}
		str += fmt.Sprintln(h.Bins[i].Value, "\t", bar)
	}

	return
}
