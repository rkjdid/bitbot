package main

import (
	"github.com/rkjdid/util"
	"time"
)

type CandleInterval string

const (
	CandleMinute    = CandleInterval("oneMin")
	Candle5Minutes  = CandleInterval("fiveMin")
	Candle30Minutes = CandleInterval("thirtyMin")
	CandleHour      = CandleInterval("hour")
	CandleDay       = CandleInterval("day")
)

var Candles = map[CandleInterval]util.Duration{
	CandleMinute:    util.Duration(time.Minute),
	Candle5Minutes:  util.Duration(time.Minute * 5),
	Candle30Minutes: util.Duration(time.Minute * 30),
	CandleHour:      util.Duration(time.Hour),
	CandleDay:       util.Duration(time.Hour * 24),
}

type Config struct {
	Scanner ScannerConfig
	Market  MarketConfig
	VPCI    VPCIConfig
}

type ScannerConfig struct {
	// only pick those markets, if nil or empty all BTC markets are monitored
	Pairs []string

	// only pick markets whose daily volume is greater than
	MinBtcVolumeDaily float64

	// number of consecutive hits to trigger notification
	NotificationThreshold int

	BittrexApiKey    string
	BittrexApiSecret string
}

type MarketConfig struct {
	// candle time string: one of "oneMin" "fiveMin" "thirtyMin" "hour" "day"
	Interval CandleInterval

	// Prefill is the amount of candles to fetch for market upon initialization
	Prefill int
}

type VPCIConfig struct {
	// long term MA length
	LongTerm int

	// short term MA length
	ShortTerm int

	BBLength   int
	Multiplier float64

	Enabled bool
}

var DefaultScannerConfig = Config{
	ScannerConfig{
		Pairs:                 []string{},
		MinBtcVolumeDaily:     200.0,
		NotificationThreshold: 0,
	},
	MarketConfig{
		Interval: Candle30Minutes,
		Prefill:  20,
	},
	VPCIConfig{
		LongTerm:   20,
		ShortTerm:  5,
		BBLength:   20,
		Multiplier: 2.5,
		Enabled:    true,
	},
}
