package execution

import (
	copilot "github.com/github/copilot-sdk/go"
	"github.com/spboyer/waza/internal/models"
)

const sessionFailedUnknown = "session failed with unknown error"

// NewSessionEventsCollector creates a new SessionEvents.
func NewSessionEventsCollector() *SessionEventsCollector {
	return &SessionEventsCollector{
		done:          make(chan struct{}),
		intentToolIDs: map[string]bool{},
	}
}

type SessionEventsCollector struct {
	sessionEvents []copilot.SessionEvent
	outputParts   []string
	errorMsg      string
	done          chan struct{}
	intentToolIDs map[string]bool
}

// SessionEvents returns the collected session events.
func (coll *SessionEventsCollector) SessionEvents() []copilot.SessionEvent {
	return coll.sessionEvents
}

// OutputParts returns the collected output text parts.
func (coll *SessionEventsCollector) OutputParts() []string {
	return coll.outputParts
}

// ErrorMessage returns the error message, if any.
func (coll *SessionEventsCollector) ErrorMessage() string {
	return coll.errorMsg
}

// Done returns the channel that is closed when the session completes.
func (coll *SessionEventsCollector) Done() <-chan struct{} {
	return coll.done
}

// On is a callback, intended to be passed to [copilot.Session.On] to receive
// events in real-time.
func (coll *SessionEventsCollector) On(event copilot.SessionEvent) {
	switch event.Type {
	case copilot.AssistantMessage, copilot.AssistantMessageDelta:
		if event.Data.Content != nil {
			coll.outputParts = append(coll.outputParts, *event.Data.Content)
		}

	case copilot.ToolExecutionStart:
		if event.Data.ToolName != nil && *event.Data.ToolName == "report_intent" {
			// report_intent always seems to be followed by the actual tool invocation,
			// so I'm just going to skip these to save a little space.
			if event.Data.ToolCallID != nil {
				coll.intentToolIDs[*event.Data.ToolCallID] = true
			}
			return
		}
	case copilot.ToolExecutionProgress,
		copilot.ToolUserRequested:
		if event.Data.ToolCallID != nil && coll.intentToolIDs[*event.Data.ToolCallID] {
			return
		}

	case copilot.ToolExecutionComplete, copilot.ToolExecutionPartialResult:
		if event.Data.ToolCallID != nil && coll.intentToolIDs[*event.Data.ToolCallID] {
			delete(coll.intentToolIDs, *event.Data.ToolCallID)
			return
		}
	// these are both termination events
	case copilot.SessionIdle, copilot.SessionError:
		if event.Type == copilot.SessionError {
			if event.Data.Message == nil || *event.Data.Message == "" {
				coll.errorMsg = sessionFailedUnknown
			} else {
				coll.errorMsg = *event.Data.Message
			}
		}

		select {
		case <-coll.done:
		default:
			close(coll.done)
		}
	}

	coll.sessionEvents = append(coll.sessionEvents, event)
}

// ToolCalls goes through the list of session events and correlates tool starts
// with Success. The resulting tool calls are not cached - if you're going to use
// it repeatedly you should store it locally.
func (coll *SessionEventsCollector) ToolCalls() []models.ToolCall {
	return models.FilterToolCalls(coll.sessionEvents)
}
