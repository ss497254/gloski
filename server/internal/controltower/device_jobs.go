package controltower

import (
	"net/http"
	"strings"

	"github.com/ss497254/gloski/internal/httpapi"
)

type runJobRequest struct {
	Command string `json:"command"`
	CWD     string `json:"cwd"`
}

func (t *Tower) handleListJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := t.agentService.ListJobs(r.PathValue("id"))
	if err != nil {
		writeAgentError(w, err)
		return
	}
	httpapi.Success(w, map[string]interface{}{"jobs": jobs})
}

func (t *Tower) handleRunJob(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeJSON[runJobRequest](w, r, 1<<20)
	if !ok {
		return
	}
	if strings.TrimSpace(request.Command) == "" {
		httpapi.BadRequest(w, "command is required")
		return
	}
	job, err := t.agentService.RunJob(r.PathValue("id"), request.Command, request.CWD)
	if err != nil {
		writeAgentError(w, err)
		return
	}
	httpapi.Success(w, job)
}

func (t *Tower) handleGetJob(w http.ResponseWriter, r *http.Request) {
	job, err := t.agentService.GetJob(r.PathValue("jobId"))
	if err != nil {
		httpapi.NotFound(w, "job not found")
		return
	}
	httpapi.Success(w, job)
}

func (t *Tower) handleJobLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := t.agentService.JobLogs(r.PathValue("jobId"))
	if err != nil {
		writeAgentError(w, err)
		return
	}
	httpapi.Success(w, map[string]interface{}{"logs": logs})
}

func (t *Tower) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	if err := t.agentService.CancelJob(r.PathValue("id"), r.PathValue("jobId"), "cancelled by user"); err != nil {
		writeAgentError(w, err)
		return
	}
	httpapi.SuccessWithMessage(w, nil)
}

func (t *Tower) handleDeleteJob(w http.ResponseWriter, r *http.Request) {
	if err := t.agentService.DeleteJob(r.PathValue("jobId")); err != nil {
		writeAgentError(w, err)
		return
	}
	httpapi.SuccessWithMessage(w, nil)
}
