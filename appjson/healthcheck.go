package appjson

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/alexellis/go-execute/v2"
	retry "github.com/avast/retry-go"
	"github.com/docker/docker/api/types"
	"github.com/go-resty/resty/v2"
	"github.com/moby/moby/client"

	"docker-container-healthchecker/logger"
)

type CheckType int

const (
	CommandCheck CheckType = iota
	ListeningCheck
	PathCheck
	UptimeCheck
)

type AppJSON struct {
	Healthchecks map[string][]Healthcheck `json:"healthchecks"`
}

type Healthcheck struct {
	Attempts     int          `json:"attempts,omitempty"`
	Command      []string     `json:"command,omitempty"`
	Content      string       `json:"content,omitempty"`
	HTTPHeaders  []HTTPHeader `json:"httpHeaders,omitempty"`
	InitialDelay int          `json:"initialDelay,omitempty"`
	Listening    bool         `json:"listening,omitempty"`
	Name         string       `json:"name,omitempty"`
	Path         string       `json:"path,omitempty"`
	Port         int          `json:"port,omitempty"`
	Scheme       string       `json:"scheme,omitempty"`
	Timeout      int          `json:"timeout,omitempty"`
	Type         string       `json:"type,omitempty"`
	Uptime       int          `json:"uptime,omitempty"`
	Wait         int          `json:"wait,omitempty"`
	Warn         bool         `json:"warn,omitempty"`
	OnFailure    *OnFailure   `json:"onFailure,omitempty"`
}

type HTTPHeader struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

type OnFailure struct {
	Command []string `json:"command,omitempty"`
	Url     string   `json:"url,omitempty"`
}

type HealthcheckContext struct {
	Headers   []string
	IPAddress string
	Network   string
	Port      int
}

func (h Healthcheck) GetAttempts() int {
	defaultAttempts := 3
	if h.Attempts <= 0 {
		return defaultAttempts
	}

	return h.Attempts
}

func (h Healthcheck) GetCheckType() CheckType {
	if h.Listening {
		return ListeningCheck
	}

	if len(h.Command) > 0 {
		return CommandCheck
	}

	if h.Path != "" {
		return PathCheck
	}

	return UptimeCheck
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
	attempts := h.GetAttempts()
	return attempts - 1
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
		} else if h.Listening {
			return fmt.Errorf("healthcheck name='%s' cannot contain both a container 'command' to execute and a 'listening' true value", h.GetName())
		}
	}

	if h.Path != "" {
		if h.Uptime > 0 {
			return fmt.Errorf("healthcheck name='%s' cannot contain both an http 'path' to check and an 'uptime' seconds value", h.GetName())
		} else if h.Listening {
			return fmt.Errorf("healthcheck name='%s' cannot contain both an http 'path' to check and a 'listening' true value", h.GetName())
		}
	}

	if h.Uptime > 0 && h.Listening {
		return fmt.Errorf("healthcheck name='%s' cannot contain both an 'uptime' seconds value and a 'listening' true value", h.GetName())
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

	if h.Listening {
		return h.executeListenerCheck(container)
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
	var b []byte
	err := retry.Do(
		func() error {
			var rerr error
			b, rerr = h.dockerExec(container)
			return rerr
		},
		retry.Attempts(uint(h.GetAttempts())),
		retry.Delay(time.Duration(h.GetWait())*time.Second),
	)

	if err != nil {
		return b, err.(retry.Error).WrappedErrors()
	}

	return b, nil
}

func (h Healthcheck) dockerExec(container types.ContainerJSON) ([]byte, error) {
	ctx := context.Background()
	if h.GetTimeout() > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(h.GetTimeout())*time.Second)
		defer cancel()
	}

	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, err
	}

	response, err := cli.ContainerExecCreate(ctx, container.ID, types.ExecConfig{
		Cmd:          h.Command,
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
	defer hijack.Close()

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

	b, _ := io.ReadAll(hijack.Reader)
	if exitCode != 0 {
		return b, fmt.Errorf("non-zero exit code %d", exitCode)
	}
	return b, nil
}

func (h Healthcheck) executePathCheck(container types.ContainerJSON, ctx HealthcheckContext) ([]byte, []error) {
	ipAddress := ctx.IPAddress
	if ipAddress == "" {
		endpoint, ok := container.NetworkSettings.Networks[ctx.Network]
		if !ok {
			return []byte{}, []error{fmt.Errorf("inspect container: container '%s' not connected to network '%s'", container.ID, ctx.Network)}
		}

		ipAddress = endpoint.IPAddress
	}

	client := resty.New()
	client.RemoveProxy()
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

	for _, header := range h.HTTPHeaders {
		client.SetHeader(header.Name, header.Value)
	}

	client.SetHeader("Accept", "*/*")

	scheme := strings.ToLower(h.Scheme)
	if scheme == "" {
		scheme = "http"
	}

	validSchemes := map[string]bool{
		"http":  true,
		"https": true,
	}
	if !validSchemes[scheme] {
		return []byte{}, []error{errors.New("invalid scheme specified, must be either http or https")}
	}

	request := client.R()
	resp, err := request.
		Get(fmt.Sprintf("%s://%s:%d%s", scheme, ipAddress, h.Port, h.GetPath()))
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

func (h Healthcheck) executeListenerCheck(container types.ContainerJSON) ([]byte, []error) {
	err := retry.Do(
		func() error {
			return h.listeningCheck(container)
		},
		retry.Attempts(uint(h.GetAttempts())),
		retry.Delay(time.Duration(h.GetWait())*time.Second),
	)

	if err != nil {
		return []byte{}, err.(retry.Error).WrappedErrors()
	}

	return []byte{}, nil
}

func (h Healthcheck) listeningCheck(container types.ContainerJSON) error {
	if !container.State.Running {
		return errors.New("container state is not running")
	}

	if container.State.Pid == 0 {
		return errors.New("container state is not running")
	}

	ctx := context.Background()
	if h.GetTimeout() > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(h.GetTimeout())*time.Second)
		defer cancel()
	}

	cmd := execute.ExecTask{
		Command:     "nsenter",
		Args:        []string{"-t", fmt.Sprint(container.State.Pid), "-n", "netstat", "-plant"},
		StreamStdio: false,
	}
	result, err := cmd.Execute(ctx)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		errorMessage := strings.TrimSpace(result.Stderr)
		if errorMessage == "nsenter: No such file or directory" {
			return errors.New("unable to enter the container to check that the process is bound to the correct port and interface: missing nsenter binary in PATH")
		}

		if strings.HasSuffix(errorMessage, "netstat: No such file or directory") {
			return errors.New("unable to enter the container to check that the process is bound to the correct port and interface: missing netstat binary in PATH")
		}

		if strings.HasPrefix(errorMessage, "nsenter: cannot open /proc/") && strings.HasSuffix(errorMessage, "No such file or directory") {
			return errors.New("unable to enter the container to check that the process is bound to the correct port and interface: ensure runtime PID namespace is host")
		}

		return fmt.Errorf("unable to enter the container to check that the process is bound to the correct port and interface: %s", errorMessage)
	}

	addresses := map[string]bool{}
	for _, line := range strings.Split(result.Stdout, "\n") {
		if !strings.Contains(line, "LISTEN") {
			continue
		}

		parts := strings.Fields(line)
		addresses[parts[3]] = true
	}

	validAddresses := map[string]bool{
		"0.0.0.0": true,
		":::":     true,
	}

	for validAddress := range validAddresses {
		if addresses[fmt.Sprintf("%s:%d", validAddress, h.Port)] {
			return nil
		}
	}

	for address := range addresses {
		portSuffix := fmt.Sprintf(":%d", h.Port)
		if strings.HasSuffix(address, portSuffix) {
			ipAddress := strings.TrimSuffix(address, portSuffix)
			ip := net.ParseIP(ipAddress)
			if ip == nil {
				return errors.New("listening ip address is not valid")
			}

			if ip.To4() == nil {
				return fmt.Errorf("container listening on expected port (%d) with unexpected IPv6 interface: expected=:: actual=%s", h.Port, ipAddress)
			}

			return fmt.Errorf("container listening on expected port (%d) with unexpected IPv4 interface: expected=0.0.0.0 actual=%s", h.Port, ipAddress)
		}

		u, err := url.ParseRequestURI(fmt.Sprintf("http://%s", address))
		if err != nil {
			return fmt.Errorf("unable to parse listening address: %w", err)
		}

		portSuffix = fmt.Sprintf(":%s", u.Port())
		ipAddress := strings.TrimSuffix(address, portSuffix)
		ip := net.ParseIP(ipAddress)
		if ip == nil {
			return errors.New("listening ip address is not valid")
		}

		if validAddresses[ipAddress] {
			if ip.To4() == nil {
				return fmt.Errorf("container listening on expected IPv6 interface with an unexpected port: expected=%d actual=%s", h.Port, u.Port())
			}

			return fmt.Errorf("container listening on expected IPv4 interface with an unexpected port: expected=%d actual=%s", h.Port, u.Port())
		}

		if ip.To4() == nil {
			return fmt.Errorf("container listening on unexpected IPv6 interface with an unexpected port: expected=:::%d actual=%s", h.Port, address)
		}

		return fmt.Errorf("container listening on unexpected IPv4 interface with an unexpected port: expected=0.0.0.0:%d actual=%s", h.Port, address)
	}

	return nil
}
