package appjson

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	retry "github.com/avast/retry-go"
	"github.com/docker/docker/api/types"
	"github.com/go-resty/resty/v2"
	"github.com/moby/moby/client"
	tcexec "github.com/testcontainers/testcontainers-go/exec"

	"docker-container-healthchecker/http"
)

type AppJSON struct {
	Healthchecks map[string][]Healthcheck `json:"healthchecks"`
}

type Healthcheck struct {
	Attempts     int      `json:"attempts"`
	Command      []string `json:"command"`
	InitialDelay int      `json:"initialDelay"`
	Name         string   `json:"name"`
	Path         string   `json:"path"`
	Type         string   `json:"type"`
	Uptime       int      `json:"uptime"`
	Wait         int      `json:"wait"`
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
	defaultAttempts := 3
	if h.Attempts <= 0 {
		return defaultAttempts - 1
	}

	return h.Attempts - 1
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

func (h Healthcheck) Execute(container types.ContainerJSON, containerPort int) ([]byte, []error) {
	if err := h.Validate(); err != nil {
		return []byte{}, []error{err}
	}

	if len(h.Command) > 0 {
		return h.executeCommandCheck(container)
	}

	if h.Path != "" {
		return h.executePathCheck(container, containerPort)
	}

	return h.executeUptimeCheck(container)
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
	)

	b, berr := io.ReadAll(reader)
	if berr != nil {
		return []byte{}, []error{berr}
	}

	return b, err.(retry.Error).WrappedErrors()
}

func (h Healthcheck) dockerExec(container types.ContainerJSON, cmd []string, options ...tcexec.ProcessOption) (io.Reader, error) {
	var ctx context.Context
	if h.GetWait() > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(h.GetWait())*time.Second)
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

func (h Healthcheck) executePathCheck(container types.ContainerJSON, containerPort int) ([]byte, []error) {
	networkName := "bridge"
	endpoint, ok := container.NetworkSettings.Networks[networkName]
	if !ok {
		return []byte{}, []error{fmt.Errorf("inspect container: container '%s' not connected to network '%s'", container.ID, networkName)}
	}

	client := resty.New()
	client.SetLogger(http.CreateLogger())
	if h.GetRetries() > 0 {
		client.SetRetryCount(h.GetRetries())
	}
	if h.GetWait() > 0 {
		client.SetTimeout(time.Duration(h.GetWait()) * time.Second)
	}

	resp, err := client.R().
		Get(fmt.Sprintf("http://%s:%d%s", endpoint.IPAddress, containerPort, h.GetPath()))
	if err != nil {
		return []byte{}, []error{err}
	}

	if resp.StatusCode() < 200 {
		return resp.Body(), []error{fmt.Errorf("unexpected status code: %d", resp.StatusCode())}
	}

	if resp.StatusCode() >= 400 {
		return resp.Body(), []error{fmt.Errorf("unexpected status code: %d", resp.StatusCode())}
	}

	return resp.Body(), []error{}
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

	return []byte(status), []error{}
}
