package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

var version = "1.0.0"

type rc int

const (
	OK rc = iota
	WARNING
	CRITICAL
)

func (s rc) String() (result string) {
	switch s {
	case OK:
		result = "OK: "
	case WARNING:
		result = "WARNING: "
	case CRITICAL:
		result = "CRITICAL: "
	}
	return result
}

func displayResult(r rc, msg string) {
	fmt.Printf(r.String() + msg)
	os.Exit(int(r))
}

func getNodeState(ctx context.Context, cli *client.Client, warning, critical int) {
	// non available nodes
	nodes, err := cli.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		msg := fmt.Sprintf("%s\n", err.Error())
		displayResult(CRITICAL, msg)
	}

	var nodes_down []string
	for _, v := range nodes {
		if v.Status.State == "down" {
			nodes_down = append(nodes_down, v.Description.Hostname)
		}
	}

	if len(nodes_down) == 0 {
		msg := fmt.Sprintf("%d nodes available\n", len(nodes))
		displayResult(OK, msg)
	}
	if len(nodes_down) >= critical {
		msg := fmt.Sprintf("%d/%d node(s) down: %s\n", len(nodes_down), len(nodes), strings.Join(nodes_down, ", "))
		displayResult(CRITICAL, msg)
	}
	if len(nodes_down) >= warning {
		msg := fmt.Sprintf("%d/%d node(s) down: %s\n", len(nodes_down), len(nodes), strings.Join(nodes_down, ", "))
		displayResult(WARNING, msg)
	}
}

func getServiceState(s string, ctx context.Context, cli *client.Client) {
	// services including replica
	var filter = filters.NewArgs(filters.Arg("name", s))
	services, err := cli.ServiceList(ctx, types.ServiceListOptions{Filters: filter, Status: true})
	if err != nil {
		msg := fmt.Sprintf("%s\n", err.Error())
		displayResult(CRITICAL, msg)
	}
	var running uint64
	var desired uint64
	if len(services) == 0 {
		msg := fmt.Sprintf("Could not find service %s!\n", s)
		displayResult(CRITICAL, msg)
	}
	for _, v := range services {
		if v.ServiceStatus != nil {
			if v.Spec.Name == s {
				running = v.ServiceStatus.RunningTasks
				desired = v.ServiceStatus.DesiredTasks
			}
		} else {
			msg := fmt.Sprintf("Could not receive ServiceStatus for %s, maybe API does not support it!\n", s)
			displayResult(CRITICAL, msg)
		}
	}
	if running == 0 && desired != 0 {
		msg := fmt.Sprintf("No service for %s is running!\n", s)
		displayResult(CRITICAL, msg)
	}

	if running != desired {
		msg := fmt.Sprintf("%d/%d services of %s are running.\n", running, desired, s)
		displayResult(WARNING, msg)
	}
	msg := fmt.Sprintf("%d/%d services of %s are running.\n", running, desired, s)
	displayResult(OK, msg)
}

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage of %s:\n\n", filepath.Base(os.Args[0]))
		fmt.Printf("It displays the state of nodes or a service of a docker swarm cluster.\n\n")
		fmt.Printf("Version: %s - https://github.com/EmmaTinten/go-monitoring-plugins\n\n", version)
		flag.PrintDefaults()
	}
	var flagNodes = flag.Bool("n", false, "check swarm node state")
	var flagNodesWarning = flag.Int("w", 1, "number of nodes down for WARNING state")
	var flagNodesCritical = flag.Int("c", 1, "number of nodes down for CRITICAL state")
	var flagService = flag.String("s", "", "check named service state")
	flag.Parse()

	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// connection
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		msg := fmt.Sprintf("%s\n", err.Error())
		displayResult(CRITICAL, msg)
	}

	if *flagNodes {
		getNodeState(ctx, cli, *flagNodesWarning, *flagNodesCritical)
	}

	if *flagService != "" {
		getServiceState(*flagService, ctx, cli)
	}
}
