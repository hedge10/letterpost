// Package pipeline runs the steps of delivering a submission: it sends the
// mail in the background and calls the registered hooks before and after.
package pipeline

import (
	"log/slog"
	"slices"

	"codeberg.org/hedge10/staticform/mailer"
)

// Hook reacts to one step of a delivery.
type Hook func(event Event, s Submission, err error)

// Event is a step of a delivery.
type Event string

const (
	Before  Event = "before"
	Success Event = "success"
	Error   Event = "error"
	After   Event = "after"
)

// Events returns all events in the order they happen.
func Events() []Event {
	return []Event{Before, Success, Error, After}
}

// Submission is what gets delivered.
type Submission struct {
	ID      string
	Message mailer.Message
	// Fields are handed to hooks, e.g. as the webhook payload.
	Fields any
}

type Pipeline struct {
	send   func(mailer.Message) error
	logger *slog.Logger
	run    func(func())
	hooks  map[Event][]Hook
}

// New returns a pipeline that sends with send and runs background work with run.
func New(send func(mailer.Message) error, logger *slog.Logger, run func(func())) *Pipeline {
	return &Pipeline{send: send, logger: logger, run: run, hooks: make(map[Event][]Hook)}
}

// On registers h for the given events. Hooks are registered before the first
// Deliver; On is not safe to call while deliveries run.
func (p *Pipeline) On(h Hook, events ...Event) {
	for _, e := range events {
		p.hooks[e] = append(p.hooks[e], h)
	}
}

// With returns a copy of p. Hooks registered on the copy do not reach p.
func (p *Pipeline) With() *Pipeline {
	c := *p
	c.hooks = make(map[Event][]Hook, len(p.hooks))
	for e, hs := range p.hooks {
		c.hooks[e] = slices.Clone(hs)
	}
	return &c
}

// Deliver fires Before, then sends the message in the background and
// fires Success or Error, and finally After.
func (p *Pipeline) Deliver(s Submission) {
	p.fire(Before, s, nil)

	p.run(func() {
		err := p.send(s.Message)
		if err != nil {
			p.fire(Error, s, err)
		} else {
			p.fire(Success, s, nil)
		}
		p.fire(After, s, err)
	})
}

// fire calls the hooks registered for event. A panicking hook is recovered
// and logged, so it cannot stop the others.
func (p *Pipeline) fire(event Event, s Submission, err error) {
	for _, h := range p.hooks[event] {
		func() {
			defer func() {
				if r := recover(); r != nil {
					p.logger.Error("hook panicked", "event", event, "panic", r)
				}
			}()
			h(event, s, err)
		}()
	}
}
