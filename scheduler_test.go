package cdp_helper

import (
	"log"
	"math/rand"
	"testing"
	"time"
)

type testJob struct {
}

func (j *testJob) Prev() ([]Arg, bool) {
	s := make([]Arg, 0)
	for i := 0; i < 100; i++ {
		s = append(s, Arg{
			"id": i,
		})
	}
	return s, true
}

func (j *testJob) Do(arg Arg) bool {
	n := rand.Int()
	if n%2 == 0 {
		log.Panicf("panic, %v", arg)
	} else {
		log.Print(arg)
		time.Sleep(100 * time.Millisecond)
	}

	return true
}

func (j *testJob) Post(args *[]Arg) {
	//TODO implement me
}

func TestScheduler_schedule(t *testing.T) {
	sch := NewScheduler()
	var job testJob
	sch.Schedule(&job)
}
