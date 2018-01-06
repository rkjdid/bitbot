package main

import (
	"github.com/toorop/go-bittrex"
)

type Market struct {
	bittrex.Market
	Interval   CandleInterval
	Candles    []bittrex.Candle
	LastCandle bittrex.Candle

	ShortMAs *MATrio
	LongMAs  *MATrio

	BBSum *MovingAverage

	ConsecutiveHits int
	TotalHits       int

	Client *bittrex.Bittrex
}

func NewMarket(market bittrex.Market, longLength, shortLength, bbLength int,
	interval CandleInterval, client *bittrex.Bittrex) *Market {

	return &Market{
		Market:   market,
		ShortMAs: NewMATrio(shortLength),
		LongMAs:  NewMATrio(longLength),
		BBSum:    NewMovingAverage(bbLength),
		Interval: interval,
		Client:   client,
	}
}
