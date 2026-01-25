package handlers

import (
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Version is set at build time via ldflags
var Version = "dev"

// StartTime is set when the server starts
var StartTime = time.Now()

// DeployHandler handles deployment status requests
type DeployHandler struct{}

// NewDeployHandler creates a new deploy handler
func NewDeployHandler() *DeployHandler {
	return &DeployHandler{}
}

// DeployStatusResponse represents the deployment status
type DeployStatusResponse struct {
	Running       bool   `json:"running"`
	Version       string `json:"version"`
	UptimeSeconds int64  `json:"uptime_seconds"`
	LastDeploy    string `json:"last_deploy,omitempty"`
	ContainerID   string `json:"container_id,omitempty"`
	Hostname      string `json:"hostname,omitempty"`
}

// GetStatus returns the current deployment status
func (h *DeployHandler) GetStatus(c *gin.Context) {
	uptime := time.Since(StartTime).Seconds()

	hostname, _ := os.Hostname()

	// Try to get container ID from Docker
	containerID := getContainerID()

	// Try to get last deploy time from container start time
	lastDeploy := ""
	if containerID != "" {
		lastDeploy = getContainerStartTime(containerID)
	}

	response := DeployStatusResponse{
		Running:       true,
		Version:       Version,
		UptimeSeconds: int64(uptime),
		LastDeploy:    lastDeploy,
		ContainerID:   containerID,
		Hostname:      hostname,
	}

	c.JSON(http.StatusOK, response)
}

// getContainerID tries to detect if we're running in a Docker container
// and returns the container ID
func getContainerID() string {
	// Method 1: Check /proc/1/cpuset
	data, err := os.ReadFile("/proc/1/cpuset")
	if err == nil {
		cpuset := strings.TrimSpace(string(data))
		if strings.HasPrefix(cpuset, "/docker/") {
			parts := strings.Split(cpuset, "/")
			if len(parts) >= 3 {
				return parts[2][:12] // Return short ID
			}
		}
	}

	// Method 2: Check /proc/self/cgroup
	data, err = os.ReadFile("/proc/self/cgroup")
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.Contains(line, "docker") {
				parts := strings.Split(line, "/")
				if len(parts) > 0 {
					id := parts[len(parts)-1]
					if len(id) >= 12 {
						return id[:12]
					}
				}
			}
		}
	}

	// Method 3: Check hostname in Docker format
	hostname, _ := os.Hostname()
	if len(hostname) == 12 {
		// Docker often sets hostname to short container ID
		return hostname
	}

	return ""
}

// getContainerStartTime gets the start time of a container
func getContainerStartTime(containerID string) string {
	cmd := exec.Command("docker", "inspect", "-f", "{{.State.StartedAt}}", containerID)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	timeStr := strings.TrimSpace(string(output))
	t, err := time.Parse(time.RFC3339Nano, timeStr)
	if err != nil {
		return timeStr
	}

	return t.Format(time.RFC3339)
}

// GetLogs returns recent application logs (placeholder)
func (h *DeployHandler) GetLogs(c *gin.Context) {
	// For now, return empty logs
	// In a real implementation, you'd read from a log file or Docker logs
	c.JSON(http.StatusOK, gin.H{
		"logs": []string{},
	})
}
