package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/JuulLabs-OSS/ble"
	log "github.com/sirupsen/logrus"
)

var adCount uint64
var measureCount uint64

var esService ble.UUID = ble.UUID16(0xfd6f)

// TODO: Fix this beaconinfo and names here
// and functions
type beaconInfo struct {
	RSSI  int
	Addr  string
	Count int
}

type beaconMap map[string]beaconInfo
type adsPerMinute []beaconMap

var adsSlice adsPerMinute
var adsMutex sync.Mutex

func (apm adsPerMinute) AddBeacon(data string, rssi int, addr string) {
	adsMutex.Lock()

	min := time.Now().Minute()

	a, ok := adsSlice[min][data]
	if !ok {
		adsSlice[min][data] = beaconInfo{
			RSSI:  rssi,
			Count: 1,
			Addr:  addr,
		}
	} else {
		a.Count++
		if a.RSSI < rssi {
			a.RSSI = rssi
		}
	}

	adsMutex.Unlock()
}

func (apm adsPerMinute) Print() {
	adsMutex.Lock()

	min := time.Now().Minute()

	log.Infof("Ads per minute")
	for i := (min + 60 - 20); i <= (min + 60); i++ {
		// log.Debugf("i is %d", i)
		idx := i % 60
		val := adsSlice[idx]

		var rssiMin = -110
		var rssiMax = -10
		// for _, adv := range val {
		// 	if adv.RSSI < rssiMin {
		// 		rssiMin = adv.RSSI
		// 	}
		// 	if adv.RSSI > rssiMax {
		// 		rssiMax = adv.RSSI
		// 	}
		// }

		if rssiMin >= rssiMax {
			fmt.Printf("%02d: %3d\n", idx, len(val))
			continue
		}
		stepLen := rssiMax - rssiMin
		// log.Debugf("max=%d min=%d steplen is %d", rssiMax, rssiMin, stepLen)
		metrics := make([]rune, stepLen+1)

		for r := 0; r <= stepLen; r++ {
			var c int
			for _, j := range val {
				if j.RSSI == rssiMin+r {
					c++
				}
			}
			if c > 8 {
				c = 8
			}

			// U+2581 - U+2588
			metric := rune(0x2580 + c)
			if c == 0 {
				metric = ' '
			}
			metrics[r] = metric
		}
		fmt.Printf("%02d: %3d %4d [%s] %4d\n", idx, len(val), rssiMin, string(metrics), rssiMax)
	}
	adsMutex.Unlock()
}

var fout *os.File

func toFile(data string) {
	_, err := fout.WriteString(data + "\n")
	if err != nil {
		log.Fatalln(err)
	}
}
func init() {
	is := time.Now()
	adsSlice = make([]beaconMap, 60)
	for i := range adsSlice {
		adsSlice[i] = make(beaconMap)
	}
	adsMutex = sync.Mutex{}
	var err error
	fout, err = os.OpenFile("covid.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalln(err)
	}
	log.Infof("data structures built in %s", time.Since(is))
}
func hand(a ble.Advertisement) {
	adCount++

	addr := a.Addr()
	svcData := a.ServiceData()

	// log.Debugf("tx power: %d", a.TxPowerLevel())
	for _, svc := range svcData {
		if svc.UUID.Equal(esService) {
			measureCount++
			ds := fmt.Sprintf("%X", svc.Data)
			as := fmt.Sprintf("%X", addr)
			dout := fmt.Sprintf("%d,%s,%s,%d", time.Now().Unix(), as, ds, a.RSSI())
			toFile(dout)
			adsSlice.AddBeacon(ds, a.RSSI(), as)
		}
	}

}

func scan() {
	ctx := context.Background()

	err := ble.Scan(ctx, true, hand, nil)

	if err != nil {
		log.Fatalln("scan error:", err)
	}
}

func printStats() {
	go func() {
		for {
			log.Printf("measures=%d, total_ads=%d", measureCount, adCount)
			adsSlice.Print()
			time.Sleep(15 * time.Second)
		}
	}()
}

func main() {
	log.SetLevel(log.DebugLevel)
	log.Debug("init")
	dev, err := getDev()
	if err != nil {
		log.Fatalln("got error:", err)
	}
	ble.SetDefaultDevice(dev)
	printStats()
	scan()
}
