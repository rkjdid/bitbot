package main

import (
	"github.com/rkjdid/util"
	"time"
)

const (
	CandleMinute    = "oneMin"
	Candle5Minutes  = "fiveMin"
	Candle30Minutes = "thirtyMin"
	CandleHour      = "hour"
	CandleDay       = "day"
)

var Candles = map[string]util.Duration{
	CandleMinute:    util.Duration(time.Minute),
	Candle5Minutes:  util.Duration(time.Minute * 5),
	Candle30Minutes: util.Duration(time.Minute * 30),
	CandleHour:      util.Duration(time.Hour),
	CandleDay:       util.Duration(time.Hour * 24),
}

type Config struct {
	Scanner *ScannerConfig
}

type ScannerConfig struct {
	// only pick those markets, if nil or empty all BTC markets are monitored
	Pairs []string

	// only pick markets whose 24h volume is greater than
	Min24hVolume float64

	// candle time string: one of "oneMin" "fiveMin" "thirtyMin" "hour" "day"
	Candle string

	// long term MA length
	LongTerm int

	// short term MA length
	ShortTerm int

	BBLength   int
	Multiplier float64

	// number of consecutive hits to trigger notification
	NotificationThreshold int
}

var DefaultConfig = Config{
	Scanner: &ScannerConfig{
		Pairs:      []string{},
		Candle:     Candle30Minutes,
		LongTerm:   20,
		ShortTerm:  5,
		BBLength:   20,
		Multiplier: 2.5,
	},
}
