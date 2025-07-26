package broker

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/tradestax/traedor/pkg/broker/profit"
	"github.com/tradestax/traedor/pkg/broker/stop"
	"github.com/tradestax/traedor/pkg/types"
)

// ErrInsufficientBalance indicates the account doesn't have enough balance for a trade
// This is a normal end condition for backtesting, not an actual error
var ErrInsufficientBalance = errors.New("insufficient account balance")

type FuturesBroker struct {
	account       *Account
	blackoutTimes BlackoutTimes
	currentTrade  *Trade
	config        *Config
	currentPrice  float64
	currentTime   int64
	stopAmount    float64
	feePerSide    float64
	wins          int
	loses         int
	cumWin        float64
	cumLose       float64
	trades        []*Trade
	margin        float64
	pointPrice    float64
	output        string
	week          int
	balanceHistory []BalancePoint
	peakBalance    float64
	currentDrawdown float64
	maxDrawdown    float64
}

type BalancePoint struct {
	Time    time.Time
	Balance float64
}

func NewFuturesBroker(c *Config) IBroker {
	// set blackout (no trade) times
	tz, err := time.LoadLocation(c.BlackoutTimes.TimeZone)
	if err != nil {
		panic(fmt.Errorf("Failed to load location from config"))
	}
	startTime, err := time.ParseInLocation(timeLayout, c.BlackoutTimes.StartTime, tz)
	if err != nil {
		panic(fmt.Errorf("Failed to load blackout start time"))
	}
	endTime, err := time.ParseInLocation(timeLayout, c.BlackoutTimes.EndTime, tz)
	if err != nil {
		panic(fmt.Errorf("Failed to load blackout end time"))
	}
	fb := &FuturesBroker{
		account: &Account{
			balance:          c.StartingBalance,
			availableBalance: c.StartingBalance,
			weeklyWithdrawl:  c.WeeklyWithdrawl,
		},
		blackoutTimes: BlackoutTimes{
			StartTime: startTime,
			EndTime:   endTime,
			TimeZone:  tz,
		},
		config:     c,
		output:     "trades.csv",
		trades:     make([]*Trade, 0),
		margin:     c.Symbol.Margin,
		pointPrice: c.Symbol.PointPrice,
		stopAmount: c.TrailingStopAmount,
		feePerSide: c.FeePerSide,
		balanceHistory: make([]BalancePoint, 0),
		peakBalance:    c.StartingBalance,
		currentDrawdown: 0,
		maxDrawdown:    0,
	}
	
	// Initial balance will be recorded when first data arrives
	
	return fb
}

func (b *FuturesBroker) GetAccountStats() (*Account, error) {
	return b.account, nil
}

func (b *FuturesBroker) AddData(d types.Data) {
	b.currentPrice = d.Close
	// Ensure currentTime is always in milliseconds
	if d.Date < 1e12 { // If less than 1 trillion, it's likely in seconds
		b.currentTime = d.Date * 1000
	} else {
		b.currentTime = d.Date
	}
	
	// Record initial balance point on first data arrival
	if len(b.balanceHistory) == 0 {
		b.trackBalanceAndDrawdown()
	}
	
	if b.currentTrade != nil {
		// update max values for trade
		b.updateTradeValues(d)
		// check stops
		for _, stop := range b.currentTrade.Stops {
			if stop.Stop(b.currentPrice) {
				b.closePosition()
				return
			}
		}
		// check profits
		for _, profit := range b.currentTrade.Profits {
			if profit.Profit(b.currentPrice) {
				b.closePosition()
				return
			}
		}
		// close all trades based on blackout period
		now := time.Unix(b.currentTime/1000, 0)
		if isBlackout(now.In(b.blackoutTimes.TimeZone), b.blackoutTimes.StartTime, b.blackoutTimes.EndTime) {
			b.closePosition()
			log.Println("Broker closed trade due to blackout times")
			return
		}
	}
}

func (b *FuturesBroker) SendTrade(t Trade) error {
	if t.Operation == None {
		return nil
	}
	tradeTime := time.Unix(t.Time/1000, 0)
	if _, week := tradeTime.ISOWeek(); week != b.week {
		b.account.WeeklyWithdrawl()
		b.week = week
	}
	// Prevent entering trades during blackout period
	if isBlackout(tradeTime.In(b.blackoutTimes.TimeZone), b.blackoutTimes.StartTime, b.blackoutTimes.EndTime) {
		log.Println("Broker not entering trade due to blackout times")
		return nil
	}
	//q := max(1, (b.account.availableBalance / ((b.margin) + b.feePerSide)))
	// q := max(1, (b.account.availableBalance / (250 * 20)))
	// tradeQ := int(min(4, q))
	tradeQ := b.config.TradeQuantity
	t.Quantity = tradeQ
	fee := float64(tradeQ) * b.feePerSide
	switch t.Operation {
	case Close:
		b.closePosition()
	case Buy:
		if b.currentTrade != nil && b.currentTrade.Operation == Sell {
			b.closePosition()
			return nil
		}
		if b.currentTrade == nil {
			if !b.validTrade(&t) {
				log.Printf("Cannot create Buy trade: %v", ErrInsufficientBalance)
				return ErrInsufficientBalance
			}
			log.Printf("Broker Create Buy: Symbol: %v Price: %.2f Quantity: %d\n", t.Symbol, b.currentPrice, tradeQ)
			b.account.availableBalance -= (float64(tradeQ) * b.margin)
			b.account.availableBalance -= fee
			b.account.balance -= fee
			b.currentTrade = &t
			b.currentTrade.Quantity = tradeQ
			b.currentTrade.Time = b.currentTime
			b.currentTrade.OpenTime = b.currentTime
			// slippage
			b.currentTrade.Price = b.currentPrice + b.config.OpenSlippage
			b.currentTrade.OpenPrice = b.currentTrade.Price
			// set stops
			tradeStops := make([]stop.IStop, len(b.config.Stops))
			for i := 0; i < len(b.config.Stops); i++ {
				b.config.Stops[i].Direction = Buy
				b.config.Stops[i].FillPrice = b.currentTrade.Price
				tradeStops[i] = stop.NewStop(&b.config.Stops[i])
			}
			b.currentTrade.Stops = tradeStops
			// set profits
			tradeProfits := make([]profit.IProfit, len(b.config.Profits))
			for i := 0; i < len(b.config.Profits); i++ {
				b.config.Profits[i].Direction = Buy
				b.config.Profits[i].FillPrice = b.currentTrade.Price
				tradeProfits[i] = profit.NewProfit(&b.config.Profits[i])
			}
			b.currentTrade.Profits = tradeProfits
			log.Printf("Updated Available Balance: %.2f\n", b.account.availableBalance)
			// Track balance after opening trade
			b.trackBalanceAndDrawdown()
		}
	case Sell:
		if b.currentTrade != nil && b.currentTrade.Operation == Buy {
			b.closePosition()
			return nil
		}
		if b.currentTrade == nil {
			if !b.validTrade(&t) {
				log.Printf("Cannot create Sell trade: %v", ErrInsufficientBalance)
				return ErrInsufficientBalance
			}
			log.Printf("Broker Create Sell: Symbol %v Price: %.2f Quantity: %d\n", t.Symbol, b.currentPrice, tradeQ)
			b.account.availableBalance -= (float64(tradeQ) * b.margin)
			b.account.availableBalance -= fee
			b.account.balance -= fee
			b.currentTrade = &t
			b.currentTrade.Quantity = tradeQ
			b.currentTrade.Time = b.currentTime
			b.currentTrade.OpenTime = b.currentTime
			// slippage
			b.currentTrade.Price = b.currentPrice - b.config.OpenSlippage
			b.currentTrade.OpenPrice = b.currentTrade.Price
			// set stops
			tradeStops := make([]stop.IStop, len(b.config.Stops))
			for i := 0; i < len(b.config.Stops); i++ {
				b.config.Stops[i].Direction = Sell
				b.config.Stops[i].FillPrice = b.currentTrade.Price
				tradeStops[i] = stop.NewStop(&b.config.Stops[i])
			}
			b.currentTrade.Stops = tradeStops
			// set profits
			tradeProfits := make([]profit.IProfit, len(b.config.Profits))
			for i := 0; i < len(b.config.Profits); i++ {
				b.config.Profits[i].Direction = Sell
				b.config.Profits[i].FillPrice = b.currentTrade.Price
				tradeProfits[i] = profit.NewProfit(&b.config.Profits[i])
			}
			b.currentTrade.Profits = tradeProfits
			log.Printf("Updated Available Balance: %.2f\n", b.account.availableBalance)
			// Track balance after opening trade
			b.trackBalanceAndDrawdown()
		}
	default:
		//fmt.Println("Broker No indicator")
		return nil
	}
	return nil
}

func (b *FuturesBroker) updateBalance() {
	var net float64
	multiplier := b.pointPrice * float64(b.currentTrade.Quantity)
	fee := float64(b.currentTrade.Quantity) * b.feePerSide
	switch b.currentTrade.Operation {
	case Buy:
		net = (b.currentPrice - b.currentTrade.Price) * multiplier
	case Sell:
		net = (b.currentTrade.Price - b.currentPrice) * multiplier
	}
	if net > 0 {
		b.wins++
		b.cumWin += net
	} else {
		b.loses++
		b.cumLose += net
	}
	// update current trade max values
	b.currentTrade.MaxProfit = b.currentTrade.MaxProfit * multiplier
	b.currentTrade.MaxDrawdown = b.currentTrade.MaxDrawdown * multiplier
	b.account.balance += net
	b.account.availableBalance += ((b.margin * float64(b.currentTrade.Quantity)) + net)
	b.account.availableBalance -= fee
	b.account.balance -= fee
	b.currentTrade.Net = net
	log.Printf("Close trade at %v\n", time.Unix(b.currentTime/1000, 0))
	
	// Track balance history and drawdown
	b.trackBalanceAndDrawdown()
}

func (b *FuturesBroker) closePosition() {
	if b.currentTrade == nil {
		log.Printf("Close called with no open position")
		return
	}
	log.Printf("Broker Close Trade: Symbol: %v Price: %.2f Quantity: %d\n", b.currentTrade.Symbol, b.currentPrice, b.currentTrade.Quantity)
	b.currentTrade.CloseTime = b.currentTime
	b.currentTrade.ClosePrice = b.currentPrice
	b.updateBalance()
	b.trades = append(b.trades, b.currentTrade)
	b.currentTrade = nil
	log.Printf("Updated Balance: %.2f Available Balance: %.2f\n", b.account.balance, b.account.availableBalance)
}

func (b *FuturesBroker) updateTradeValues(d types.Data) {
	currDiff := d.Close - b.currentTrade.Price
	if b.currentTrade.Operation == Sell {
		currDiff = currDiff * -1
	}
	
	// Track MFE and MAE in points
	if currDiff > 0 {
		// trade is currently positive
		b.currentTrade.MaxProfit = max(b.currentTrade.MaxProfit, currDiff)
		b.currentTrade.MFE = b.currentTrade.MaxProfit * b.pointPrice // Convert points to dollars
	} else {
		// trade is currently negative
		b.currentTrade.MaxDrawdown = min(b.currentTrade.MaxDrawdown, currDiff)
		b.currentTrade.MAE = -b.currentTrade.MaxDrawdown * b.pointPrice // Convert points to dollars
	}
	
	// Calculate MFE and MAE as percentages
	if b.currentTrade.Price != 0 && b.pointPrice != 0 {
		// Calculate percentage based on points, not dollars  
		// MFE and MAE are in dollars, so convert back to points for percentage calculation
		mfePoints := b.currentTrade.MFE / b.pointPrice
		maePoints := b.currentTrade.MAE / b.pointPrice
		b.currentTrade.MFEPercent = (mfePoints / b.currentTrade.Price) * 100
		b.currentTrade.MAEPercent = (maePoints / b.currentTrade.Price) * 100
	}
}

func (b *FuturesBroker) validTrade(t *Trade) bool {
	if b.account.availableBalance-(b.margin*float64(t.Quantity)) < 0 {
		return false
	}
	return true
}

func (b *FuturesBroker) Summary() {
	// Close any open trades before generating summary
	if b.currentTrade != nil {
		log.Println("Closing open trade at end of backtest")
		b.closePosition()
	}
	
	WriteTradesToFile(b.output, b.trades)
	log.Printf("Number of wins: %v\n", b.wins)
	log.Printf("Cumulative win amount: $%v\n", b.cumWin)
	log.Printf("Number of loses: %v\n", b.loses)
	log.Printf("Cumulative lose amount: $%v\n", b.cumLose)
	log.Printf("Number of trades: %v\n", len(b.trades))
	log.Printf("Net profit: $%v\n", (b.cumWin + b.cumLose))
	if b.cumWin > 0 {
		log.Printf("Net profit percentage: %.02f%%\n", ((b.cumWin+b.cumLose)/b.cumWin)*100)
	} else {
		log.Printf("Net profit percentage: N/A (no winning trades)\n")
	}
	total := float64(b.wins + b.loses)
	if total > 0 {
		log.Printf("Accuracy: %.02f%%\n", (float64(b.wins)/total)*100)
	} else {
		log.Printf("Accuracy: N/A (no trades)\n")
	}
}

func (b *FuturesBroker) GetTrades() ([]*Trade, error) {
	// Return a copy of trades to prevent external modification
	tradesCopy := make([]*Trade, len(b.trades))
	copy(tradesCopy, b.trades)
	return tradesCopy, nil
}

func (b *FuturesBroker) trackBalanceAndDrawdown() {
	currentBalance := b.account.balance
	timestamp := time.Unix(b.currentTime/1000, 0)
	
	// Record balance point
	b.balanceHistory = append(b.balanceHistory, BalancePoint{
		Time:    timestamp,
		Balance: currentBalance,
	})
	
	// Update peak balance
	if currentBalance > b.peakBalance {
		b.peakBalance = currentBalance
		b.currentDrawdown = 0
	} else {
		// Calculate current drawdown
		b.currentDrawdown = b.peakBalance - currentBalance
		
		// Update max drawdown
		if b.currentDrawdown > b.maxDrawdown {
			b.maxDrawdown = b.currentDrawdown
		}
	}
}

func (b *FuturesBroker) GetBalanceHistory() []BalancePoint {
	// Return a copy to prevent external modification
	historyCopy := make([]BalancePoint, len(b.balanceHistory))
	copy(historyCopy, b.balanceHistory)
	return historyCopy
}

func (b *FuturesBroker) GetMaxDrawdown() float64 {
	return b.maxDrawdown
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
