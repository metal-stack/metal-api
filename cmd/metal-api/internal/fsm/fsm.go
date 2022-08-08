package fsm

import (
	"errors"
	"fmt"

	"github.com/looplab/fsm"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/fsm/states"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"go.uber.org/zap"
)

type provisioningFSM struct {
	fsm       *fsm.FSM
	container *metal.ProvisioningEventContainer
	event     *metal.ProvisioningEvent
	log       *zap.SugaredLogger
}

// HandleProvisioningEvent can be called to determine whether the given incoming event follows an expected lifecycle of a machine considering the event history of the given provisioning event container.
//
// The function returns a new provisioning event container that can then be persisted in the database. If an error is returned, the incoming event is not supposed to be persisted in the database.
//
// Among other things, this function can detect crash loops or other irregularities within a machine lifecycle and enriches the returned provisioning event container with this information.
func HandleProvisioningEvent(log *zap.SugaredLogger, ec *metal.ProvisioningEventContainer, event *metal.ProvisioningEvent) (*metal.ProvisioningEventContainer, error) {
	if ec == nil {
		return nil, fmt.Errorf("provisioning event container must not be nil")
	}

	if event == nil {
		return nil, fmt.Errorf("provisioning event must not be nil")
	}

	clone := *ec
	container := &clone

	initial := states.Initial.String()
	if len(container.Events) != 0 {
		initial = container.Events[0].Event.String()
	}

	provisioningFSM := newProvisioningFSM(initial, container, event, log)

	err := provisioningFSM.fsm.Event(event.Event.String(), provisioningFSM, event)
	if err == nil {
		return container, nil
	}

	if errors.As(err, &fsm.InvalidEventError{}) {
		container.Events = append([]metal.ProvisioningEvent{*event}, container.Events...)
		container.LastEventTime = &event.Time
		container.CrashLoop = true

		return container, nil
	}

	return nil, fmt.Errorf("internal error while calculating provisioning event container for machine %s", container.ID)
}

func newProvisioningFSM(lastEvent string, container *metal.ProvisioningEventContainer, event *metal.ProvisioningEvent, log *zap.SugaredLogger) *provisioningFSM {
	p := provisioningFSM{
		container: container,
		event:     event,
		log:       log,
	}

	callbacks := fsm.Callbacks{
		"before_" + metal.ProvisioningEventAlive.String(): p.handleAliveEvent,
	}

	for _, s := range states.AllStates(&states.StateConfig{Log: log, Event: event, Container: container}) {
		callbacks["before_"+s.Name()] = s.Handle
	}

	p.fsm = fsm.NewFSM(
		getEventDestination(lastEvent),
		Events(),
		callbacks,
	)

	return &p
}

func getEventDestination(event string) string {
	for _, e := range Events() {
		if e.Name == event && e.Dst != "" {
			return e.Dst
		}
	}

	return states.Initial.String()
}

func (f *provisioningFSM) handleAliveEvent(e *fsm.Event) {
	states.UpdateTimeAndLiveliness(f.event, f.container)
	f.log.Debugw("received provisioning alive event", "id", f.container.ID)
}
