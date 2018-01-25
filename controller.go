package main

import (
	"github.com/toorop/go-bittrex"
	"log"
	"strings"
	"time"
)

type Controller struct {
	Config  *ControllerConfig
	Markets map[string]*Market
	Client  *bittrex.Bittrex
	stop    chan interface{}
	signals chan Signal
}

func NewController(cfg ControllerConfig) *Controller {
	return &Controller{
		Config:  &cfg,
		Client:  bittrex.New(cfg.BittrexApiKey, cfg.BittrexApiSecret),
		Markets: make(map[string]*Market),
		signals: make(chan Signal),
	}
}

func (ctrl *Controller) NewMarket(market bittrex.Market, cfg MarketConfig) *Market {
	return NewMarket(market, cfg, ctrl.Client)
}

func (ctrl *Controller) InitMarkets(mcfg MarketConfig, vpcicfg VPCIConfig) error {
	markets, err := ctrl.Client.GetMarkets()
	if err != nil {
		return err
	}

	for _, market := range markets {
		name := market.MarketName
		if !market.IsActive {
			// only monitor active markets
			continue
		}

		if ctrl.Config.Pairs != nil && len(ctrl.Config.Pairs) > 0 {
			// only track this market if it is in Config.Pairs filter
			trackMarket := false
			for _, pair := range ctrl.Config.Pairs {
				if pair == market.MarketName {
					trackMarket = true
					break
				}
			}
			if !trackMarket {
				continue
			}
		} else {
			if strings.Index(name, "BTC") != 0 {
				// only monitor btc markets
				continue
			}

			summary, err := ctrl.Client.GetMarketSummary(name)
			if err != nil {
				log.Printf("error retreiving market history for %s: %s", name, err)
				continue
			}

			bv, _ := summary[0].BaseVolume.Float64()
			if bv < ctrl.Config.MinBtcVolumeDaily {
				// filter out low volume markets
				log.Printf("filtering out low volume market %s (base vol: %5f)", name, bv)
				continue
			}
		}

		m := ctrl.NewMarket(market, mcfg)
		err = m.PrefillCandles()
		if err != nil {
			log.Printf("error filling candles for %s: %s", market.MarketName, err)
			continue
		}

		// add desired indicators
		ind := NewVPCI(market.MarketName, vpcicfg, []chan<- Signal{chan<- Signal(ctrl.signals)})
		m.AddIndicator(ind)

		ctrl.Markets[name] = m
		log.Println("tracking market", name)
	}
	return nil
}

func (ctrl *Controller) Stop() {
	if ctrl.stop == nil {
		return
	}
	ctrl.stop <- nil
	ctrl.stop = nil
}

func (ctrl *Controller) run() {
	ctrl.stop = make(chan interface{})
	for {
		select {
		case sig := <-ctrl.signals:
			log.Println(sig)
		case <-ctrl.stop:
			for _, market := range ctrl.Markets {
				market.Stop()
			}
			return
		}
	}
}

func (ctrl *Controller) Start(vpci VPCIConfig) error {
	log.Printf("%d tracked markets", len(ctrl.Markets))
	go ctrl.run()
	for _, market := range ctrl.Markets {
		go market.StartPolling()
	}
	return nil
}

func (ctrl *Controller) Analyze(marketName string, cfg MarketConfig, from, to time.Time) error {
	log.Printf("starting market analysis for \"%s\"", marketName)
	var bFrom, bTo bool
	if !from.Equal(time.Time{}) {
		bFrom = true
		log.Printf("from: %s", from)
	}
	if !to.Equal(time.Time{}) {
		bTo = true
		log.Printf("  to: %s", to)
	}

	m := ctrl.NewMarket(bittrex.Market{MarketName: marketName}, cfg)
	candles, err := ctrl.Client.GetTicks(marketName, string(cfg.Interval))
	if err != nil {
		return err
	}

	for _, c := range candles {
		// candles should come in historical order..
		if bFrom && c.TimeStamp.Time.Before(from) {
			continue
		}
		if bTo && c.TimeStamp.Time.After(to) {
			break
		}

		_ = m.AddTick(c, false)
	}

	return nil
}
