/*
 *
 * k6 - a next-generation load testing tool
 * Copyright (C) 2019 Load Impact
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package lib

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"go.k6.io/k6/lib/metrics"
	"go.k6.io/k6/stats"
)

// An ExecutionScheduler is in charge of initializing executors and using them
// to initialize and schedule VUs created by a wrapped Runner. It decouples how
// a swarm of VUs is controlled from the details of how or even where they're
// scheduled.
//
// The core/local execution scheduler schedules VUs on the local machine, but
// the same interface may be implemented to control a test running on a cluster
// or in the cloud.
//
// TODO: flesh out the interface after actually having more than one
// implementation...
type ExecutionScheduler interface {
	// Returns the wrapped runner. May return nil if not applicable, eg.
	// if we're remote controlling a test running on another machine.
	GetRunner() Runner

	// Return the ExecutionState instance from which different statistics for the
	// current state of the runner could be retrieved.
	GetState() *ExecutionState

	// Return the instances of the configured executors
	GetExecutors() []Executor

	// Init initializes all executors, including all of their needed VUs.
	Init(ctx context.Context, samplesOut chan<- stats.SampleContainer) error

	// Run the ExecutionScheduler, funneling the generated metric samples
	// through the supplied out channel.
	Run(
		globalCtx, runCtx context.Context, samplesOut chan<- stats.SampleContainer,
		builtinMetrics *metrics.BuiltinMetrics,
	) error

	// Pause a test, or start/resume it. To check if a test is paused, use
	// GetState().IsPaused().
	//
	// Currently, any executor, so any test, can be started in a paused state.
	// This will cause k6 to initialize all needed VUs, but it won't actually
	// start the test. Later, the test can be started for real by
	// resuming/unpausing it from the REST API.
	//
	// After a test is actually started, it may become impossible to pause it
	// again. That is denoted by having SetPaused(true) return an error. The
	// likely cause is that some of the executors for the test don't support
	// pausing after the test has been started.
	//
	// IMPORTANT: Currently only the externally controlled executor can be
	// paused and resumed multiple times in the middle of the test execution!
	// Even then, "pausing" is a bit misleading, since k6 won't pause in the
	// middle of the currently executing iterations. It will allow the currently
	// in progress iterations to finish, and it just won't start any new ones
	// nor will it increment the value returned by GetCurrentTestRunDuration().
	SetPaused(paused bool) error
}

// MaxTimeToWaitForPlannedVU specifies the maximum allowable time for an executor
// to wait for a planned VU to be retrieved from the ExecutionState.PlannedVUs
// buffer. If it's exceeded, k6 will emit a warning log message, since it either
// means that there's a bug in the k6 scheduling code, or that the machine is
// overloaded and the scheduling code suffers from delays.
//
// Critically, exceeding this time *doesn't* result in an aborted test or any
// test errors, and the executor will continue to try and borrow the VU
// (potentially resulting in further warnings). We likely should emit a k6
// metric about it in the future. TODO: emit a metric every time this is
// exceeded?
const MaxTimeToWaitForPlannedVU = 400 * time.Millisecond

// MaxRetriesGetPlannedVU how many times we should wait for
// MaxTimeToWaitForPlannedVU before we actually return an error.
const MaxRetriesGetPlannedVU = 5

// ExecutionStatus is similar to RunStatus, but more fine grained and concerns
// only local execution.
//go:generate enumer -type=ExecutionStatus -trimprefix ExecutionStatus -output execution_status_gen.go
type ExecutionStatus uint32

// Possible execution status values
const (
	ExecutionStatusCreated ExecutionStatus = iota
	ExecutionStatusInitVUs
	ExecutionStatusInitExecutors
	ExecutionStatusInitDone
	ExecutionStatusPausedBeforeRun
	ExecutionStatusStarted
	ExecutionStatusSetup
	ExecutionStatusRunning
	ExecutionStatusTeardown
	ExecutionStatusEnded
	ExecutionStatusInterrupted
)

// ExecutionState contains a few different things:
//  -  Some convenience items, that are needed by all executors, like the
//     execution segment and the unique VU ID generator. By keeping those here,
//     we can just pass the ExecutionState to the different executors, instead of
//     individually passing them each item.
//  -  Mutable counters that different executors modify and other parts of
//     k6 can read, e.g. for the vus and vus_max metrics k6 emits every second.
//  -  Pausing controls and statistics.
//
// The counters and timestamps here are primarily meant to be used for
// information extraction and avoidance of ID collisions. Using many of the
// counters here for synchronization between VUs could result in HIDDEN data
// races, because the Go data race detector can't detect any data races
// involving atomics...
//
// The only functionality intended for synchronization is the one revolving
// around pausing, and uninitializedUnplannedVUs for restricting the number of
// unplanned VUs being initialized.
type ExecutionState struct {
	resumeNotify               chan struct{}
	ExecutionTuple             *ExecutionTuple
	vus                        chan InitializedVU
	vuIDSegIndexMx             *sync.Mutex
	vuIDSegIndex               *SegmentedIndex
	initializedVUs             *int64
	uninitializedUnplannedVUs  *int64
	initVUFunc                 InitVUFunc
	activeVUs                  *int64
	fullIterationsCount        *uint64
	interruptedIterationsCount *uint64
	executionStatus            *uint32
	startTime                  *int64
	endTime                    *int64
	currentPauseTime           *int64
	Options                    Options
	totalPausedDuration        time.Duration
	pauseStateLock             sync.RWMutex
}

// NewExecutionState initializes all of the pointers in the ExecutionState
// with zeros. It also makes sure that the initial state is unpaused, by
// setting resumeNotify to an already closed channel.
func NewExecutionState(options Options, et *ExecutionTuple, maxPlannedVUs, maxPossibleVUs uint64) *ExecutionState {
	resumeNotify := make(chan struct{})
	close(resumeNotify) // By default the ExecutionState starts unpaused

	maxUnplannedUninitializedVUs := int64(maxPossibleVUs - maxPlannedVUs)

	segIdx := NewSegmentedIndex(et)
	return &ExecutionState{
		Options: options,
		vus:     make(chan InitializedVU, maxPossibleVUs),

		executionStatus:            new(uint32),
		vuIDSegIndexMx:             new(sync.Mutex),
		vuIDSegIndex:               segIdx,
		initializedVUs:             new(int64),
		uninitializedUnplannedVUs:  &maxUnplannedUninitializedVUs,
		activeVUs:                  new(int64),
		fullIterationsCount:        new(uint64),
		interruptedIterationsCount: new(uint64),
		startTime:                  new(int64),
		endTime:                    new(int64),
		currentPauseTime:           new(int64),
		pauseStateLock:             sync.RWMutex{},
		totalPausedDuration:        0, // Accessed only behind the pauseStateLock
		resumeNotify:               resumeNotify,
		ExecutionTuple:             et,
	}
}

// GetUniqueVUIdentifiers returns the next unique VU IDs, both local (for the
// current instance, exposed as __VU) and global (across k6 instances, exposed
// in the k6/execution module). It starts from 1, for backwards compatibility.
func (es *ExecutionState) GetUniqueVUIdentifiers() (uint64, uint64) {
	es.vuIDSegIndexMx.Lock()
	defer es.vuIDSegIndexMx.Unlock()
	scaled, unscaled := es.vuIDSegIndex.Next()
	return uint64(scaled), uint64(unscaled)
}

// GetInitializedVUsCount returns the total number of currently initialized VUs.
//
// Important: this doesn't include any temporary/service VUs that are destroyed
// after they are used. These are created for the initial retrieval of the
// exported script options and for the execution of setup() and teardown()
//
// IMPORTANT: for UI/information purposes only, don't use for synchronization.
func (es *ExecutionState) GetInitializedVUsCount() int64 {
	return atomic.LoadInt64(es.initializedVUs)
}

// ModInitializedVUsCount changes the total number of currently initialized VUs.
//
// IMPORTANT: for UI/information purposes only, don't use for synchronization.
func (es *ExecutionState) ModInitializedVUsCount(mod int64) int64 {
	return atomic.AddInt64(es.initializedVUs, mod)
}

// GetCurrentlyActiveVUsCount returns the number of VUs that are currently
// executing the test script. This also includes any VUs that are in the process
// of gracefully winding down.
//
// IMPORTANT: for UI/information purposes only, don't use for synchronization.
func (es *ExecutionState) GetCurrentlyActiveVUsCount() int64 {
	return atomic.LoadInt64(es.activeVUs)
}

// ModCurrentlyActiveVUsCount changes the total number of currently active VUs.
//
// IMPORTANT: for UI/information purposes only, don't use for synchronization.
func (es *ExecutionState) ModCurrentlyActiveVUsCount(mod int64) int64 {
	return atomic.AddInt64(es.activeVUs, mod)
}

// GetFullIterationCount returns the total of full (i.e uninterrupted) iterations
// that have been completed so far.
//
// IMPORTANT: for UI/information purposes only, don't use for synchronization.
func (es *ExecutionState) GetFullIterationCount() uint64 {
	return atomic.LoadUint64(es.fullIterationsCount)
}

// AddFullIterations increments the number of full (i.e uninterrupted) iterations
// by the provided amount.
//
// IMPORTANT: for UI/information purposes only, don't use for synchronization.
func (es *ExecutionState) AddFullIterations(count uint64) uint64 {
	return atomic.AddUint64(es.fullIterationsCount, count)
}

// GetPartialIterationCount returns the total of partial (i.e interrupted)
// iterations that have been completed so far.
//
// IMPORTANT: for UI/information purposes only, don't use for synchronization.
func (es *ExecutionState) GetPartialIterationCount() uint64 {
	return atomic.LoadUint64(es.interruptedIterationsCount)
}

// AddInterruptedIterations increments the number of partial (i.e interrupted)
// iterations by the provided amount.
//
// IMPORTANT: for UI/information purposes only, don't use for synchronization.
func (es *ExecutionState) AddInterruptedIterations(count uint64) uint64 {
	return atomic.AddUint64(es.interruptedIterationsCount, count)
}

// SetExecutionStatus changes the current execution status to the supplied value
// and returns the current value.
func (es *ExecutionState) SetExecutionStatus(newStatus ExecutionStatus) (oldStatus ExecutionStatus) {
	return ExecutionStatus(atomic.SwapUint32(es.executionStatus, uint32(newStatus)))
}

// GetCurrentExecutionStatus returns the current execution status. Don't use
// this for synchronization unless you've made the k6 behavior somewhat
// predictable with options like --paused or --linger.
func (es *ExecutionState) GetCurrentExecutionStatus() ExecutionStatus {
	return ExecutionStatus(atomic.LoadUint32(es.executionStatus))
}

// MarkStarted saves the current timestamp as the test start time.
//
// CAUTION: Calling MarkStarted() a second time for the same execution state will
// result in a panic!
func (es *ExecutionState) MarkStarted() {
	if !atomic.CompareAndSwapInt64(es.startTime, 0, time.Now().UnixNano()) {
		panic("the execution scheduler was started a second time")
	}
	es.SetExecutionStatus(ExecutionStatusStarted)
}

// MarkEnded saves the current timestamp as the test end time.
//
// CAUTION: Calling MarkEnded() a second time for the same execution state will
// result in a panic!
func (es *ExecutionState) MarkEnded() {
	if !atomic.CompareAndSwapInt64(es.endTime, 0, time.Now().UnixNano()) {
		panic("the execution scheduler was stopped a second time")
	}
	es.SetExecutionStatus(ExecutionStatusEnded)
}

// HasStarted returns true if the test has actually started executing.
// It will return false while a test is in the init phase, or if it has
// been initially paused. But if will return true if a test is paused
// midway through its execution (see above for details regarding the
// feasibility of that pausing for normal executors).
func (es *ExecutionState) HasStarted() bool {
	return atomic.LoadInt64(es.startTime) != 0
}

// HasEnded returns true if the test has finished executing. It will return
// false until MarkEnded() is called.
func (es *ExecutionState) HasEnded() bool {
	return atomic.LoadInt64(es.endTime) != 0
}

// IsPaused quickly returns whether the test is currently paused, by reading
// the atomic currentPauseTime timestamp
func (es *ExecutionState) IsPaused() bool {
	return atomic.LoadInt64(es.currentPauseTime) != 0
}

// GetCurrentTestRunDuration returns the duration for which the test has already
// ran. If the test hasn't started yet, that's 0. If it has started, but has
// been paused midway through, it will return the time up until the pause time.
// And if it's currently running, it will return the time since the start time.
//
// IMPORTANT: for UI/information purposes only, don't use for synchronization.
func (es *ExecutionState) GetCurrentTestRunDuration() time.Duration {
	startTime := atomic.LoadInt64(es.startTime)
	if startTime == 0 {
		// The test hasn't started yet
		return 0
	}

	es.pauseStateLock.RLock()
	endTime := atomic.LoadInt64(es.endTime)
	pausedDuration := es.totalPausedDuration
	es.pauseStateLock.RUnlock()

	if endTime == 0 {
		pauseTime := atomic.LoadInt64(es.currentPauseTime)
		if pauseTime != 0 {
			endTime = pauseTime
		} else {
			// The test isn't paused or finished, use the current time instead
			endTime = time.Now().UnixNano()
		}
	}

	return time.Duration(endTime-startTime) - pausedDuration
}

// Pause pauses the current execution. It acquires the lock, writes
// the current timestamp in currentPauseTime, and makes a new
// channel for resumeNotify.
// Pause can return an error if the test was already paused.
func (es *ExecutionState) Pause() error {
	es.pauseStateLock.Lock()
	defer es.pauseStateLock.Unlock()

	if !atomic.CompareAndSwapInt64(es.currentPauseTime, 0, time.Now().UnixNano()) {
		return errors.New("test execution was already paused")
	}
	es.resumeNotify = make(chan struct{})
	return nil
}

// Resume unpauses the test execution. Unless the test wasn't
// yet started, it calculates the duration between now and
// the old currentPauseTime and adds it to
// Resume will emit an error if the test wasn't paused.
func (es *ExecutionState) Resume() error {
	es.pauseStateLock.Lock()
	defer es.pauseStateLock.Unlock()

	currentPausedTime := atomic.SwapInt64(es.currentPauseTime, 0)
	if currentPausedTime == 0 {
		return errors.New("test execution wasn't paused")
	}

	// Check that it's not the pause before execution actually starts
	if atomic.LoadInt64(es.startTime) != 0 {
		es.totalPausedDuration += time.Duration(time.Now().UnixNano() - currentPausedTime)
	}

	close(es.resumeNotify)

	return nil
}

// ResumeNotify returns a channel which will be closed (i.e. could
// be read from) as soon as the test execution is resumed.
//
// Since tests would likely be paused only rarely, unless you
// directly need to be notified via a channel that the test
// isn't paused or that it has resumed, it's probably a good
// idea to first use the IsPaused() method, since it will be much
// faster.
//
// And, since tests won't be paused most of the time, it's
// probably better to check for that like this:
//   if executionState.IsPaused() {
//       <-executionState.ResumeNotify()
//   }
func (es *ExecutionState) ResumeNotify() <-chan struct{} {
	es.pauseStateLock.RLock()
	defer es.pauseStateLock.RUnlock()
	return es.resumeNotify
}

// GetPlannedVU tries to get a pre-initialized VU from the buffer channel. This
// shouldn't fail and should generally be an instantaneous action, but if it
// doesn't happen for MaxTimeToWaitForPlannedVU (for example, because the system
// is overloaded), a warning will be printed. If we reach that timeout more than
// MaxRetriesGetPlannedVU number of times, this function will return an error,
// since we either have a bug with some executor, or the machine is very, very
// overloaded.
//
// If modifyActiveVUCount is true, the method would also increment the counter
// for active VUs. In most cases, that's the desired behavior, but some
// executors might have to retrieve their reserved VUs without using them
// immediately - for example, the externally-controlled executor when the
// configured maxVUs number is greater than the configured starting VUs.
func (es *ExecutionState) GetPlannedVU(logger *logrus.Entry, modifyActiveVUCount bool) (InitializedVU, error) {
	for i := 1; i <= MaxRetriesGetPlannedVU; i++ {
		select {
		case vu := <-es.vus:
			if modifyActiveVUCount {
				es.ModCurrentlyActiveVUsCount(+1)
			}
			// TODO: set environment and exec
			return vu, nil
		case <-time.After(MaxTimeToWaitForPlannedVU):
			logger.Warnf("Could not get a VU from the buffer for %s", time.Duration(i)*MaxTimeToWaitForPlannedVU)
		}
	}
	return nil, fmt.Errorf(
		"could not get a VU from the buffer in %s",
		MaxRetriesGetPlannedVU*MaxTimeToWaitForPlannedVU,
	)
}

// SetInitVUFunc is called by the execution scheduler's init function, and it's
// used for setting the "constructor" function used for the initializing
// unplanned VUs.
//
// TODO: figure out a better dependency injection method?
func (es *ExecutionState) SetInitVUFunc(initVUFunc InitVUFunc) {
	es.initVUFunc = initVUFunc
}

// GetUnplannedVU checks if any unplanned VUs remain to be initialized, and if
// they do, it initializes one and returns it. If all unplanned VUs have already
// been initialized, it returns one from the global vus buffer, but doesn't
// automatically increment the active VUs counter in either case.
//
// IMPORTANT: GetUnplannedVU() doesn't do any checking if the requesting
// executor is actually allowed to have the VU at this particular time.
// Executors are trusted to correctly declare their needs (via their
// GetExecutionRequirements() methods) and then to never ask for more VUs than
// they have specified in those requirements.
func (es *ExecutionState) GetUnplannedVU(ctx context.Context, logger *logrus.Entry) (InitializedVU, error) {
	remVUs := atomic.AddInt64(es.uninitializedUnplannedVUs, -1)
	if remVUs < 0 {
		logger.Debug("Reusing a previously initialized unplanned VU")
		atomic.AddInt64(es.uninitializedUnplannedVUs, 1)
		return es.GetPlannedVU(logger, false)
	}

	logger.Debug("Initializing an unplanned VU, this may affect test results")
	return es.InitializeNewVU(ctx, logger)
}

// InitializeNewVU creates and returns a brand new VU, updating the relevant
// tracking counters.
func (es *ExecutionState) InitializeNewVU(ctx context.Context, logger *logrus.Entry) (InitializedVU, error) {
	if es.initVUFunc == nil {
		return nil, fmt.Errorf("initVUFunc wasn't set in the execution state")
	}
	newVU, err := es.initVUFunc(ctx, logger)
	if err != nil {
		return nil, err
	}
	es.ModInitializedVUsCount(+1)
	return newVU, err
}

// AddInitializedVU is a helper function that adds VUs into the buffer and
// increases the initialized VUs counter.
func (es *ExecutionState) AddInitializedVU(vu InitializedVU) {
	es.vus <- vu
	es.ModInitializedVUsCount(+1)
}

// ReturnVU is a helper function that puts VUs back into the buffer and
// decreases the active VUs counter.
func (es *ExecutionState) ReturnVU(vu InitializedVU, wasActive bool) {
	es.vus <- vu
	if wasActive {
		es.ModCurrentlyActiveVUsCount(-1)
	}
}
