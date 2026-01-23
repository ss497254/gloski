// Package docker provides Docker container management functionality.
package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

var (
	ErrDockerNotFound    = errors.New("docker not found")
	ErrDockerNotRunning  = errors.New("docker daemon not running")
	ErrContainerNotFound = errors.New("container not found")
)

// Service provides Docker management operations.
type Service struct {
	available bool
}

// NewService creates a new Docker service.
func NewService() (*Service, error) {
	s := &Service{}

	// Check if docker is available
	if err := s.checkDocker(); err != nil {
		return nil, err
	}

	s.available = true
	return s, nil
}

func (s *Service) checkDocker() error {
	// Check if docker command exists
	_, err := exec.LookPath("docker")
	if err != nil {
		return ErrDockerNotFound
	}

	// Check if docker daemon is running
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "info")
	if err := cmd.Run(); err != nil {
		return ErrDockerNotRunning
	}

	return nil
}

// Container represents a Docker container.
type Container struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	Status  string            `json:"status"`
	State   string            `json:"state"`
	Created time.Time         `json:"created"`
	Ports   []Port            `json:"ports"`
	Labels  map[string]string `json:"labels,omitempty"`
}

// Port represents a container port mapping.
type Port struct {
	Private  int    `json:"private"`
	Public   int    `json:"public,omitempty"`
	Protocol string `json:"protocol"`
}

// Image represents a Docker image.
type Image struct {
	ID         string    `json:"id"`
	Repository string    `json:"repository"`
	Tag        string    `json:"tag"`
	Size       int64     `json:"size"`
	Created    time.Time `json:"created"`
}

// ListContainers returns all containers (running and stopped).
func (s *Service) ListContainers(all bool) ([]Container, error) {
	args := []string{"ps", "--format", "{{json .}}"}
	if all {
		args = append(args, "-a")
	}

	cmd := exec.Command("docker", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker ps failed: %w", err)
	}

	var containers []Container
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		var raw struct {
			ID      string `json:"ID"`
			Names   string `json:"Names"`
			Image   string `json:"Image"`
			Status  string `json:"Status"`
			State   string `json:"State"`
			Created string `json:"CreatedAt"`
			Ports   string `json:"Ports"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &raw); err != nil {
			continue
		}

		container := Container{
			ID:     raw.ID,
			Name:   strings.TrimPrefix(raw.Names, "/"),
			Image:  raw.Image,
			Status: raw.Status,
			State:  raw.State,
			Ports:  parsePorts(raw.Ports),
		}

		// Parse created time
		if t, err := time.Parse("2006-01-02 15:04:05 -0700 MST", raw.Created); err == nil {
			container.Created = t
		}

		containers = append(containers, container)
	}

	return containers, nil
}

// ListImages returns all Docker images.
func (s *Service) ListImages() ([]Image, error) {
	cmd := exec.Command("docker", "images", "--format", "{{json .}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker images failed: %w", err)
	}

	var images []Image
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		var raw struct {
			ID         string `json:"ID"`
			Repository string `json:"Repository"`
			Tag        string `json:"Tag"`
			Size       string `json:"Size"`
			CreatedAt  string `json:"CreatedAt"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &raw); err != nil {
			continue
		}

		image := Image{
			ID:         raw.ID,
			Repository: raw.Repository,
			Tag:        raw.Tag,
			Size:       parseSize(raw.Size),
		}

		if t, err := time.Parse("2006-01-02 15:04:05 -0700 MST", raw.CreatedAt); err == nil {
			image.Created = t
		}

		images = append(images, image)
	}

	return images, nil
}

// ContainerAction performs an action on a container (start, stop, restart, remove).
func (s *Service) ContainerAction(containerID, action string) error {
	var args []string
	switch action {
	case "start":
		args = []string{"start", containerID}
	case "stop":
		args = []string{"stop", containerID}
	case "restart":
		args = []string{"restart", containerID}
	case "remove":
		args = []string{"rm", "-f", containerID}
	case "pause":
		args = []string{"pause", containerID}
	case "unpause":
		args = []string{"unpause", containerID}
	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	cmd := exec.Command("docker", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s failed: %s", action, string(output))
	}

	return nil
}

// ContainerLogs returns logs from a container.
func (s *Service) ContainerLogs(containerID string, tail int, follow bool) (string, error) {
	args := []string{"logs", "--tail", fmt.Sprintf("%d", tail)}
	if !follow {
		args = append(args, containerID)
	}

	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("logs failed: %w", err)
	}

	return string(output), nil
}

// ContainerInspect returns detailed information about a container.
func (s *Service) ContainerInspect(containerID string) (map[string]interface{}, error) {
	cmd := exec.Command("docker", "inspect", containerID)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("inspect failed: %w", err)
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, ErrContainerNotFound
	}

	return result[0], nil
}

// ContainerStats returns resource usage statistics for a container.
func (s *Service) ContainerStats(containerID string) (map[string]interface{}, error) {
	cmd := exec.Command("docker", "stats", "--no-stream", "--format", "{{json .}}", containerID)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("stats failed: %w", err)
	}

	var stats map[string]interface{}
	if err := json.Unmarshal(output, &stats); err != nil {
		return nil, err
	}

	return stats, nil
}

// PullImage pulls a Docker image.
func (s *Service) PullImage(image string) error {
	cmd := exec.Command("docker", "pull", image)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pull failed: %s", string(output))
	}
	return nil
}

// RemoveImage removes a Docker image.
func (s *Service) RemoveImage(imageID string, force bool) error {
	args := []string{"rmi", imageID}
	if force {
		args = []string{"rmi", "-f", imageID}
	}

	cmd := exec.Command("docker", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("remove image failed: %s", string(output))
	}
	return nil
}

// Info returns Docker system information.
func (s *Service) Info() (map[string]interface{}, error) {
	cmd := exec.Command("docker", "info", "--format", "{{json .}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("info failed: %w", err)
	}

	var info map[string]interface{}
	if err := json.Unmarshal(output, &info); err != nil {
		return nil, err
	}

	return info, nil
}

// Shutdown gracefully shuts down the Docker service.
func (s *Service) Shutdown() error {
	// Nothing to clean up for Docker service
	return nil
}

// Helper functions

func parsePorts(portsStr string) []Port {
	var ports []Port
	if portsStr == "" {
		return ports
	}

	// Format: "0.0.0.0:8080->80/tcp, 443/tcp"
	parts := strings.Split(portsStr, ", ")
	for _, part := range parts {
		port := Port{Protocol: "tcp"}

		// Check for protocol
		if strings.HasSuffix(part, "/udp") {
			port.Protocol = "udp"
			part = strings.TrimSuffix(part, "/udp")
		} else {
			part = strings.TrimSuffix(part, "/tcp")
		}

		// Check for mapping
		if strings.Contains(part, "->") {
			mapping := strings.Split(part, "->")
			hostPart := mapping[0]
			containerPort := mapping[1]

			// Extract public port
			if strings.Contains(hostPart, ":") {
				hostParts := strings.Split(hostPart, ":")
				fmt.Sscanf(hostParts[len(hostParts)-1], "%d", &port.Public)
			}
			fmt.Sscanf(containerPort, "%d", &port.Private)
		} else {
			fmt.Sscanf(part, "%d", &port.Private)
		}

		if port.Private > 0 {
			ports = append(ports, port)
		}
	}

	return ports
}

func parseSize(sizeStr string) int64 {
	var size float64
	var unit string
	fmt.Sscanf(sizeStr, "%f%s", &size, &unit)

	unit = strings.ToUpper(strings.TrimSpace(unit))
	switch unit {
	case "KB":
		return int64(size * 1024)
	case "MB":
		return int64(size * 1024 * 1024)
	case "GB":
		return int64(size * 1024 * 1024 * 1024)
	default:
		return int64(size)
	}
}
