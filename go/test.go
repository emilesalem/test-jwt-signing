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

func main() {

	// duration of the test
	elapsed := 2 * time.Second

	deadline := time.Now().Add(elapsed)

	mainCtx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	var w sync.WaitGroup

	w.Add(1)
	jobs := producer(mainCtx, &w)

	doneJobs := make(chan time.Duration)

	w.Add(1)
	go func() {
		jobsDone := 0
		var avgWorkTime time.Duration
		for {
			select {
			case timeWorked := <-doneJobs:
				jobsDone++
				avgWorkTime = time.Duration(int64((float64(avgWorkTime)*float64(jobsDone-1) + float64(timeWorked)) / float64(jobsDone)))
			case <-mainCtx.Done():
				fmt.Printf("signed requests: %v\n", jobsDone)
				fmt.Printf("average time to sign: %vmicroseconds\n", avgWorkTime.Microseconds())
				w.Done()
				return
			}
		}
	}()
	for range jobs {
		go func() {
			s := time.Now()
			sign()
			t := time.Since(s)
			doneJobs <- t
		}()
	}
	w.Wait()
}
