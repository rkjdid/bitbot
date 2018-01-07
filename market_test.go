package main

import (
	"github.com/toorop/go-bittrex"
	"testing"
	"time"
)

var market_ETHBTC = &Market{
	Market: bittrex.Market{
		MarketCurrency: "ETH",
		BaseCurrency:   "BTC",
		MarketName:     "BTC-ETH",
		IsActive:       true,
	},

	Interval: CandleHour,
	Client:   bittrex.New("", ""),
}

func TestCandlesEqual(t *testing.T) {
	c0, err := market_ETHBTC.GetLatestTick()
	if err != nil {
		t.Fatalf("GetLatestTick: %s", err)
	}

	if !CandlesEqual(c0, c0) {
		t.Error("c0 should equal c1")
	}

	c1 := c0
	c1.Open = c1.Open.Ceil()
	if CandlesEqual(c0, c1) {
		t.Error("c0 should not equal c1")
	}

	c1.Open = c0.Open
	if !CandlesEqual(c0, c1) {
		t.Error("c0 should equal c1")
	}

	c1.TimeStamp = bittrex.CandleTime{
		Time: c1.TimeStamp.Time.Add(time.Second),
	}
	if CandlesEqual(c0, c1) {
		t.Error("c0 should not equal c1")
	}
}

func TestMarket_IsCandleNew(t *testing.T) {
	c0, err := market_ETHBTC.GetLatestTick()
	if err != nil {
		t.Fatalf("GetLatestTick: %s", err)
	}

	market_ETHBTC.LastCandle = c0

	c1, err := market_ETHBTC.GetLatestTick()
	if err != nil {
		t.Fatalf("GetLatestTick: %s", err)
	}

	if market_ETHBTC.IsCandleNew(c1) {
		// bad hourly timing?
		market_ETHBTC.LastCandle = c1
		c1, err = market_ETHBTC.GetLatestTick()
		if err != nil {
			t.Fatalf("GetLatestTick: %s", err)
		}
	}

	if market_ETHBTC.IsCandleNew(c1) {
		t.Error("Candle shouldn't be new")
	}
}
