package main

import (
	"github.com/shopspring/decimal"
)

// https://github.com/RobinUS2/golang-moving-average
// taken from cause lazy
// @author Robin Verlangen
// Moving average implementation for Go

type MovingAverage struct {
	Window      int
	values      []decimal.Decimal
	valPos      int
	slotsFilled bool
}

func (ma *MovingAverage) Avg() decimal.Decimal {
	var sum = decimal.NewFromFloat(0.0)
	var c = ma.Window - 1

	// Are all slots filled? If not, ignore unused
	if !ma.slotsFilled {
		c = ma.valPos - 1
		if c < 0 {
			// Empty register
			return decimal.New(0, 0)
		}
	}

	// Sum values
	var ic = 0
	for i := 0; i <= c; i++ {
		sum = sum.Add(ma.values[i])
		ic++
	}

	return sum.Div(decimal.NewFromFloat(float64(ic)))
}

func (ma *MovingAverage) Add(val decimal.Decimal) {
	// Put into values array
	ma.values[ma.valPos] = val

	// Increment value position
	ma.valPos = (ma.valPos + 1) % ma.Window

	// Did we just go back to 0, effectively meaning we filled all registers?
	if !ma.slotsFilled && ma.valPos == 0 {
		ma.slotsFilled = true
	}
}

func (ma MovingAverage) Values() []decimal.Decimal {
	if ma.slotsFilled {
		return ma.values
	}

	return ma.values[:ma.valPos]
}

func (ma MovingAverage) FloatValues() []float64 {
	values := ma.Values()
	fvalues := make([]float64, len(values))
	for i := range values {
		f, _ := values[i].Float64()
		fvalues[i] = f
	}
	return fvalues
}

func NewMovingAverage(window int) *MovingAverage {
	return &MovingAverage{
		Window:      window,
		values:      make([]decimal.Decimal, window),
		valPos:      0,
		slotsFilled: false,
	}
}

type MATrio struct {
	Length int
	P      *MovingAverage
	V      *MovingAverage
	PV     *MovingAverage
}

func NewMATrio(length int) *MATrio {
	return &MATrio{
		length,
		NewMovingAverage(length),
		NewMovingAverage(length),
		NewMovingAverage(length),
	}
}

func (t *MATrio) Add(p decimal.Decimal, v decimal.Decimal) {
	t.P.Add(p)
	t.V.Add(v)
	t.PV.Add(p.Mul(v))
}
