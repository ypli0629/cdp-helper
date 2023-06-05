package cdp_helper

import (
	"sync"
	"time"
)

type Arg map[string]any

type Job interface {
	Prev() ([]Arg, bool)
	Do(arg Arg) bool
	Post(args *[]Arg)
}

type Scheduler struct {
	args              []Arg
	PrevRetry         bool          // whether retry in prev phase
	PrevRetryTimes    int           // prev phase retry times
	PrevRetryInterval time.Duration // prev phase retry interval
	ErrRetry          bool          // whether retry when error occur
	ErrRetryTimes     int           // retry times when error occur
	Concurrent        bool          // execute do function concurrently
	WaitStep          int
	Done              chan any
	step              int
	errArgs           []Arg
	Timeout           time.Duration
}

func (scheduler *Scheduler) Schedule(job Job) bool {
	// prev phase
	args, ok := job.Prev()
	if !ok {
		if scheduler.PrevRetry {
			for i := 0; i < scheduler.PrevRetryTimes; i++ {
				args, ok = job.Prev()
				if ok {
					goto run
				} else {
					time.Sleep(scheduler.PrevRetryInterval)
				}
			}
			return false
		}
	}
	// do phase
run:
	var stepWaitGroup sync.WaitGroup
	for _, arg := range args {
		if scheduler.Concurrent {
			stepWaitGroup.Add(1)
			go func() {
				defer func() {
					stepWaitGroup.Done()
				}()
				ok = runJob(job, arg)
				if !ok {
					if scheduler.ErrRetry {
						// record err arg, used to retry
						scheduler.errArgs = append(scheduler.errArgs, arg)
					}
				}
			}()

			// wait by step
			scheduler.step++
			if scheduler.step%scheduler.WaitStep == 0 {
				stepWaitGroup.Wait()
			}
		} else {
			ok = runJob(job, arg)
			if !ok {
				if scheduler.ErrRetry {
					// record err arg, used to retry
					scheduler.errArgs = append(scheduler.errArgs, arg)
				}
			}
		}
	}

	// do until timeout
	go func() {
		stepWaitGroup.Wait()
		scheduler.Done <- struct{}{}
	}()

	select {
	case <-scheduler.Done:
	case <-time.NewTimer(scheduler.Timeout).C:
	}

	// error retry
	if scheduler.ErrRetry {
		for i := 0; i < scheduler.ErrRetryTimes; i++ {
			executedArgs := make([]Arg, len(scheduler.errArgs))
			copy(executedArgs, scheduler.errArgs)
			scheduler.errArgs = scheduler.errArgs[0:0]
			for _, arg := range executedArgs {
				ok = runJob(job, arg)
				if !ok {
					scheduler.errArgs = append(scheduler.errArgs, arg)
				}
			}
		}
	}

	// post phase
	job.Post(&scheduler.args)
	return true
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		args:              nil,
		PrevRetry:         true,
		PrevRetryTimes:    3,
		PrevRetryInterval: 3 * time.Second,
		Concurrent:        true,
		WaitStep:          3,
		Done:              make(chan any),
		Timeout:           60 * time.Second,
		ErrRetry:          true,
		ErrRetryTimes:     3,
	}
}

func runJob(job Job, arg Arg) (result bool) {
	defer func() {
		if err := recover(); err != nil {
			result = false
			return
		}
	}()

	result = job.Do(arg)
	return
}
