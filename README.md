# sweepload

`sweepload` runs a user-defined workload via a command passed, then stops and runs (SIGSTOP, SIGCONT) the workload for a varying pulse widths to validate system cooling behaviour.

Each varying pulse of workload (on) and sleep (off) time, it prints the maximum temperature from all sockets. This may identify and suboptimal tuning in system fan PID control.

---

## Installation
```bash
go install github.com/dblueman/sweepload@latest
```

---

## Running
```base
~/go/bin/sweepload <bin> [arg] ...
```

It is recommend to use a scalable, affine (compute-bound) OpenMP workload, for example:
```bash
git clone https://github.com/sudden6/m-queens.git
cd m-queens
gcc -std=c99 -march=native -fopenmp -Ofast -o m-queens main.c
~/go/bin/sweepload ./m-queens 20
```

`sweepload` sets appropriate OpenMP environment variables for optimal thread pinning.

---

## Output

`sweepload` performs cycles of workload and rest times, printing loops of `ontime/offtime=temp` where times are in seconds and temperature is the highest socket temperature in Celcius. Time ranges up to 10 seconds are swept by default in 0.1s increments, taking around 8h. After each group of times, a heatmap PDF and SVG file is written for an early look.
