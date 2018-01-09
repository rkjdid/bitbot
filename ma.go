package main

// https://github.com/RobinUS2/golang-moving-average
// taken from cause lazy
// @author Robin Verlangen
// Moving average implementation for Go

type MovingAverage struct {
	Window      int
	values      []float64
	valPos      int
	slotsFilled bool
}

func (ma *MovingAverage) Avg() float64 {
	var sum = float64(0)
	var c = ma.Window - 1

	// Are all slots filled? If not, ignore unused
	if !ma.slotsFilled {
		c = ma.valPos - 1
		if c < 0 {
			// Empty register
			return 0
		}
	}

	// Sum values
	var ic = 0
	for i := 0; i <= c; i++ {
		sum += ma.values[i]
		ic++
	}

	// Finalize average and return
	avg := sum / float64(ic)
	return avg
}

func (ma *MovingAverage) Add(val float64) {
	// Put into values array
	ma.values[ma.valPos] = val

	// Increment value position
	ma.valPos = (ma.valPos + 1) % ma.Window

	// Did we just go back to 0, effectively meaning we filled all registers?
	if !ma.slotsFilled && ma.valPos == 0 {
		ma.slotsFilled = true
	}
}

func (ma MovingAverage) Values() []float64 {
	if ma.slotsFilled {
		return ma.values
	}
	return ma.values[:ma.valPos]
}

func NewMovingAverage(window int) *MovingAverage {
	return &MovingAverage{
		Window:      window,
		values:      make([]float64, window),
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

func (t *MATrio) Add(p float64, v float64) {
	t.P.Add(p)
	t.V.Add(v)
	t.PV.Add(p * v)
}
