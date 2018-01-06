package main

import (
	"github.com/toorop/go-bittrex"
)

type Market struct {
	bittrex.Market
	Candles    []bittrex.Candle
	LastCandle bittrex.Candle

	ShortMAs *MATrio
	LongMAs  *MATrio

	BBSum *MovingAverage

	ConsecutiveHits int
	TotalHits       int
}

func NewMarket(market bittrex.Market, longLength, shortLength, bbLength int) *Market {
	return &Market{
		Market:   market,
		ShortMAs: NewMATrio(shortLength),
		LongMAs:  NewMATrio(longLength),
		BBSum:    NewMovingAverage(bbLength),
	}
}
