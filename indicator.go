package main

import (
	"fmt"
	"github.com/montanaflynn/stats"
	"github.com/rkjdid/util"
	"github.com/shopspring/decimal"
	"github.com/toorop/go-bittrex"
	"log"
	"sync"
	"time"
)

type Action int

const (
	Buy = Action(iota)
	Sell
)

// Hit holds information to be passed to subscribers when an Indicator triggers.
type Signal struct {
	Action      Action
	Name        string
	Consecutive int
	Total       int
	Data        interface{}
}

// Indicator is an interface wrapping both technical analysis computing and communication methods.
//
// An indicator must describe itself (fmt.Stringer), compute a new bittrex candle (AddTick)
// and accept subscription to provided channel (Subscribe).
//
// Each call to AddCandle is susceptible to trigger a send to provided channel in Subscribe.
type Indicator interface {
	fmt.Stringer
	Subscribe(chan<- Signal)
	AddTick(bittrex.Candle) error
}

// BaseIndicator holds basic information and communication mechanisms for any indicator. It is
// not an Indicator since it doesn't implement AddTick()
type BaseIndicator struct {
	BuyConsecutives int
	BuyTotal        int
	Name            string
	Subscriptions   []chan<- Signal
	Timeout         time.Duration
	sync.Mutex
}

func (b *BaseIndicator) Subscribe(ch chan<- Signal) {
	b.Lock()
	if b.Subscriptions == nil {
		b.Subscriptions = make([]chan<- Signal, 0)
	}
	b.Subscriptions = append(b.Subscriptions, ch)
	b.Unlock()
}

func (b *BaseIndicator) Unsubscribe(ch chan<- Signal) {
	if b.Subscriptions == nil {
		return
	}

	b.Lock()
	for k, v := range b.Subscriptions {
		if v == ch {
			b.Subscriptions = append(b.Subscriptions[:k], b.Subscriptions[k+1:]...)
			break
		}
	}
	b.Unlock()
}

func (b *BaseIndicator) Broadcast(hit Signal) {
	b.Lock()
	done := make(chan bool)
	go func() {
		for _, ch := range b.Subscriptions {
			ch <- hit
		}
		close(done)
	}()

	select {
	case <-time.After(b.Timeout):
		log.Printf("%s: broadcast timed out %s", b, b.Timeout)
	case <-done:
	}
	b.Unlock()
}

func (b *BaseIndicator) String() string {
	return b.Name
}

type VPCI struct {
	BaseIndicator
	ShortMAs   *MATrio
	LongMAs    *MATrio
	BBSum      *MovingAverage
	Multiplier float64
}

func NewVPCI(name string, cfg VPCIConfig) *VPCI {
	return &VPCI{
		BaseIndicator: BaseIndicator{
			Name:    name,
			Timeout: time.Second * 10,
		},
		ShortMAs:   NewMATrio(cfg.ShortTerm),
		LongMAs:    NewMATrio(cfg.LongTerm),
		BBSum:      NewMovingAverage(cfg.BBLength),
		Multiplier: cfg.Multiplier,
	}
}

func (vpci *VPCI) AddTick(c bittrex.Candle) error {
	p := c.Close
	v := c.Volume
	vpci.ShortMAs.Add(p, v)
	vpci.LongMAs.Add(p, v)

	vpc := (vpci.LongMAs.PV.Avg().Div(vpci.LongMAs.V.Avg())).Sub(vpci.LongMAs.P.Avg())
	vpr := (vpci.ShortMAs.PV.Avg().Div(vpci.ShortMAs.V.Avg())).Div(vpci.ShortMAs.P.Avg())
	vm := vpci.ShortMAs.V.Avg().Div(vpci.LongMAs.V.Avg())
	result := vpc.Mul(vpr).Mul(vm)

	vpci.BBSum.Add(result)

	dev, err := stats.StandardDeviation(stats.Float64Data(vpci.BBSum.FloatValues()))
	if err != nil {
		return err
	}

	basis := vpci.BBSum.Avg()
	dev *= vpci.Multiplier

	buy := result.GreaterThan(basis.Add(decimal.NewFromFloat(dev)))
	if buy {
		vpci.BuyConsecutives += 1
		vpci.BuyTotal += 1
	} else {
		vpci.BuyConsecutives = 0
	}

	if *verbose {
		log.Printf("%12s - %s - price %s - volume: %s - MA_P: %s / %s, MA_V: %s / %s, MA_PV: %s / %s\n\t"+
			"vpc: %s, vpr: %s, vm: %s, vpci: %s, basis: %s, dev: %f",
			vpci, util.ParisTime(c.TimeStamp.Time), p, v,
			vpci.ShortMAs.P.Avg(), vpci.LongMAs.P.Avg(),
			vpci.ShortMAs.V.Avg(), vpci.LongMAs.V.Avg(),
			vpci.ShortMAs.PV.Avg(), vpci.LongMAs.PV.Avg(),
			vpc, vpr, vm, result, basis, dev)
	}

	if buy {
		vpci.Broadcast(Signal{
			Action:      Buy,
			Consecutive: vpci.BuyConsecutives,
			Total:       vpci.BuyTotal,
			Name:        vpci.String(),
		})
	}
	return nil
}
