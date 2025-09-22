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
