package master

import (
	"net/http"
	"strings"

	"github.com/ss497254/gloski/internal/api/handlers"
)

type runJobRequest struct {
	Command string `json:"command"`
	CWD     string `json:"cwd"`
}

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.agents.ListJobs(r.PathValue("id"))
	if err != nil {
		writeAgentError(w, err)
		return
	}
	handlers.Success(w, map[string]interface{}{"jobs": jobs})
}

func (s *Server) handleRunJob(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeJSON[runJobRequest](w, r, 1<<20)
	if !ok {
		return
	}
	if strings.TrimSpace(request.Command) == "" {
		handlers.BadRequest(w, "command is required")
		return
	}
	job, err := s.agents.RunJob(r.PathValue("id"), request.Command, request.CWD)
	if err != nil {
		writeAgentError(w, err)
		return
	}
	handlers.Success(w, job)
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	job, err := s.agents.GetJob(r.PathValue("jobId"))
	if err != nil {
		handlers.NotFound(w, "job not found")
		return
	}
	handlers.Success(w, job)
}

func (s *Server) handleJobLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := s.agents.JobLogs(r.PathValue("jobId"))
	if err != nil {
		writeAgentError(w, err)
		return
	}
	handlers.Success(w, map[string]interface{}{"logs": logs})
}

func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	if err := s.agents.CancelJob(r.PathValue("id"), r.PathValue("jobId"), "cancelled by user"); err != nil {
		writeAgentError(w, err)
		return
	}
	handlers.SuccessWithMessage(w, nil)
}

func (s *Server) handleDeleteJob(w http.ResponseWriter, r *http.Request) {
	if err := s.agents.DeleteJob(r.PathValue("jobId")); err != nil {
		writeAgentError(w, err)
		return
	}
	handlers.SuccessWithMessage(w, nil)
}
