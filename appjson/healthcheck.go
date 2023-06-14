package appjson

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	retry "github.com/avast/retry-go"
	"github.com/docker/docker/api/types"
	"github.com/go-resty/resty/v2"
	"github.com/moby/moby/client"
	tcexec "github.com/testcontainers/testcontainers-go/exec"

	"docker-container-healthchecker/logger"
)

type AppJSON struct {
	Healthchecks map[string][]Healthcheck `json:"healthchecks"`
}

type Healthcheck struct {
	Attempts     int        `json:"attempts,omitempty"`
	Command      []string   `json:"command,omitempty"`
	Content      string     `json:"content,omitempty"`
	InitialDelay int        `json:"initialDelay,omitempty"`
	Name         string     `json:"name,omitempty"`
	Path         string     `json:"path,omitempty"`
	Timeout      int        `json:"timeout,omitempty"`
	Type         string     `json:"type,omitempty"`
	Uptime       int        `json:"uptime,omitempty"`
	Wait         int        `json:"wait,omitempty"`
	OnFailure    *OnFailure `json:"onFailure,omitempty"`
}

type OnFailure struct {
	Command []string `json:"command,omitempty"`
	Url     string   `json:"url,omitempty"`
}

type HealthcheckContext struct {
	Headers []string
	Network string
	Port    int
}

func (h Healthcheck) GetInitialDelay() int {
	if h.InitialDelay <= 0 {
		return 0
	}

	return h.InitialDelay
}

func (h Healthcheck) GetName() string {
	if h.Name != "" {
		return h.Name
	}

	out, err := json.Marshal(h)
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString([]byte(out))
}

func (h Healthcheck) GetPath() string {
	if h.Path == "" {
		return "/"
	}

	return h.Path
}

func (h Healthcheck) GetRetries() int {
	defaultAttempts := 5
	if h.Attempts <= 0 {
		return defaultAttempts - 1
	}

	return h.Attempts - 1
}

func (h Healthcheck) GetTimeout() int {
	if h.Timeout <= 0 {
		return 5
	}

	return h.Timeout
}

func (h Healthcheck) GetWait() int {
	if h.Wait <= 0 {
		return 5
	}

	return h.Wait
}

func (h Healthcheck) Validate() error {
	if len(h.Command) > 0 {
		if h.Path != "" {
			return fmt.Errorf("healthcheck name='%s' cannot contain both a container 'command' to execute and an http 'path' to check", h.GetName())
		} else if h.Uptime > 0 {
			return fmt.Errorf("healthcheck name='%s' cannot contain both a container 'command' to execute and an 'uptime' seconds value", h.GetName())
		}
	}

	if h.Path != "" && h.Uptime > 0 {
		return fmt.Errorf("healthcheck name='%s' cannot contain both an http 'path' to check and an 'uptime' seconds value", h.GetName())
	}

	return nil
}

func (h Healthcheck) Execute(container types.ContainerJSON, ctx HealthcheckContext) ([]byte, []error) {
	if err := h.Validate(); err != nil {
		return []byte{}, []error{err}
	}

	if len(h.Command) > 0 {
		return h.executeCommandCheck(container)
	}

	if h.Path != "" {
		return h.executePathCheck(container, ctx)
	}

	return h.executeUptimeCheck(container)
}

func (h Healthcheck) HandleFailure(errors []error) error {
	if h.OnFailure == nil {
		return nil
	}
	if len(h.OnFailure.Command) > 0 {
		cmd := exec.Command(h.OnFailure.Command[0], h.OnFailure.Command[1:]...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to execute on failure command: %s", err)
		}
	}
	if len(h.OnFailure.Url) > 0 {
		data := map[string]interface{}{
			"healthcheck_name": h.GetName(),
			"errors":           errors,
		}
		json_data, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to encode data as JSON: %s", err)
		}
		response, err := http.Post(h.OnFailure.Url, "application/json", bytes.NewBuffer(json_data))
		if err != nil {
			return fmt.Errorf("failed to send data to URL: %s", err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("post request failed with status: %s", response.Status)
		}
	}
	return nil
}

func (h Healthcheck) executeCommandCheck(container types.ContainerJSON) ([]byte, []error) {
	var reader io.Reader
	err := retry.Do(
		func() error {
			var rerr error
			reader, rerr = h.dockerExec(container, h.Command)
			return rerr
		},
		retry.Attempts(uint(h.Attempts)),
		retry.Delay(time.Duration(h.GetWait())*time.Second),
	)

	b, berr := io.ReadAll(reader)
	if berr != nil {
		return []byte{}, []error{berr}
	}

	return b, err.(retry.Error).WrappedErrors()
}

func (h Healthcheck) dockerExec(container types.ContainerJSON, cmd []string, options ...tcexec.ProcessOption) (io.Reader, error) {
	var ctx context.Context
	if h.GetTimeout() > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(h.GetTimeout())*time.Second)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, err
	}

	response, err := cli.ContainerExecCreate(ctx, container.ID, types.ExecConfig{
		Cmd:          cmd,
		Detach:       false,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return nil, err
	}

	hijack, err := cli.ContainerExecAttach(ctx, response.ID, types.ExecStartCheck{})
	if err != nil {
		return nil, err
	}

	opt := &tcexec.ProcessOptions{
		Reader: hijack.Reader,
	}

	for _, o := range options {
		o.Apply(opt)
	}

	var exitCode int
	for {
		execResp, err := cli.ContainerExecInspect(ctx, response.ID)
		if err != nil {
			return nil, err
		}

		if !execResp.Running {
			exitCode = execResp.ExitCode
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	if exitCode != 0 {
		return opt.Reader, fmt.Errorf("non-zero exit code %d", exitCode)
	}
	return opt.Reader, nil
}

func (h Healthcheck) executePathCheck(container types.ContainerJSON, ctx HealthcheckContext) ([]byte, []error) {
	endpoint, ok := container.NetworkSettings.Networks[ctx.Network]
	if !ok {
		return []byte{}, []error{fmt.Errorf("inspect container: container '%s' not connected to network '%s'", container.ID, ctx.Network)}
	}

	client := resty.New()
	client.SetLogger(logger.CreateLogger())
	client.SetRetryCount(h.GetRetries())
	client.SetRetryWaitTime(time.Duration(h.GetWait()) * time.Second)
	if h.GetTimeout() > 0 {
		client.SetTimeout(time.Duration(h.GetTimeout()) * time.Second)
	}

	for _, header := range ctx.Headers {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) != 2 {
			return []byte{}, []error{fmt.Errorf("invalid header, must be delimited by ':' (colon) character: '%s'", header)}
		}

		client.SetHeader(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
	}

	resp, err := client.R().
		Get(fmt.Sprintf("http://%s:%d%s", endpoint.IPAddress, ctx.Port, h.GetPath()))
	if err != nil {
		return []byte{}, []error{err}
	}

	body := resp.Body()
	if resp.StatusCode() < 200 {
		return body, []error{fmt.Errorf("unexpected status code: %d", resp.StatusCode())}
	}

	if resp.StatusCode() >= 400 {
		return body, []error{fmt.Errorf("unexpected status code: %d", resp.StatusCode())}
	}

	if h.Content != "" && !bytes.Contains(body, []byte(h.Content)) {
		return body, []error{fmt.Errorf("unable to find expected content in response body: %s", h.Content)}
	}

	return body, []error{}
}

func (h Healthcheck) executeUptimeCheck(container types.ContainerJSON) ([]byte, []error) {
	tt, err := time.Parse(time.RFC3339Nano, container.State.StartedAt)
	if err != nil {
		return []byte{}, []error{err}
	}
	delay := 0
	uptime := int(time.Since(tt).Seconds())
	if uptime < h.Uptime {
		delay = h.Uptime - uptime
	}

	if delay > 0 {
		time.Sleep(time.Duration(delay) * time.Second)
	}

	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return []byte{}, []error{err}
	}

	container, err = cli.ContainerInspect(context.Background(), container.ID)
	if err != nil {
		return []byte{}, []error{err}
	}

	status := fmt.Sprintf("state=%s", container.State.Status)
	if !container.State.Running {
		return []byte(status), []error{errors.New("container state is not running")}
	}

	if container.RestartCount > 0 {
		return []byte(status), []error{fmt.Errorf("container has restarted %d times", container.RestartCount)}
	}

	return []byte(status), []error{}
}
