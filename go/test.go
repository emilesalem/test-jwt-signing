package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt"
)

func sign() {
	jwt.NewWithClaims(jwt.SigningMethodHS256, &jwt.StandardClaims{
		Issuer: "bar",
	}).SignedString([]byte("shhhhh"))
}

func producer(ctx context.Context, w *sync.WaitGroup) chan struct{} {
	jobs := make(chan struct{}, 2000000)
	go func() {
		requests := 0
		t := time.Tick(1 * time.Microsecond)
		for {
			select {
			case <-t:
				requests++
				jobs <- struct{}{}
			case <-ctx.Done():
				fmt.Printf("signatures requested: %v\n", requests)
				close(jobs)
				w.Done()
				return
			}
		}
	}()
	return jobs
}

func monitorJobs(ctx context.Context, doneJobs chan time.Duration) {
	var jobsDone int64
	var avgWorkTime int64
	for {
		select {
		case timeWorked := <-doneJobs:
			jobsDone++
			avgWorkTime = int64((float64(avgWorkTime*(jobsDone-1) + int64(timeWorked))) / float64(jobsDone))
		case <-ctx.Done():
			for range doneJobs {
				//empty doneJobs channel to unblock workers
			}
			fmt.Printf("requests signed: %v\n", jobsDone)
			fmt.Printf("average time to sign: %s\n", time.Duration(avgWorkTime))
			return
		}
	}
}

func worker(ctx context.Context, jobs chan struct{}, doneJobs chan time.Duration) {
	for {
		select {
		case <-jobs:
			s := time.Now()
			sign()
			t := time.Since(s)
			doneJobs <- t
		case <-ctx.Done():
			for range jobs {
				//empty jobs channel to unblock job producer
			}
			close(doneJobs)
			return
		}
	}
}

func main() {
	start := time.Now()
	// duration of the test
	elapsed := 2 * time.Second

	deadline := time.Now().Add(elapsed)

	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	var w sync.WaitGroup

	w.Add(1)
	jobs := producer(ctx, &w)

	doneJobs := make(chan time.Duration)

	nbWorkers := 1
	for i := 0; i < nbWorkers; i++ {
		go worker(ctx, jobs, doneJobs)
	}
	monitorJobs(ctx, doneJobs)

	w.Wait()

	fmt.Printf("number of signing goroutines: %v\n", nbWorkers)

	fmt.Printf("elapsed time %s\n", time.Since(start))
}
