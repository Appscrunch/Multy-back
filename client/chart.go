package client

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/bitly/go-nsq"
)

var (
	s1 = rand.NewSource(time.Now().UnixNano())
	r1 = rand.New(s1)
)

const (
	secondsInDay   = 8640
	numOfChartDots = 120 //12 minutes

	defaultNSQAddr = "127.0.0.1:4150"
)

type Rates struct {
	BTCtoUSDDay   map[time.Time]float64
	echangeSingle *EventExchangeChart

	m *sync.Mutex
}

type exchangeChart struct {
	rates *Rates

	ticker      *time.Ticker
	interval    int
	nsqProducer *nsq.Producer
}

type EventExchangeChart struct {
	EURtoBTC float64
	USDtoBTC float64
	ETHtoBTC float64

	ETHtoUSD float64
	ETHtoEUR float64

	BTCtoUSD float64
}

func initExchangeChart() (*exchangeChart, error) {
	chart := &exchangeChart{
		rates: &Rates{
			BTCtoUSDDay: make(map[time.Time]float64),
			m:           &sync.Mutex{},
		},
		interval: secondsInDay / numOfChartDots,
	}

	p, err := nsq.NewProducer(defaultNSQAddr, nsq.NewConfig())
	if err != nil {
		return nil, fmt.Errorf("exchange chart: NSQ new producer: %s", err.Error())
	}
	chart.nsqProducer = p
	chart.updateRateAll()

	return chart, nil

}

func (eChart *exchangeChart) run() error {
	eChart.updateRateAll()
	eChart.ticker = time.NewTicker(time.Duration(eChart.interval) * time.Second)
	log.Printf("[DEBUG] updateExchange: ticker=%ds\n", eChart.interval)

	for {
		select {
		case _ = <-eChart.ticker.C:
			log.Println("[DEBUG] updateExchange ticker")
			eChart.updateRate()

			exchangesJSON, err := json.Marshal(eChart.rates.echangeSingle)
			if err != nil {
				log.Printf("[ERR] exchange chart run: %s\n", err.Error())
				continue
			}

			if err = eChart.nsqProducer.Publish("/exchangeChart", exchangesJSON); err != nil {
				return fmt.Errorf("[ERR] exchange chart: NSQ publish: %s", err.Error())
			}
		}
	}
}

func (eChart *exchangeChart) updateRate() {
	log.Printf("[DEBUG] updateExchange; mock implementation\n")

	eChart.rates.m.Lock()
	defer eChart.rates.m.Unlock()

	eChart.rates.echangeSingle.ETHtoBTC = r1.Float64()*5 + 5
	eChart.rates.echangeSingle.USDtoBTC = r1.Float64()*5 + 5
	eChart.rates.echangeSingle.ETHtoBTC = r1.Float64()*5 + 5

	eChart.rates.echangeSingle.ETHtoUSD = r1.Float64()*5 + 5
	eChart.rates.echangeSingle.ETHtoEUR = r1.Float64()*5 + 5

	// TODO: do it gracefully
	theOldest, theNewest := getExtremRates(eChart.rates.BTCtoUSDDay)
	delete(eChart.rates.BTCtoUSDDay, theOldest)
	eChart.rates.BTCtoUSDDay[theNewest.Add(time.Duration(eChart.interval)*time.Second)] = r1.Float64()*5 + 5

	eChart.rates.echangeSingle.BTCtoUSD = eChart.rates.BTCtoUSDDay[theNewest.Add(time.Duration(eChart.interval)*time.Second)]

	return
}

func (eChart *exchangeChart) updateRateAll() {
	log.Printf("[DEBUG] updateExchange; mock implementation\n")

	aDayAgoTime := time.Now()
	aDayAgoTime.AddDate(0, 0, -1)

	for i := 0; i < numOfChartDots; i += eChart.interval {
		eChart.rates.BTCtoUSDDay[aDayAgoTime.Add(-time.Second*time.Duration(i))] = r1.Float64()*5 + 5
	}

	log.Printf("[DEBUG] updateRateAll: BTCtoUSDDay=%+v/n", eChart.rates.BTCtoUSDDay)
	return
}

func getExtremRates(rates map[time.Time]float64) (time.Time, time.Time) {
	var min, max time.Time
	for rt := range rates {
		if rt.Unix() <= min.Unix() {
			min = rt
		}
		if rt.Unix() > max.Unix() {
			max = rt
		}

	}
	return min, max
}
