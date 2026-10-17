package pipeline

import (
	"errors"
	"io"
	"log/slog"
	"slices"
	"testing"

	"codeberg.org/hedge10/staticform/mailer"
)

func sendOK(mailer.Message) error { return nil }

// call is one hook invocation.
type call struct {
	hook  string
	event Event
	err   error
}

type recorder struct{ calls []call }

func (r *recorder) hook(name string) Hook {
	return func(event Event, _ Submission, err error) {
		r.calls = append(r.calls, call{hook: name, event: event, err: err})
	}
}

// run runs background work immediately, so the order of events is fixed.
func run(fn func()) { fn() }

func newTestPipeline(send func(mailer.Message) error, run func(func())) *Pipeline {
	return New(send, slog.New(slog.NewTextHandler(io.Discard, nil)), run)
}

func TestDeliverEventOrder(t *testing.T) {
	sendErr := errors.New("smtp unavailable")
	tests := []struct {
		name    string
		sendErr error
		want    []call
	}{
		{name: "mail sent", want: []call{
			{"a", Before, nil}, {"a", Success, nil}, {"a", After, nil},
		}},
		{name: "mail failed", sendErr: sendErr, want: []call{
			{"a", Before, nil}, {"a", Error, sendErr}, {"a", After, sendErr},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := &recorder{}
			p := newTestPipeline(func(mailer.Message) error { return tt.sendErr }, run)
			p.On(rec.hook("a"), Events()...)

			p.Deliver(Submission{})

			if !slices.Equal(rec.calls, tt.want) {
				t.Errorf("calls %v, want %v", rec.calls, tt.want)
			}
		})
	}
}

func TestDeliverSendsMessage(t *testing.T) {
	var sent []mailer.Message
	p := newTestPipeline(func(msg mailer.Message) error {
		sent = append(sent, msg)
		return nil
	}, run)

	p.Deliver(Submission{Message: mailer.Message{To: []string{"owner@example.com"}, Template: "new_mail.tmpl"}})

	if len(sent) != 1 || sent[0].Template != "new_mail.tmpl" {
		t.Errorf("sent %v, want one message with template new_mail.tmpl", sent)
	}
}

func TestDeliverSendsInBackground(t *testing.T) {
	sends := 0
	rec := &recorder{}
	var background func()
	p := newTestPipeline(func(mailer.Message) error {
		sends++
		return nil
	}, func(fn func()) { background = fn })
	p.On(rec.hook("a"), Events()...)

	p.Deliver(Submission{})

	// Only Before runs before Deliver returns.
	if want := []call{{"a", Before, nil}}; !slices.Equal(rec.calls, want) {
		t.Fatalf("calls %v, want %v", rec.calls, want)
	}
	if sends != 0 {
		t.Fatalf("sent before background work ran")
	}

	background()

	if sends != 1 || len(rec.calls) != 3 {
		t.Errorf("after background: sends %d, calls %v", sends, rec.calls)
	}
}

func TestOnOnlyRequestedEvents(t *testing.T) {
	rec := &recorder{}
	p := newTestPipeline(sendOK, run)
	p.On(rec.hook("a"), Success)

	p.Deliver(Submission{})

	if want := []call{{"a", Success, nil}}; !slices.Equal(rec.calls, want) {
		t.Errorf("calls %v, want %v", rec.calls, want)
	}
}

func TestHookReceivesSubmission(t *testing.T) {
	var got Submission
	p := newTestPipeline(sendOK, run)
	p.On(func(_ Event, s Submission, _ error) { got = s }, After)

	p.Deliver(Submission{ID: "sub_123", Fields: map[string]string{"name": "Jane"}})

	if got.ID != "sub_123" || got.Fields.(map[string]string)["name"] != "Jane" {
		t.Errorf("hook got %+v", got)
	}
}

func TestWithDoesNotChangeParent(t *testing.T) {
	rec := &recorder{}
	p := newTestPipeline(sendOK, run)
	p.On(rec.hook("global"), Before)
	fp := p.With()
	fp.On(rec.hook("form"), Before)

	p.Deliver(Submission{})
	if want := []call{{"global", Before, nil}}; !slices.Equal(rec.calls, want) {
		t.Errorf("parent calls %v, want %v", rec.calls, want)
	}

	rec.calls = nil
	fp.Deliver(Submission{})
	if len(rec.calls) != 2 {
		t.Errorf("copy calls %v, want global and form", rec.calls)
	}
}

func TestPanickingHookDoesNotStopOthers(t *testing.T) {
	sends := 0
	rec := &recorder{}
	p := newTestPipeline(func(mailer.Message) error {
		sends++
		return nil
	}, run)
	p.On(func(Event, Submission, error) { panic("boom") }, Before)
	p.On(rec.hook("b"), Events()...)

	p.Deliver(Submission{})

	if sends != 1 || len(rec.calls) != 3 {
		t.Errorf("sends %d, calls %v; want 1 send and 3 calls", sends, rec.calls)
	}
}
