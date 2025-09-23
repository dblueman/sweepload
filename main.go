package main

// To use this, run: go get gonum.org/v1/plot/...
import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	sampleInterval = 10 * time.Millisecond
)

var (
	paths []string
)

func sample(deadline time.Time) (int, int) {
	maxTemp := 0
	maxSocket := -1
	once := false

	for {
		for n, path := range paths {
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
				maxSocket = n
			}

			left := time.Until(deadline)
			time.Sleep(left)

			if left < sampleInterval && once {
				return maxTemp / 1000, maxSocket
			}
		}

		once = true
	}
}

func launch(args []string) (*exec.Cmd, error) {
	cmd := exec.Command(args[0], args[1:]...)
	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	// reap process when exited; causes signal to fail
	go func() {
		err := cmd.Wait()
		if err != nil {
			panic(err)
		}
	}()

	return cmd, nil
}

func top() error {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: sweepload <workload> [args] ...")
		fmt.Fprintln(os.Stderr, "ensure no background activity eg via: systemctl stop fwupd irqbalance tuned")
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(2)
	}

	for s := 0; true; s++ {
		path := "/sys/class/thermal/thermal_zone" + strconv.Itoa(s) + "/temp"
		_, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				break
			} else {
				return err
			}
		}
		paths = append(paths, path)
	}

	const limit = 10.
	const step = 0.1
	const totalSteps = int64(limit / step * ((limit / step) + 1) * ((limit / step) + 1))

	heatmap := NewHeatmap(int(limit/step)+1, int(limit/step)+1)

	// schedule 1 less thread for parent monitoring
	os.Setenv("OMP_NUM_THREADS", strconv.Itoa(runtime.NumCPU()-1))
	os.Setenv("OMP_PROC_BIND", "true")
	os.Setenv("OMP_WAIT_POLICY", "active")

	args := flag.Args()
	cmd, err := launch(args)
	if err != nil {
		return err
	}

	// prevent scheduling latency on control process
	err = syscall.Setpriority(syscall.PRIO_PROCESS, cmd.Process.Pid, 5)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "sweeping up to %.1fs over %d steps in %.1fs increments\n", limit, totalSteps, step)

	tjMax, err := getTjMax(0)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "tjMax=%d'C; waiting for thermal equilibrium...")

	time.Sleep(100 * time.Millisecond)

	err = cmd.Process.Signal(syscall.SIGSTOP)
	if err != nil {
		return err
	}

	time.Sleep(2 * time.Second)
	minTempOverall := tjMax
	maxTempOverall := 0

	for total := 0.; total <= limit; total += step {
	again:
		for onTime := 0.; onTime <= total; onTime += step {
			offTime := total - onTime
			stopDeadline := time.Now().Add(time.Duration(onTime * float64(time.Second)))

			err = cmd.Process.Signal(syscall.SIGCONT)
			if err != nil {
				return err
			}

			maxTemp, socket := sample(stopDeadline)

			if maxTemp > maxTempOverall {
				maxTempOverall = maxTemp
				fmt.Fprintf(os.Stderr, "<new max %v with %.1f/%.1f on S%d> ", maxTempOverall, onTime, offTime, socket)
			}

			if maxTemp < minTempOverall {
				minTempOverall = maxTemp
			}

			err = cmd.Process.Signal(syscall.SIGSTOP)
			if err != nil {
				if !errors.Is(err, os.ErrProcessDone) {
					return err
				}
				fmt.Fprintf(os.Stderr, "<relaunching workload>")
				cmd, err = launch(args)
				if err != nil {
					return err
				}
				goto again
			}

			time.Sleep(time.Duration(offTime * float64(time.Second)))
			fmt.Printf("%.1f/%.1f=%d ", onTime, offTime, maxTemp)
			heatmap.Set(int(onTime/step), int(offTime/step), float64(maxTemp))
		}

		// render at each row for early results
		err = heatmap.Render(float64(minTempOverall), float64(maxTempOverall), "heatmap.pdf")
		if err != nil {
			return err
		}
		fmt.Fprint(os.Stderr, "<updated heatmap.pdf>")
	}

	return nil
}

func main() {
	err := top()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}
