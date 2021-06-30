package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
)

func sign() {
	jwt.NewWithClaims(jwt.SigningMethodHS256, &jwt.StandardClaims{
		Issuer: "bar",
	}).SignedString([]byte("shhhhh"))
}

func producer(ctx context.Context, w *sync.WaitGroup) chan struct{} {
	jobs := make(chan struct{})

	go func() {
		requests := 0
		for {
			select {
			case <-time.Tick(1 * time.Millisecond):
				requests++
				jobs <- struct{}{}
			case <-ctx.Done():
				fmt.Printf("jobs requested: %v\n", requests)
				close(jobs)
				w.Done()
				return
			}
		}
	}()
	return jobs
}

func monitorJobs(ctx context.Context, doneJobs chan time.Duration, w *sync.WaitGroup) {
	var jobsDone int64
	var avgWorkTime int64
	for {
		select {
		case timeWorked := <-doneJobs:
			jobsDone++
			avgWorkTime = int64((float64(avgWorkTime*(jobsDone-1) + int64(timeWorked))) / float64(jobsDone))
		case <-ctx.Done():
			fmt.Printf("signed requests: %v\n", jobsDone)
			fmt.Printf("average time to sign: %s\n", time.Duration(avgWorkTime))
			w.Done()
			return
		}
	}
}

func main() {

	// duration of the test
	elapsed := 2 * time.Second

	deadline := time.Now().Add(elapsed)

	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	var w sync.WaitGroup

	w.Add(1)
	jobs := producer(ctx, &w)

	doneJobs := make(chan time.Duration)

	w.Add(1)
	go monitorJobs(ctx, doneJobs, &w)

	for range jobs {
		go func() {
			s := time.Now()
			sign()
			t := time.Since(s)
			doneJobs <- t
		}()
	}
	close(doneJobs)
	w.Wait()
}
