package master

import (
	"errors"
	"net/http"

	"github.com/ss497254/gloski/internal/actions"
	"github.com/ss497254/gloski/internal/httpapi"
	"github.com/ss497254/gloski/internal/logger"
)

// actionLimit is the number of recent changes shown on a device page.
const actionLimit = 20

type undoOperation int

const (
	undoStart undoOperation = iota
	undoStop
)

// undoOperationFor maps a recorded change to the operation that restores its
// previous wanted state. Keep this in one place so the direction cannot be
// reversed by accident.
func undoOperationFor(kind string) (undoOperation, bool) {
	switch kind {
	case actions.KindAgentStop, actions.KindAgentRestart:
		return undoStart, true
	case actions.KindAgentStart:
		return undoStop, true
	default:
		return 0, false
	}
}

func (d *deployHandlers) handleListDeviceActions(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	if _, err := d.fleetSvc.Get(deviceID); err != nil {
		httpapi.NotFound(w, "device not found")
		return
	}
	list, err := d.actionStore.Recent(r.Context(), deviceID, actionLimit)
	if err != nil {
		httpapi.InternalError(w, "failed to load device actions", err.Error())
		return
	}
	httpapi.SuccessWithMessage(w, map[string]interface{}{"actions": list})
}

// handleUndoAction restores the wanted state recorded in a device action. Only
// changes whose previous state is known can be undone, and only once.
func (d *deployHandlers) handleUndoAction(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	device, err := d.fleetSvc.Get(deviceID)
	if err != nil {
		httpapi.NotFound(w, "device not found")
		return
	}
	action, err := d.actionStore.Get(r.Context(), r.PathValue("actionId"))
	if err != nil {
		if errors.Is(err, actions.ErrActionNotFound) {
			httpapi.NotFound(w, "action not found")
			return
		}
		httpapi.InternalError(w, "failed to load action", err.Error())
		return
	}
	if action.DeviceID != deviceID {
		httpapi.NotFound(w, "action not found")
		return
	}
	if !action.Undoable || action.UndoneAt != nil || action.Result != actions.ResultOK {
		httpapi.BadRequest(w, "this change cannot be undone")
		return
	}

	op, ok := undoOperationFor(action.Kind)
	if !ok {
		httpapi.BadRequest(w, "this kind of change cannot be undone")
		return
	}
	claimed, claimErr := d.actionStore.ClaimUndo(r.Context(), action.ID)
	if claimErr != nil {
		httpapi.InternalError(w, "failed to claim the undo", claimErr.Error())
		return
	}
	if !claimed {
		httpapi.BadRequest(w, "this change has already been undone")
		return
	}
	var undoErr error
	if op == undoStart {
		_, undoErr = d.installDevice(r.Context(), device, false)
	} else {
		_, undoErr = d.stopDevice(r.Context(), device)
	}
	if undoErr != nil {
		// Release the claim so the change can be retried, and record the
		// failed attempt just like a failed start or stop.
		if releaseErr := d.actionStore.ReleaseUndo(r.Context(), action.ID); releaseErr != nil {
			logger.Warn("Failed to release undo claim on %s: %v", action.ID, releaseErr)
		}
		d.recordAction(deviceID, actions.Action{
			Kind: actions.KindUndo, Target: action.Target, From: action.To, To: action.From,
		}, undoErr)
		httpapi.InternalError(w, "failed to undo the change", undoErr.Error())
		return
	}
	d.recordAction(deviceID, actions.Action{
		Kind: actions.KindUndo, Target: action.Target, From: action.To, To: action.From,
	}, nil)
	httpapi.SuccessWithMessage(w, map[string]interface{}{"undone": true})
}
