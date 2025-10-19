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

	"gonum.org/v1/gonum/mat"
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

	const limit = 8
	const steps = 8

	heatmap := Heatmap{
		Data: mat.NewDense(limit*steps, limit*steps, nil),
	}

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

	fmt.Fprintf(os.Stderr, "sweeping up to %.1fs over %d steps in %.1fs increments\n", float64(limit)/steps, steps, 1/float64(steps))

	maxTempOverall, err := getTjMax(0)
	if err != nil {
		return err
	}

	fmt.Fprint(os.Stderr, "waiting for thermal equilibrium...")
	time.Sleep(100 * time.Millisecond)

	err = cmd.Process.Signal(syscall.SIGSTOP)
	if err != nil {
		if errors.Is(err, os.ErrProcessDone) {
			return errors.New("workload exited prematurely; check arguments")
		}

		return err
	}

	time.Sleep(4 * time.Second)
	minTempOverall, _ := sample(time.Now().Add(100 * time.Millisecond))
	minTempOverall -= 3 // add margin
	fmt.Fprintf(os.Stderr, "range %d-%d'C (tjMax)\n", minTempOverall, maxTempOverall)

	for offTime := range limit * steps {
		for onTime := range limit * steps {
		again:
			stopDeadline := time.Now().Add(time.Duration(float64(onTime) / steps * float64(time.Second)))

			err = cmd.Process.Signal(syscall.SIGCONT)
			if err != nil {
				if !errors.Is(err, os.ErrProcessDone) {
					return err
				}
				fmt.Fprintf(os.Stderr, "<relaunching workload>")
				cmd, err = launch(args)
				if err != nil {
					return err
				}
				time.Sleep(time.Second) // allow cooling
				goto again
			}

			maxTemp, _ := sample(stopDeadline)

			err = cmd.Process.Signal(syscall.SIGSTOP)
			// FIXME address duplication
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

			time.Sleep(time.Duration(float64(offTime) / steps * float64(time.Second)))
			fmt.Printf("%.1f/%.1f=%d ", float64(onTime)/steps, float64(offTime)/steps, maxTemp)

			heatmap.Data.Set(onTime, offTime, float64(maxTemp-minTempOverall))
		}

		err = heatmap.Render(float64(minTempOverall), float64(maxTempOverall), limit*steps, limit*steps, steps,
			"System temperature under pulsed workloads ('C)",
			"idle time (s)",
			"compute time (s)",
			"heatmap.pdf")
		if err != nil {
			return err
		}

		// render at each row for early results
		fmt.Fprint(os.Stderr, "<updated heatmap.pdf>")
		time.Sleep(time.Second) // allow cooling
	}

	return nil
}

func main() {
	err := top()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}
