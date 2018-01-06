package main

import (
	"github.com/rkjdid/bitbot/movingaverage"
	"github.com/toorop/go-bittrex"
)

type Market struct {
	bittrex.Market
	Candles []bittrex.Candle

	ShortMAs *MATrio
	LongMAs  *MATrio

	BBSum *movingaverage.MovingAverage

	ConsecutiveHits int
	TotalHits       int
}

type MATrio struct {
	Length int
	P      *movingaverage.MovingAverage
	V      *movingaverage.MovingAverage
	PV     *movingaverage.MovingAverage
}

func NewMATrio(length int) *MATrio {
	return &MATrio{
		length,
		movingaverage.New(length),
		movingaverage.New(length),
		movingaverage.New(length),
	}
}

func (t *MATrio) Add(p float64, v float64) {
	t.P.Add(p)
	t.V.Add(v)
	t.PV.Add(p * v)
}

func NewMarket(market bittrex.Market, longLength, shortLength, bbLength int) *Market {
	return &Market{
		Market:   market,
		ShortMAs: NewMATrio(shortLength),
		LongMAs:  NewMATrio(longLength),
		BBSum:    movingaverage.New(bbLength),
	}
}
