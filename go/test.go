package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
)

type Worker struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func (w Worker) sign() {
	jwt.NewWithClaims(jwt.SigningMethodHS256, &jwt.StandardClaims{
		Issuer: "bar",
	}).SignedString([]byte("shhhhh"))
}

type WorkerPool struct {
	work             chan struct{}
	workers          chan Worker
	remove           chan struct{}
	workMonitor      chan int
	workDone         chan int64
	avgWorkTime      float64
	worked           int64
	inflightRequests int
	totalRequested   int
	workForce        []Worker
	removedWorkers   int
	addedWorkers     int
	maxPressure      int
}

func createWorkerPool() *WorkerPool {

	return &WorkerPool{
		work:             make(chan struct{}),
		workers:          make(chan Worker),
		workMonitor:      make(chan int),
		remove:           make(chan struct{}),
		workDone:         make(chan int64),
		workForce:        make([]Worker, 0, 1000),
		avgWorkTime:      0,
		worked:           0,
		inflightRequests: 0,
		totalRequested:   0,
		removedWorkers:   0,
		addedWorkers:     0,
		maxPressure:      5000,
	}
}

func (w *WorkerPool) addWork() {
	w.work <- struct{}{}
	w.workMonitor <- 1
}

func (w *WorkerPool) addWorker() {
	ctx, cancel := context.WithCancel(context.Background())
	worker := Worker{ctx, cancel}
	w.workers <- worker
	w.workForce = append(w.workForce, worker)
	w.addedWorkers++
}

func (w *WorkerPool) removeWorker() {
	if len(w.workForce) <= 1 {
		return
	}
	worker := w.workForce[0]
	w.workForce = w.workForce[1:]
	worker.cancel()
	w.removedWorkers++
}

func (w *WorkerPool) countWork() {
	for x := range w.workDone {
		w.worked++
		w.avgWorkTime = (w.avgWorkTime*float64(w.worked-1) + float64(x)) / float64(w.worked)
		w.workMonitor <- -1
	}
}

func (w *WorkerPool) monitorWork() {
	for x := range w.workMonitor {
		w.inflightRequests += x
	}
}

func (w *WorkerPool) start() {
	go w.addWorker()
	go w.checkWorkRate()
	go w.countWork()
	go w.monitorWork()
	for worker := range w.workers {
		go func(worker Worker) {
		loop:
			for {
				select {
				case <-w.work:
					start := time.Now()
					worker.sign()
					w.workDone <- time.Since(start).Nanoseconds()
				case <-worker.ctx.Done():
					break loop
				}
			}
		}(worker)
	}
}

func (w *WorkerPool) checkWorkRate() {
	for range time.Tick(time.Millisecond) {
		if w.inflightRequests > w.maxPressure {
			w.addWorker()
		} else {
			w.removeWorker()
		}
	}
}

func main() {
	// rate of signing requests per milliseconds
	milliRate := 1

	// duration of the test
	elapsed := 2 * time.Second

	p := createWorkerPool()

	go p.start()

	end := time.After(elapsed)
	for range time.Tick(1 * time.Millisecond) {
		select {
		case <-end:
			fmt.Printf("signed %v times in %v seconds, requested %v signatures\naverage work time: %v nanoseconds\n", p.worked, elapsed, p.totalRequested, int(p.avgWorkTime))
			fmt.Printf("had to add %v workers and remove %v\n", p.addedWorkers, p.removedWorkers)
			os.Exit(0)
		default:
			for i := 0; i < milliRate; i++ {
				go p.addWork()
				p.totalRequested++
			}
		}
	}
}
