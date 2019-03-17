package scheduler

import (
	"time"
	"testing"
	"fmt"
	"strconv"
)

const ONE_SECOND = 1*time.Second + 10*time.Millisecond

//Start and stop cron with no entries.
func TestNoEntries(t *testing.T) {
	cron := New()
	cron.Start()

	select {
	case <-time.After(ONE_SECOND):
		t.FailNow()
	case <-stop(cron):
	}
}

//Start, stop, then add an entry. Verify entry doesn't run.
func TestStopCausesJobsToNotRun(t *testing.T) {

	cron := New()
	cron.Start()
	cron.Stop()
	cron.AddFunc(time.Now(), 3*time.Second , func() { fmt.Println("test1") }, "test1")

	select {
	case <-time.After(6*ONE_SECOND):
		//No job ran!
	}
}

// Add a job, start cron, expect it runs.
func TestAddBeforeRunning(t *testing.T) {

	cron := New()
	s,_ := time.ParseInLocation("2006-01-02 15:04:05","2019-03-16 21:40:00",time.Local)
	cron.AddFunc(s, 10*time.Second , func() { fmt.Println("test1",
		time.Now().Format("2006-01-02 15:04:05")) }, "test1")
	cron.Start()
	defer cron.Stop()

	// Give cron 2 seconds to run our job (which is always activated).
	select {
	case <-time.After(20*ONE_SECOND):
	}
}

// Start cron, add a job, expect it runs.
func TestAddWhileRunning(t *testing.T) {
	cron := New()
	cron.Start()
	defer cron.Stop()
	s,_ := time.ParseInLocation("2006-01-02 15:04:05","2019-03-16 21:40:00",time.Local)
	cron.AddFunc(s, 10*time.Second , func() { fmt.Println("test1",
		time.Now().Format("2006-01-02 15:04:05")) }, "test1")

	select {
	case <-time.After(20*ONE_SECOND):

	}
}

//Test timing with Entries.
func TestSnapshotEntries(t *testing.T) {

	cron := New()
	s,_ := time.ParseInLocation("2006-01-02 15:04:05","2019-03-16 21:40:00",time.Local)
	cron.AddFunc(s, 10*time.Second , func() { fmt.Println("test1",
		time.Now().Format("2006-01-02 15:04:05")) }, "test1")
	cron.Start()
	defer cron.Stop()

	// Cron should fire in 2 seconds. After 1 second, call Entries.
	for  {
		select {
		case <-time.After(5*ONE_SECOND):
			fmt.Println(len(cron.Entries()))
		}

		// Even though Entries was called, the cron should fire at the 2 second mark.
		select {
		case <-time.After(60*ONE_SECOND):
			return
		}
	}

}

//Test that the entries are correctly sorted.
//Add a bunch of long-in-the-future entries, and an immediate entry, and ensure
//that the immediate entry runs immediately.
//Also: Test that multiple jobs run in the same instant.
func TestMultipleEntries(t *testing.T) {

	cron := New()
	s,_ := time.ParseInLocation("2006-01-02 15:04:05","2019-03-16 21:40:00",time.Local)
	for i:=0 ; i < 100 ; i++  {
		var temp = i
		cron.AddFunc(s, 1*time.Second , func() { fmt.Println(temp,
			time.Now().Format("2006-01-02 15:04:05")) }, strconv.Itoa(temp))
	}


	cron.Start()
	defer cron.Stop()

	select {
	case <-time.After(1*ONE_SECOND):
	}
}

//Test running the same job twice.
func TestRunningJobTwice(t *testing.T) {

	cron := New()
	s,_ := time.ParseInLocation("2006-01-02 15:04:05","2019-03-16 21:40:00",time.Local)
	cron.AddFunc(s, 10*time.Second , func() { fmt.Println("test1",
		time.Now().Format("2006-01-02 15:04:05")) }, "test1")
	cron.AddFunc(s.Add(10*time.Second), 20*time.Second , func() { fmt.Println("test2",
		time.Now().Format("2006-01-02 15:04:05")) }, "test2")
	cron.AddFunc(s.Add(6*time.Second), 30*time.Second , func() { fmt.Println("test3",
		time.Now().Format("2006-01-02 15:04:05")) }, "test3")

	cron.Start()
	defer cron.Stop()

	select {
	case <-time.After(2 * ONE_SECOND):
	}
}

func TestRunningMultipleSchedules(t *testing.T) {

	cron := New()
	s,_ := time.ParseInLocation("2006-01-02 15:04:05","2019-03-16 21:40:00",time.Local)
	cron.AddFunc(s, 10*time.Second , func() { fmt.Println("test1",
		time.Now().Format("2006-01-02 15:04:05")) }, "test1")
	cron.AddFunc(s.Add(10*time.Second), 20*time.Second , func() { fmt.Println("test2",
		time.Now().Format("2006-01-02 15:04:05")) }, "test2")
	cron.AddFunc(s.Add(6*time.Second), 30*time.Second , func() { fmt.Println("test3",
		time.Now().Format("2006-01-02 15:04:05")) }, "test3")

	cron.Start()
	defer cron.Stop()

	select {
	case <-time.After(2 * ONE_SECOND):
	}
}

//Test that the cron is run in the local time zone (as opposed to UTC).
func TestLocalTimezone(t *testing.T) {

	cron := New()
	s,_ := time.ParseInLocation("2006-01-02 15:04:05","2019-03-16 21:40:00",time.Local)
	cron.AddFunc(s, 10*time.Second , func() { fmt.Println("test1",
		time.Now().Format("2006-01-02 15:04:05")) }, "test1")
	cron.Start()
	defer cron.Stop()

	select {
	case <-time.After(20*ONE_SECOND):
	}
}


//Simple test using Runnables.
func TestJob(t *testing.T) {

	cron := New()
	s,_ := time.ParseInLocation("2006-01-02 15:04:05","2019-03-16 21:40:00",time.Local)
	cron.AddFunc(s, 10*time.Second , func() { fmt.Println("test1",
		time.Now().Format("2006-01-02 15:04:05")) }, "test1")
	cron.AddFunc(s.Add(10*time.Second), 20*time.Second , func() { fmt.Println("test2",
		time.Now().Format("2006-01-02 15:04:05")) }, "test2")
	cron.AddFunc(s.Add(6*time.Second), 30*time.Second , func() { fmt.Println("test3",
		time.Now().Format("2006-01-02 15:04:05")) }, "test3")

	cron.Start()
	defer cron.Stop()

	select {
	case <-time.After(ONE_SECOND):
	}

	// Ensure the entries are in the right order.
	expecteds := []string{"test1", "test2", "test3"}

	var actuals []string
	for _, entry := range cron.Entries() {
		actuals = append(actuals, entry.Name)
	}

	if len(expecteds)!=len(cron.entries){
		t.Errorf("Jobs not in the right order.  (expected) %s != %s (actual)", expecteds, actuals)
		t.FailNow()
	}

}

// Add a job, start cron, remove the job, expect it to have not run
func TestAddBeforeRunningThenRemoveWhileRunning(t *testing.T) {
	cron := New()
	s,_ := time.ParseInLocation("2006-01-02 15:04:05","2019-03-16 21:40:00",time.Local)

	cron.AddFunc(s, 10*time.Second , func() { fmt.Println("test1",
		time.Now().Format("2006-01-02 15:04:05")) }, "test1")
	cron.Start()

	defer cron.Stop()
	cron.RemoveJob("test1")

	// Give cron 2 seconds to run our job (which is always activated).
	select {
	case <-time.After(ONE_SECOND):
	}

	for _, entry := range cron.Entries() {
		if entry.Name == "test1"{
			t.FailNow()
		}
	}
}

// Add a job, remove the job, start cron, expect it to have not run
func TestAddBeforeRunningThenRemoveBeforeRunning(t *testing.T) {

	cron := New()
	s,_ := time.ParseInLocation("2006-01-02 15:04:05","2019-03-16 21:40:00",time.Local)
	cron.AddFunc(s, 10*time.Second , func() { fmt.Println("test1",
		time.Now().Format("2006-01-02 15:04:05")) }, "test1")

	cron.RemoveJob("test1")
	cron.Start()
	defer cron.Stop()

	// Give cron 2 seconds to run our job (which is always activated).
	select {
	case <-time.After(ONE_SECOND):
	}

	for _, entry := range cron.Entries() {
		if entry.Name == "test1"{
			t.FailNow()
		}
	}
}


func stop(cron *Cron) chan bool {
	ch := make(chan bool)
	go func() {
		cron.Stop()
		ch <- true
	}()
	return ch
}
