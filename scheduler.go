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
	PrevRetry         bool
	PrevRetryTimes    int
	PrevRetryInterval time.Duration
	ErrRetry          bool
	ErrRetryTimes     int
	Concurrent        bool
	WaitStep          int
	Done              chan any
	step              int
	errArgs           []Arg
	Timeout           time.Duration
}

func (scheduler *Scheduler) schedule(job Job) bool {
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
	stepWaitGroup.Wait()

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
