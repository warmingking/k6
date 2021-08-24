/*
 *
 * k6 - a next-generation load testing tool
 * Copyright (C) 2016 Load Impact
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

// Package k6 implements the module imported as 'k6' from inside k6.
package k6

import (
	"errors"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/dop251/goja"

	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/modules"
	"go.k6.io/k6/lib/metrics"
	"go.k6.io/k6/stats"
)

// K6 is just the module struct.
type K6 struct {
	modules.InstanceCore
}

// ErrGroupInInitContext is returned when group() are using in the init context.
var ErrGroupInInitContext = common.NewInitContextError("Using group() in the init context is not supported")

// ErrCheckInInitContext is returned when check() are using in the init context.
var ErrCheckInInitContext = common.NewInitContextError("Using check() in the init context is not supported")

// New returns a new module Struct.
func New() *K6Root {
	return &K6Root{}
}

type K6Root struct{}

var _ modules.IsModuleV2 = &K6Root{}

func (*K6Root) NewModuleInstance(core modules.InstanceCore) modules.Instance {
	return &K6{InstanceCore: core}
}

func (k *K6) GetExports() modules.Exports {
	return modules.Exports{
		Named: map[string]interface{}{
			"fail":       k.Fail,
			"sleep":      k.Sleep,
			"group":      k.Group,
			"randomSeed": k.RandomSeed,
			"check":      k.Check,
		},
	}
}

// Fail is a fancy way of saying `throw "something"`.
func (*K6) Fail(msg string) (goja.Value, error) {
	return goja.Undefined(), errors.New(msg)
}

// Sleep waits the provided seconds before continuing the execution.
func (k *K6) Sleep(secs float64) {
	// let other things run
	k.YieldRuntime()
	defer k.GetRuntime()
	ctx := k.GetContext()
	timer := time.NewTimer(time.Duration(secs * float64(time.Second)))
	select {
	case <-timer.C:
	case <-ctx.Done():
		timer.Stop()
	}
}

// RandomSeed sets the seed to the random generator used for this VU.
func (k *K6) RandomSeed(seed int64) {
	randSource := rand.New(rand.NewSource(seed)).Float64 //nolint:gosec

	k.GetRuntime().SetRandSource(randSource)
}

// Group wraps a function call and executes it within the provided group name.
func (k *K6) Group(name string, fn goja.Callable) (goja.Value, error) {
	state := k.GetState()
	if state == nil {
		return nil, ErrGroupInInitContext
	}

	if fn == nil {
		return nil, errors.New("group() requires a callback as a second argument")
	}

	g, err := state.Group.Group(name)
	if err != nil {
		return goja.Undefined(), err
	}

	old := state.Group
	state.Group = g

	shouldUpdateTag := state.Options.SystemTags.Has(stats.TagGroup)
	if shouldUpdateTag {
		state.Tags["group"] = g.Path
	}
	defer func() {
		state.Group = old
		if shouldUpdateTag {
			state.Tags["group"] = old.Path
		}
	}()

	startTime := time.Now()
	ret, err := fn(goja.Undefined())
	t := time.Now()

	tags := state.CloneTags()
	ctx := k.GetContext()
	stats.PushIfNotDone(ctx, state.Samples, stats.Sample{
		Time:   t,
		Metric: metrics.GroupDuration,
		Tags:   stats.IntoSampleTags(&tags),
		Value:  stats.D(t.Sub(startTime)),
	})

	return ret, err
}

// Check will emit check metrics for the provided checks.
//nolint:cyclop
func (k *K6) Check(arg0, checks goja.Value, extras ...goja.Value) (bool, error) {
	state := k.GetState()
	if state == nil {
		return false, ErrCheckInInitContext
	}
	rt := k.GetRuntime()
	t := time.Now()

	// Prepare the metric tags
	commonTags := state.CloneTags()
	if len(extras) > 0 {
		obj := extras[0].ToObject(rt)
		for _, k := range obj.Keys() {
			commonTags[k] = obj.Get(k).String()
		}
	}

	succ := true
	var exc error
	obj := checks.ToObject(rt)
	for _, name := range obj.Keys() {
		val := obj.Get(name)

		tags := make(map[string]string, len(commonTags))
		for k, v := range commonTags {
			tags[k] = v
		}

		// Resolve the check record.
		check, err := state.Group.Check(name)
		if err != nil {
			return false, err
		}
		if state.Options.SystemTags.Has(stats.TagCheck) {
			tags["check"] = check.Name
		}

		// Resolve callables into values.
		fn, ok := goja.AssertFunction(val)
		if ok {
			tmpVal, err := fn(goja.Undefined(), arg0)
			val = tmpVal
			if err != nil {
				val = rt.ToValue(false)
				exc = err
			}
		}

		sampleTags := stats.IntoSampleTags(&tags)

		// Emit! (But only if we have a valid context.)
		ctx := k.GetContext()
		select {
		case <-ctx.Done():
		default:
			if val.ToBoolean() {
				atomic.AddInt64(&check.Passes, 1)
				stats.PushIfNotDone(ctx, state.Samples, stats.Sample{Time: t, Metric: metrics.Checks, Tags: sampleTags, Value: 1})
			} else {
				atomic.AddInt64(&check.Fails, 1)
				stats.PushIfNotDone(ctx, state.Samples, stats.Sample{Time: t, Metric: metrics.Checks, Tags: sampleTags, Value: 0})
				// A single failure makes the return value false.
				succ = false
			}
		}

		if exc != nil {
			return succ, exc
		}
	}

	return succ, nil
}
