package main

import (
	"github.com/toorop/go-bittrex"
	"testing"
	"time"
)

var c = bittrex.Candle{
	TimeStamp: bittrex.CandleTime{
		Time: time.Now(),
	},
}

var m = Market{
	Market:     bittrex.Market{},
	LastCandle: c,
}

func TestCandlesTimeDiff(t *testing.T) {
	c0 := c

	if CandleTimeDiff(c0, c0) != 0 {
		t.Error("c0 should equal c0")
	}

	c1 := c
	c1.TimeStamp.Time = c1.TimeStamp.Add(time.Second)
	if CandleTimeDiff(c0, c1) >= 0 {
		t.Error("c0 should be before c1")
	}
	if CandleTimeDiff(c1, c0) <= 0 {
		t.Error("c1 should be after c0")
	}
	if CandleTimeDiff(c1, c0) != time.Second {
		t.Error("c1 - c0 should equal 1sec")
	}
}

func TestMarket_IsCandleNew(t *testing.T) {
	c0 := c
	m0 := m
	m0.LastCandle = c0

	if m0.IsCandleNew(c0) {
		t.Error("c0 time should equal m0.LastCandle")
	}

	// 1 sec after c0
	c1 := c0
	c1.TimeStamp.Time = c1.TimeStamp.Time.Add(time.Second)

	if !m0.IsCandleNew(c1) {
		t.Error("c1 time should be newer of 1 sec")
	}

	m0.LastCandle = c1
	if m0.IsCandleNew(c0) {
		t.Error("c0 shouldn't be new now")
	}
}
