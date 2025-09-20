package main

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
   "strings"
	"sync"
	"time"

	"gonum.org/v1/gonum/mat"
)

const (
   n = 100
)

var (
	mutex = sync.RWMutex{}
)

func sample(paths []string) (float64, string) {
	maxTemp := 0
   var maxSocket string

	for _, path := range paths {
		val, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}

		val2, err := strconv.Atoi(strings.TrimSpace(string(val)))
		if err != nil {
			panic(err)
		}

		if val2 > maxTemp {
			maxTemp = val2
         maxSocket = path
		}
	}

	return float64(maxTemp) / 1000., maxSocket
}

func worker() {
	m := mat.NewDense(n, n, nil)

	for i := 0; i < m.RawMatrix().Rows; i++ {
		for j := 0; j < m.RawMatrix().Cols; j++ {
			m.Set(i, j, rand.Float64())
		}
	}

	var lu mat.LU
	var qr mat.QR
	var eig mat.Eigen

	for {
		mutex.RLock()

		lu.Factorize(m)
		qr.Factorize(m)

		ok := eig.Factorize(m, mat.EigenBoth)
		if !ok {
			panic("failed to factorise")
		}

		_ = eig.Values(nil)

		mutex.RUnlock()
	}
}

func top() error {
	paths, err := filepath.Glob("/sys/class/thermal/thermal_zone*/temp")
	if err != nil {
		return err
	}

   n := runtime.NumCPU() - 1
   fmt.Fprintf(os.Stderr, "starting up %d threads...", n)
	mutex.Lock()

	for _ = range n {
		go worker()
	}

   fmt.Fprintln(os.Stderr, "done")
   time.Sleep(1 * time.Second) // allow temperature to stabilise
	mutex.Unlock()

   maxTempOverall := 0.
   maxSocket := ""
   const limit = 10.0
   const step = 0.1
   const totalSteps = int64(limit / step * ((limit / step) + 1) * ((limit / step) + 1))

   fmt.Fprintf(os.Stderr, "sweeping up to %.1fs over %d steps in %.1fs increments\n", limit, totalSteps, step)

  	for total := 0.; total <= total * 2; total += step {
		for onTime := 0.; onTime <= total; onTime += step {
			offTime := total - onTime

			// heating
			deadline := time.Now().Add(time.Duration(onTime * float64(time.Second)))

			maxTemp := 0.
         interval := 100 * time.Millisecond

         for {
				val, socket := sample(paths)
				if val > maxTemp {
					maxTemp = val
               maxSocket = socket
				}

            // if sleeping for interval puts us past the deadline, sleep only what's needed
            left := time.Until(deadline)
            if left > interval {
               time.Sleep(interval)
            } else {
               time.Sleep(left)
               break
            }
			}

			// cooling
			mutex.Lock()
			time.Sleep(time.Duration(offTime * float64(time.Second)))
			mutex.Unlock()

         fmt.Printf("%.1f/%.1f=%v ", onTime, offTime, maxTemp)
         if maxTemp > maxTempOverall {
            maxTempOverall = maxTemp
            fmt.Fprintf(os.Stderr, "<new max %v at %s with %.1f/%.1f> ", maxTempOverall, maxSocket, onTime, offTime)
         }
		}
	}
	return nil
}

func main() {
	err := top()
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
	}
}
