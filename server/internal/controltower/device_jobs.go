package controltower

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ss497254/gloski/internal/httpapi"
	"github.com/ss497254/gloski/internal/jobs"
)

type runJobRequest struct {
	Command string `json:"command"`
	CWD     string `json:"cwd"`
}

func (d *deviceHandlers) handleListJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := d.agentService.ListJobs(r.PathValue("id"))
	if err != nil {
		writeAgentError(w, err)
		return
	}
	httpapi.Success(w, map[string]interface{}{"jobs": jobs})
}

func (d *deviceHandlers) handleRunJob(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeJSON[runJobRequest](w, r, 1<<20)
	if !ok {
		return
	}
	if strings.TrimSpace(request.Command) == "" {
		httpapi.BadRequest(w, "command is required")
		return
	}
	job, err := d.agentService.RunJob(r.PathValue("id"), request.Command, request.CWD)
	if err != nil {
		writeAgentError(w, err)
		return
	}
	httpapi.Success(w, job)
}

// deviceJob loads the job named in the URL and checks that it belongs to the
// device in the same URL, so one device's path cannot reach another's jobs.
func (d *deviceHandlers) deviceJob(w http.ResponseWriter, r *http.Request) (*jobs.Job, bool) {
	job, err := d.agentService.GetJob(r.PathValue("jobId"))
	if err != nil {
		if errors.Is(err, jobs.ErrJobNotFound) {
			httpapi.NotFound(w, "job not found")
			return nil, false
		}
		writeAgentError(w, err)
		return nil, false
	}
	if job.DeviceID != r.PathValue("id") {
		httpapi.NotFound(w, "job not found")
		return nil, false
	}
	return job, true
}

func (d *deviceHandlers) handleGetJob(w http.ResponseWriter, r *http.Request) {
	if job, ok := d.deviceJob(w, r); ok {
		httpapi.Success(w, job)
	}
}

func (d *deviceHandlers) handleJobLogs(w http.ResponseWriter, r *http.Request) {
	job, ok := d.deviceJob(w, r)
	if !ok {
		return
	}
	logs, err := d.agentService.JobLogs(job.ID)
	if err != nil {
		writeAgentError(w, err)
		return
	}
	httpapi.Success(w, map[string]interface{}{"logs": logs})
}

func (d *deviceHandlers) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	job, ok := d.deviceJob(w, r)
	if !ok {
		return
	}
	if err := d.agentService.CancelJob(job.DeviceID, job.ID, "cancelled by user"); err != nil {
		writeAgentError(w, err)
		return
	}
	httpapi.SuccessWithMessage(w, nil)
}

func (d *deviceHandlers) handleDeleteJob(w http.ResponseWriter, r *http.Request) {
	job, ok := d.deviceJob(w, r)
	if !ok {
		return
	}
	if err := d.agentService.DeleteJob(job.ID); err != nil {
		writeAgentError(w, err)
		return
	}
	httpapi.SuccessWithMessage(w, nil)
}
