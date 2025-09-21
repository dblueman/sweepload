package main

import (
	"fmt"
	"math/rand/v2"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
//	"syscall"
	"time"
	"golang.org/x/sys/unix"

	"gonum.org/v1/gonum/mat"
)

const (
   dimension = 100
)

var (
	mutex      = sync.RWMutex{}
   maxTemp    atomic.Int32
	tjMax      int
)

func pin(core int) {
	runtime.LockOSThread()

   var set unix.CPUSet
   set.Set(core)
   err := unix.SchedSetaffinity(0, &set)
   if err != nil {
      panic(err)
   }
}

/*func priority(prio int) {
	unix.SchedSetAttr()
tid := syscall.Gettid()
	param := &unix.SchedParam{SchedPriority: prio}

	err := unix.SchedSetscheduler(tid, unix.SCHED_FIFO, param)
	if err != nil {
		panic(err)
	}
}*/

func worker(core int) {
	pin(core)
//	priority(3)

	tjMax2, err := getTjMax(core)
	if err != nil {
		panic("failed to get tjMax")
	}

	if tjMax2 != tjMax {
		panic("core junction temperature disagree")
	}

	m := mat.NewDense(dimension, dimension, nil)

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

		// track temperature
		temp, err := getTemp(core, IA32_THERM_STATUS)
		if err != nil {
			panic("failed to read IA32_THERM_STATUS")
		}

		pkgTemp, err := getTemp(core, IA32_PACKAGE_THERM_STATUS)
		if err != nil {
			panic("failed to read IA32_PACKAGE_THERM_STATUS")
		}

		if pkgTemp > temp {
			temp = pkgTemp
		}
again:
		maxTempLocal := maxTemp.Load()
		if int32(temp) > maxTempLocal {
			if !maxTemp.CompareAndSwap(maxTempLocal, int32(temp)) {
				goto again
			}
		}

		mutex.RUnlock()
	}
}

func top() error {
	pin(0)

	var err error
	tjMax, err = getTjMax(0)
	if err != nil {
		panic("failed to get tjMax")
	}

	cores := runtime.NumCPU()
	mutex.Lock()

	for c := 1; c < cores; c++ {
		go worker(c)
	}

   fmt.Fprintln(os.Stderr, "%v threads started; waiting for thermal equilibrium...", cores)
   time.Sleep(2 * time.Second)
	fmt.Fprintln(os.Stderr, "done")

   var maxTempOverall int32
   const limit = 10.0
   const step = 0.1
   const totalSteps = int64(limit / step * ((limit / step) + 1) * ((limit / step) + 1))

   fmt.Fprintf(os.Stderr, "sweeping up to %.1fs over %d steps in %.1fs increments\n", limit, totalSteps, step)

  	for total := 0.; total <= total * 2; total += step {
		for onTime := 0.; onTime <= total; onTime += step {
			offTime := total - onTime

			maxTemp.Store(0)
			mutex.Unlock() // workers working -> heating
			time.Sleep(time.Duration(onTime * float64(time.Second)))
			mutex.Lock()

			time.Sleep(time.Duration(offTime * float64(time.Second))) // workers paused -> cooling
			fmt.Printf("%.1f/%.1f=%v ", onTime, offTime, maxTemp.Load())

         if maxTemp.Load() > maxTempOverall {
            maxTempOverall = maxTemp.Load()
            fmt.Fprintf(os.Stderr, "<new max %v with %.1f/%.1f> ", maxTempOverall, onTime, offTime)
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
