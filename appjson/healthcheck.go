package appjson

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"

	retry "github.com/avast/retry-go"
	"github.com/docker/docker/api/types"
	"github.com/go-resty/resty/v2"
	"github.com/moby/moby/client"
	tcexec "github.com/testcontainers/testcontainers-go/exec"

	"docker-container-healthchecks/http"
)

type AppJSON struct {
	Healthchecks map[string][]Healthcheck `json:"healthchecks"`
}

type Healthcheck struct {
	Type                string   `json:"type"`
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	Path                string   `json:"path"`
	Content             string   `json:"content"`
	InitialDelaySeconds int      `json:"initialDelaySeconds"`
	Wait                int      `json:"wait"`
	Attempts            int      `json:"attempts"`
	Command             []string `json:"command"`
}

func (h Healthcheck) GetInitialDelay() int {
	if h.InitialDelaySeconds <= 0 {
		return 0
	}

	return h.InitialDelaySeconds
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
	if h.Attempts <= 0 {
		return 0
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
			return fmt.Errorf("healthchecks '%s' cannot contain both an http 'path' to check and a container 'command' to execute", h.GetName())
		}

		if h.Content != "" {
			return fmt.Errorf("command healthcheck '%s' cannot specify content to expect", h.GetName())
		}
	}
	return nil
}

func (h Healthcheck) Execute(container types.ContainerJSON, containerPort int) ([]byte, []error) {
	if err := h.Validate(); err != nil {
		return []byte{}, []error{err}
	}

	if h.Path != "" {
		return h.executePathCheck(container, containerPort)
	}

	return h.executeCommandCheck(container)
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
	client.SetDebug(true)
	client.EnableTrace()
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

	if h.Content != "" && !bytes.Contains(resp.Body(), []byte(h.Content)) {
		return resp.Body(), []error{fmt.Errorf("unable to find expected content: %s", h.Content)}
	}

	return resp.Body(), []error{}
}
