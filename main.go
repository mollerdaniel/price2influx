// Bwulfs super-price-pusher-3000
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

var influxaddr = flag.String("influxaddr", "http://localhost:8086", "InfluxDB address")
var influxtoken = flag.String("influxtoken", "my-token", "InfluxDB token")

const (
	InfluxOrg    = "my-org"
	InfluxBucket = "my-bucket"
	Locale       = "Europe/Stockholm"
)

// {"SEK_per_kWh":0.4931,"EUR_per_kWh":0.04295,"EXR":11.480681,"time_start":"2025-01-29T00:00:00+01:00","time_end":"2025-01-29T01:00:00+01:00"}
type Price struct {
	SEKPerkWh float64   `json:"SEK_per_kWh"`
	EURPerkWh float64   `json:"EUR_per_kWh"`
	EXR       float64   `json:"EXR"`
	TimeStart time.Time `json:"time_start"`
	TimeEnd   time.Time `json:"time_end"`
}

type Prices []Price

func createElprisURL() string {
	loc, _ := time.LoadLocation(Locale)
	return fmt.Sprintf("https://www.elprisetjustnu.se/api/v1/prices/%d/%s_SE3.json",
		time.Now().In(loc).Year(),
		time.Now().In(loc).Format("01-02"),
	)
}

type PriceClient struct {
	prices Prices
	mutex  sync.Mutex
}

func (p *PriceClient) GetCurrentPrice() (float64, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	now := time.Now()
	for _, price := range p.prices {
		if price.TimeStart.Before(now) && price.TimeEnd.After(now) {
			return price.SEKPerkWh, nil
		}
	}
	return 0, fmt.Errorf("no current price found, no fresh data?")
}

func (p *PriceClient) LoadPrices() error {
	resp, err := http.Get(createElprisURL())
	if err != nil {
		return fmt.Errorf("error reading from elprisetjust.nu: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body) // response body is []byte
	if err != nil {
		return fmt.Errorf("error reading response: %v", err)
	}
	var prices Prices
	err = json.Unmarshal(body, &prices)
	if err != nil {
		return fmt.Errorf("error parsing json: %v", err)
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.prices = prices
	fmt.Println("Prices loaded from elprisetjust.nu")
	return nil
}

func main() {
	wg := sync.WaitGroup{}
	priceClient := &PriceClient{}
	err := priceClient.LoadPrices()
	if err != nil {
		log.Fatal(err)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			loc, err := time.LoadLocation(Locale)
			if err != nil {
				log.Fatal(err)
			}
			year, month, day := time.Now().In(loc).Date()
			t := time.Date(year, month, day, 0, 0, 0, 0, loc).AddDate(0, 0, 1).Add(time.Second)
			fmt.Println("Fetching new prices at:", t)
			<-time.After(time.Until(t))
			fmt.Println("Fetching new prices")
			priceClient.GetCurrentPrice()
		}
	}()

	client := influxdb2.NewClient(*influxaddr, *influxtoken)
	writeAPI := client.WriteAPIBlocking(InfluxOrg, InfluxBucket)

	timeout := 10 * time.Second
	ticker := time.NewTicker(timeout)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			<-ticker.C
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), timeout-(time.Millisecond*500))
				defer cancel()
				// Create point using fluent style
				price, err := priceClient.GetCurrentPrice()
				if err != nil {
					log.Printf("GetCurrentPrice: %v", err)
					return
				}
				p := influxdb2.NewPointWithMeasurement("price").
					AddTag("currency", "SEK").
					AddField("price", price).
					SetTime(time.Now())
				err = writeAPI.WritePoint(ctx, p)
				if err != nil {
					log.Printf("Write to influx failed: %v", err)
				}
			}()
		}
	}()

	fmt.Println("Started")
	wg.Wait()
}
