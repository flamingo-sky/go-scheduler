// This library implements a cron spec parser and runner.  See the README for
// more details.
package scheduler

import (
	"sort"
	"time"
)

type entries []*Entry

// Cron keeps track of any number of entries, invoking the associated func as
// specified by the schedule. It may be started, stopped, and the entries may
// be inspected while running.
type Cron struct {
	entries  entries
	stop     chan struct{}
	add      chan *Entry
	remove   chan string
	snapshot chan entries
	running  bool
}

// Job is an interface for submitted cron jobs.
type Job interface {
	Run()
}

// The Schedule describes a job's duty cycle.
type Schedule interface {
	// Return the next activation time, later than the given time.
	// Next is invoked initially, and then each time the job is run.
	Next(time.Time) time.Time
}

// Entry consists of a schedule and the func to execute on that schedule.
type Entry struct {
	//用户设定的起始时间
	setStartTime time.Time

	//执行周期
	Interval time.Duration
	// started or this entry's schedule is unsatisfiable
	// The next time the job will run. This is the zero time if Cron has not been
	NextTime time.Time

	// The Job to run.
	Job Job

	// Unique name to identify the Entry so as to be able to remove it later.
	Name string
}

// byTime is a wrapper for sorting the entry array by time
// (with zero time at the end).
type byTime []*Entry

func (s byTime) Len() int      { return len(s) }
func (s byTime) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s byTime) Less(i, j int) bool {
	// Two zero times should return false.
	// Otherwise, zero is "greater" than any other time.
	// (To sort it at the end of the list.)
	if s[i].NextTime.IsZero() {
		return false
	}
	if s[j].NextTime.IsZero() {
		return true
	}
	return s[i].NextTime.Before(s[j].NextTime)
}

func (t *Entry) Next() {
	if t.NextTime.IsZero() {
		if t.setStartTime.Before(time.Now()) {
			dur := time.Now().Sub(t.setStartTime)
			cnt := dur.Nanoseconds() / t.Interval.Nanoseconds()
			t.NextTime = t.setStartTime.Add(time.Duration((cnt + 1) * t.Interval.Nanoseconds()))
		} else {
			//t.NextTime = t.setStartTime.Add(t.Interval)
			t.NextTime = t.setStartTime
		}
	} else {
		t.NextTime = t.NextTime.Add(t.Interval)
	}
}

// New returns a new Cron job runner.
func New() *Cron {
	return &Cron{
		entries:  nil,
		add:      make(chan *Entry),
		remove:   make(chan string),
		stop:     make(chan struct{}),
		snapshot: make(chan entries),
		running:  false,
	}
}

// A wrapper that turns a func() into a cron.Job
type FuncJob func()

func (f FuncJob) Run() { f() }

// AddFunc adds a func to the Cron to be run on the given schedule.
func (c *Cron) AddFunc(startTime time.Time, Interval time.Duration, cmd func(), name string) {
	c.AddJob(startTime, Interval, FuncJob(cmd), name)
}

// AddFunc adds a Job to the Cron to be run on the given schedule.
func (c *Cron) AddJob(startTime time.Time, Interval time.Duration, cmd Job, name string) {
	c.Schedule(startTime, Interval, cmd, name)
}

// RemoveJob removes a Job from the Cron based on name.
func (c *Cron) RemoveJob(name string) {
	if !c.running {
		i := c.entries.pos(name)

		if i == -1 {
			return
		}

		c.entries = c.entries[:i+copy(c.entries[i:], c.entries[i+1:])]
		return
	}

	c.remove <- name
}

func (entrySlice entries) pos(name string) int {
	for p, e := range entrySlice {
		if e.Name == name {
			return p
		}
	}
	return -1
}

// Schedule adds a Job to the Cron to be run on the given schedule.
func (c *Cron) Schedule(startTime time.Time, Interval time.Duration, cmd Job, name string) {
	entry := &Entry{
		setStartTime: startTime,
		Interval:     Interval,
		Job:          cmd,
		Name:         name,
	}

	if !c.running {
		i := c.entries.pos(entry.Name)
		if i != -1 {
			c.entries = c.entries[:i+copy(c.entries[i:], c.entries[i+1:])]
		}
		c.entries = append(c.entries, entry)
		return
	}

	c.add <- entry
}

// Entries returns a snapshot of the cron entries.
func (c *Cron) Entries() []*Entry {
	if c.running {
		c.snapshot <- nil
		x := <-c.snapshot
		return x
	}
	return c.entrySnapshot()
}

// Start the cron scheduler in its own go-routine.
func (c *Cron) Start() {
	if c.running == false {
		c.running = true
		go c.run()
	}
}

// Run the scheduler.. this is private just due to the need to synchronize
// access to the 'running' state variable.
func (c *Cron) run() {
	// Figure out the next activation times for each entry.
	now := time.Now().Local()
	for _, entry := range c.entries {
		entry.Next()
	}

	for {
		// Determine the next entry to run.
		sort.Sort(byTime(c.entries))
		var effective time.Time
		if len(c.entries) == 0 || c.entries[0].NextTime.IsZero() {
			// If there are no entries yet, just sleep - it still handles new entries
			// and stop requests.
			effective = now.AddDate(10, 0, 0)
		} else {
			effective = c.entries[0].NextTime
		}

		select {
		case now = <-time.After(effective.Sub(now)):
			// Run every entry whose next time was this effective time.
			for _, e := range c.entries {
				if !e.NextTime.Round(time.Second).Equal(effective.Round(time.Second)) {
					break
				}
				go e.Job.Run()
				e.Next()
			}
			continue

		case newEntry := <-c.add:
			i := c.entries.pos(newEntry.Name)
			if i != -1 {
				c.entries = c.entries[:i+copy(c.entries[i:], c.entries[i+1:])]
			}
			c.entries = append(c.entries, newEntry)
			newEntry.Next()

		case name := <-c.remove:
			i := c.entries.pos(name)

			if i == -1 {
				break
			}

			c.entries = c.entries[:i+copy(c.entries[i:], c.entries[i+1:])]

		case <-c.snapshot:
			c.snapshot <- c.entrySnapshot()

		case <-c.stop:
			return
		}

		// 'now' should be updated after newEntry and snapshot cases.
		now = time.Now().Local()
	}
}

// Stop the cron scheduler.
func (c *Cron) Stop() {
	if c.running == true {
		c.stop <- struct{}{}
		c.running = false
	}
}

// entrySnapshot returns a copy of the current cron entry list.
func (c *Cron) entrySnapshot() []*Entry {
	entries := []*Entry{}
	for _, e := range c.entries {
		entries = append(entries, &Entry{
			setStartTime: e.setStartTime,
			NextTime:     e.NextTime,
			Interval:     e.Interval,
			Job:          e.Job,
			Name:         e.Name,
		})
	}
	return entries
}
