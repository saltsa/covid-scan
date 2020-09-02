package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/JuulLabs-OSS/ble"
)

// This code bases on goble example code

const minRSSI = -100
const maxRSSI = -20

// If no beacon, mark O
var oldEntry = 15 * time.Second

// Removes entry if no beacon
var cleanEntry = 2 * time.Minute

// flags L in output stats
var longlivedEntry = 5 * time.Minute

var statsChan chan struct{}
var statsTicker = time.NewTicker(10 * time.Second)
var total uint64
var adCount uint64
var esService ble.UUID = ble.UUID16(0xfd6f)

type enEntry struct {
	RSSI      []int
	FirstSeen time.Time
	LastSeen  time.Time
	Count     int
	Data      string
}

func rssiToRune(rssi int) rune {
	// level := (rssi - minRSSI) % 8
	step := (maxRSSI - minRSSI) / 8
	levelDown := (rssi - minRSSI)
	level := levelDown / step
	// levelUp := (maxRSSI - rssi)
	// level := int(float64(levelDown) / float64(levelDown+levelUp) * 0.8)
	return rune(0x2581 + level)
}

// Abs returns the absolute value of x.
func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (e *enEntry) String() string {
	first := time.Since(e.FirstSeen)
	last := time.Since(e.LastSeen)
	age := first - last

	var avg float64
	for _, i := range e.RSSI {
		avg += float64(i)
	}
	avg /= float64(len(e.RSSI))

	var tooOld = "-"
	var longLived = "-"
	if last > oldEntry {
		tooOld = "O"
	}

	if age > longlivedEntry {
		longLived = "L"
	}

	rssiS := make([]rune, 60)
	for i := 0; i < len(rssiS); i++ {
		idxOut := len(rssiS) - i - 1
		idxIn := len(e.RSSI) - 1 - i
		if idxOut >= 0 {
			if idxIn >= 0 {
				rssiS[idxOut] = rssiToRune(e.RSSI[idxIn])
			} else {
				rssiS[idxOut] = rune(0x20)
			}
		}
	}

	return fmt.Sprintf("%s%s rssi=[%s] last=%s age=%s first=%s count=%d RSSI avg=%f latest=%d (%s)", tooOld, longLived, string(rssiS), last, first, age, e.Count, avg, e.RSSI[len(e.RSSI)-1], e.Data)
}

func (e *enEntry) CandidatePrevious() (bool, string) {
	step := time.Since(e.FirstSeen)
	const beaconCount = 20
	if step > 10*time.Second && step < 60*time.Second {
		if e.Count > beaconCount {
			return false, ""
		}
		// TODO: Concurrency problem
		var candi string
		mutex.RLock()
		for _, ee := range srcMap {
			if ee.Data == e.Data {
				continue
			}
			if Abs(ee.RSSI[len(ee.RSSI)-1]-e.RSSI[len(e.RSSI)-1]) > 15 {
				continue
			}
			// Beacons are sent about once per second, make sure we haven't seen them too much

			if ee.Count < beaconCount {
				continue
			}
			if e.FirstSeen.After(ee.LastSeen) {
				candi = ee.Data
				break
			}
		}
		mutex.RUnlock()

		if candi != "" {
			return true, candi
		}
	}

	return false, ""
}

func (e *enEntry) Expired() bool {
	if time.Since(e.LastSeen) > cleanEntry {
		return true
	}
	return false
}

var dataMap map[string]int
var srcMap map[string]*enEntry
var mutex sync.RWMutex

func init() {
	dataMap = make(map[string]int)
	srcMap = make(map[string]*enEntry)

	statsChan = make(chan struct{})
}

func Add(srcs string, data []byte, rssi int) {
	var found bool
	var new bool

	// if rssi < -70 {
	// 	return
	// }
	// log.Println("RSSI:", rssi)
	datas := fmt.Sprintf("%x", data)
	if len(datas) != 40 {
		log.Println("invalid data length")
		return
	}
	_, found = dataMap[datas]
	if !found {
		dataMap[datas] = 1
		new = true
	} else {
		dataMap[datas]++
	}

	mutex.RLock()
	ent, found := srcMap[srcs]
	mutex.RUnlock()
	if !found {
		ent = &enEntry{
			FirstSeen: time.Now(),
			RSSI:      make([]int, 0),
			Data:      datas,
		}
		new = true

		mutex.Lock()
		srcMap[srcs] = ent
		mutex.Unlock()

	} else {
		if ent.Data != datas {
			log.Printf("data differs in saved entity: %s vs. %s", ent.Data, datas)
		}
	}

	mutex.Lock()
	ent.Count++
	ent.LastSeen = time.Now()
	ent.RSSI = append(ent.RSSI, rssi)
	total++
	mutex.Unlock()

	if new {
		statsChan <- struct{}{}
	} else if total%10 == 0 {
		statsChan <- struct{}{}
	}
}

func Delete(key string) {
	mutex.Lock()
	delete(srcMap, key)
	mutex.Unlock()
}

func stats() {
	for {

		var keys []string
		mutex.RLock()
		for key := range srcMap {
			keys = append(keys, key)
		}
		mutex.RUnlock()

		sort.Slice(keys, func(i, j int) bool {
			ar := srcMap[keys[i]].RSSI
			br := srcMap[keys[j]].RSSI

			a := ar[len(ar)-1]
			b := br[len(br)-1]
			return a > b
		})

		fmt.Printf("List of found beacons at %s\n", time.Now().Format("15:04:05"))
		for _, key := range keys {
			mutex.RLock()
			data := srcMap[key]
			mutex.RUnlock()
			fmt.Printf("%s\n", data)
			ok, candi := data.CandidatePrevious()

			if ok {
				fmt.Printf("Possibly change in data from %s to %s\n", candi, data.Data)
			}
			if data.Expired() {
				Delete(key)
			}
		}
		fmt.Println("end of list")
		fmt.Println()

		select {
		case <-statsChan:
		case <-statsTicker.C:
		}
	}
}

func hand(a ble.Advertisement) {
	adCount++

	addr := a.Addr()
	svcData := a.ServiceData()

	for _, svc := range svcData {
		if svc.UUID.Equal(esService) {
			Add(addr.String(), svc.Data, a.RSSI())
		}
	}

}

func scan(dups bool) {
	ctx := context.Background()

	err := ble.Scan(ctx, dups, hand, nil)

	if err != nil {
		log.Fatalln("scan error:", err)
	}
}

func main() {
	dups := flag.Bool("allow-duplicates", true, "allow duplicates when scanning")
	flag.Parse()

	dev, err := getDev()
	if err != nil {
		log.Fatalln("get dev", err)
	}
	ble.SetDefaultDevice(dev)
	go stats()

	log.Println("start scan")
	scan(*dups)
	log.Println("scan exit")

}
