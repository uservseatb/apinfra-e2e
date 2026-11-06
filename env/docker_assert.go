package env

import (
	"context"
	"fmt"
	"net/netip"
	"strconv"
	"strings"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/rogpeppe/go-internal/testscript"
)

const (
	dockerAssertRunning       = "docker-assert-running"
	dockerAssertNoPublicPorts = "docker-assert-no-public-ports"
	dockerAssertPortBinding   = "docker-assert-port-binding"
	dockerStateRunning        = "running"
)

func dockerAssertRunningCmd(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) != 1 {
		ts.Fatalf("usage: %s <name-substring>", dockerAssertRunning)
	}

	substr := args[0]
	found, name, _, err := findRunningContainer(context.Background(), substr)
	if err != nil {
		ts.Fatalf("%s: %v", dockerAssertRunning, err)
	}

	if neg {
		if found {
			ts.Fatalf("%s: unexpected running container %q matching %q", dockerAssertRunning, name, substr)
		}

		return
	}

	if !found {
		ts.Fatalf("%s: no running container matching %q", dockerAssertRunning, substr)
	}
}

func dockerAssertNoPublicPortsCmd(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! %s", dockerAssertNoPublicPorts)
	}

	if len(args) != 1 {
		ts.Fatalf("usage: %s <name-substring>", dockerAssertNoPublicPorts)
	}

	substr := args[0]
	found, name, ports, err := findRunningContainer(context.Background(), substr)
	if err != nil {
		ts.Fatalf("%s: %v", dockerAssertNoPublicPorts, err)
	}

	if !found {
		ts.Fatalf("%s: no running container matching %q", dockerAssertNoPublicPorts, substr)
	}

	for _, port := range ports {
		if port.PublicPort == 0 {
			continue
		}

		ts.Fatalf(
			"%s: container %q publishes host port %d -> container %d/%s",
			dockerAssertNoPublicPorts,
			name,
			port.PublicPort,
			port.PrivatePort,
			port.Type,
		)
	}
}

// docker-assert-port-binding <name-substring> <host-ip> <public-port> <private-port>
// Checks that a running container publishes exactly that host binding.
func dockerAssertPortBindingCmd(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! %s", dockerAssertPortBinding)
	}

	if len(args) != 4 {
		ts.Fatalf(
			"usage: %s <name-substring> <host-ip> <public-port> <private-port>",
			dockerAssertPortBinding,
		)
	}

	wantIP, err := netip.ParseAddr(args[1])
	if err != nil {
		ts.Fatalf("%s: host-ip: %v", dockerAssertPortBinding, err)
	}

	wantPublic, err := strconv.ParseUint(args[2], 10, 16)
	if err != nil {
		ts.Fatalf("%s: public-port: %v", dockerAssertPortBinding, err)
	}

	wantPrivate, err := strconv.ParseUint(args[3], 10, 16)
	if err != nil {
		ts.Fatalf("%s: private-port: %v", dockerAssertPortBinding, err)
	}

	substr := args[0]
	found, name, ports, err := findRunningContainer(context.Background(), substr)
	if err != nil {
		ts.Fatalf("%s: %v", dockerAssertPortBinding, err)
	}

	if !found {
		ts.Fatalf("%s: no running container matching %q", dockerAssertPortBinding, substr)
	}

	for _, port := range ports {
		if port.PublicPort != uint16(wantPublic) || port.PrivatePort != uint16(wantPrivate) {
			continue
		}

		if !port.IP.IsValid() || port.IP != wantIP {
			ts.Fatalf(
				"%s: container %q port %d->%d/%s bound to %s, want %s",
				dockerAssertPortBinding,
				name,
				port.PublicPort,
				port.PrivatePort,
				port.Type,
				port.IP,
				wantIP,
			)
		}

		return
	}

	ts.Fatalf(
		"%s: container %q has no binding %s:%d->%d (ports=%v)",
		dockerAssertPortBinding,
		name,
		wantIP,
		wantPublic,
		wantPrivate,
		ports,
	)
}

func findRunningContainer(
	ctx context.Context,
	substr string,
) (bool, string, []container.PortSummary, error) {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return false, "", nil, fmt.Errorf("docker client: %w", err)
	}
	defer func() { _ = cli.Close() }()

	result, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		return false, "", nil, fmt.Errorf("container list: %w", err)
	}

	for _, item := range result.Items {
		name := primaryContainerName(item.Names)
		if !strings.Contains(name, substr) {
			continue
		}

		if string(item.State) == dockerStateRunning {
			return true, name, item.Ports, nil
		}
	}

	return false, "", nil, nil
}

func primaryContainerName(names []string) string {
	if len(names) == 0 {
		return ""
	}

	return strings.TrimPrefix(names[0], "/")
}
