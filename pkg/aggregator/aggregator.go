package aggregator

import (
	"fmt"
	"sync"
	"time"

	"github.com/tradestax/traedor/pkg/datafeed"
)

type OHLC struct {
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	Timestamp int64
	StartTime int64
	EndTime   int64
}

type TimeAggregator struct {
	intervalMinutes int
	currentBar      *OHLC
	currentBarStart time.Time // Store the actual start time of current bar
	mu              sync.Mutex
	barCallback     func(OHLC)
	tickCount       int // Add tick counter for debugging
}

func NewTimeAggregator(intervalMinutes int, callback func(OHLC)) *TimeAggregator {
	fmt.Printf("TimeAggregator: Creating %d-minute aggregator\n", intervalMinutes)
	return &TimeAggregator{
		intervalMinutes: intervalMinutes,
		barCallback:     callback,
	}
}

func (ta *TimeAggregator) ProcessTick(tick datafeed.Data) {
	ta.mu.Lock()
	defer ta.mu.Unlock()

	ta.tickCount++
	
	// Convert tick timestamp to time (assuming tick.Date is Unix seconds)
	tickTime := time.Unix(tick.Date, 0)
	
	// Debug: Log first few ticks and every 1000th tick to see timestamp progression
	if ta.tickCount <= 5 || ta.tickCount%1000 == 0 {
		fmt.Printf("TimeAggregator: Tick #%d - Raw timestamp: %d, Converted time: %v, Price: %.2f\n", 
			ta.tickCount, tick.Date, tickTime, tick.Close)
	}
	
	// Calculate the bar period this tick belongs to
	tickBarStart := ta.getBarStartTime(tickTime)
	
	// If no current bar or tick belongs to a new bar
	if ta.currentBar == nil {
		// Start the first bar
		ta.currentBarStart = tickBarStart
		fmt.Printf("TimeAggregator: Starting first %d-minute bar at %v (start: %v, end: %v)\n", 
			ta.intervalMinutes, tickTime, ta.currentBarStart, ta.currentBarStart.Add(time.Duration(ta.intervalMinutes) * time.Minute))
		ta.currentBar = &OHLC{
			Open:      tick.Close,
			High:      tick.Close,
			Low:       tick.Close,
			Close:     tick.Close,
			Volume:    tick.Volume,
			Timestamp: tick.Date,
			StartTime: ta.currentBarStart.UnixMilli(),
			EndTime:   ta.currentBarStart.Add(time.Duration(ta.intervalMinutes) * time.Minute).UnixMilli(),
		}
	} else {
		// Calculate end time of current bar
		currentBarEnd := ta.currentBarStart.Add(time.Duration(ta.intervalMinutes) * time.Minute)

		// If tick belongs to a new bar (beyond current bar's end time)
		if tickTime.Unix() >= currentBarEnd.Unix() {
			// Send completed bar
			fmt.Printf("TimeAggregator: Completing %d-minute bar - OHLC: %.2f/%.2f/%.2f/%.2f (start: %v, end: %v, tick: %v)\n", 
				ta.intervalMinutes, ta.currentBar.Open, ta.currentBar.High, ta.currentBar.Low, ta.currentBar.Close, 
				ta.currentBarStart, currentBarEnd, tickTime)
			ta.barCallback(*ta.currentBar)

			// Start new bar with the tick's bar start time
			ta.currentBarStart = tickBarStart
			newBarEnd := ta.currentBarStart.Add(time.Duration(ta.intervalMinutes) * time.Minute)
			fmt.Printf("TimeAggregator: Starting new %d-minute bar at %v (start: %v, end: %v)\n", 
				ta.intervalMinutes, tickTime, ta.currentBarStart, newBarEnd)
			ta.currentBar = &OHLC{
				Open:      tick.Close,
				High:      tick.Close,
				Low:       tick.Close,
				Close:     tick.Close,
				Volume:    tick.Volume,
				Timestamp: tick.Date,
				StartTime: ta.currentBarStart.UnixMilli(),
				EndTime:   ta.currentBarStart.Add(time.Duration(ta.intervalMinutes) * time.Minute).UnixMilli(),
			}
		} else {
			// Update current bar with this tick
			if tick.Close > ta.currentBar.High {
				ta.currentBar.High = tick.Close
			}
			if tick.Close < ta.currentBar.Low {
				ta.currentBar.Low = tick.Close
			}
			ta.currentBar.Close = tick.Close
			ta.currentBar.Volume += tick.Volume
			ta.currentBar.Timestamp = tick.Date
		}
	}
}

func (ta *TimeAggregator) getBarStartTime(t time.Time) time.Time {
	// Round down to the nearest interval aligned with hour boundaries
	// For 5-minute bars: :00, :05, :10, :15, :20, :25, :30, :35, :40, :45, :50, :55
	// For 30-minute bars: :00, :30
	
	// Get the time at the start of the current hour
	hourStart := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
	
	// Calculate minutes since hour start
	minutesSinceHour := t.Sub(hourStart).Minutes()
	
	// Round down to nearest interval within the hour
	intervalsSinceHour := int(minutesSinceHour) / ta.intervalMinutes
	alignedMinutes := intervalsSinceHour * ta.intervalMinutes
	
	// Return the aligned time
	return hourStart.Add(time.Duration(alignedMinutes) * time.Minute)
}

func (ta *TimeAggregator) Flush() {
	ta.mu.Lock()
	defer ta.mu.Unlock()

	if ta.currentBar != nil {
		ta.barCallback(*ta.currentBar)
		ta.currentBar = nil
	}
}

func (ta *TimeAggregator) Reset() {
	ta.mu.Lock()
	defer ta.mu.Unlock()

	ta.currentBar = nil
}