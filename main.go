package main

import (
	"flag"
	"fmt"
	"os"
	"k8s.io/client-go/kubernetes"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	apiV1 "k8s.io/api/core/v1"
)

func printContainers(pod *apiV1.Pod) {
	for _, container := range pod.Spec.Containers {
		fmt.Printf("    container: %-30s image: %s\n", container.Name, container.Image)
	}
}

func main() {
	var kubeconfig, nodename *string
	var verbose *bool

	// Parse the command line arguments
	kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	nodename = flag.String("nodename", "", "name of node to print info for (default=all nodes)")
	verbose = flag.Bool("verbose", false, "enable verbose mode")
	flag.Parse()

	//
	// Configure our K8S API client using the config file specified on the
	// command line.
	//
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating clientset: %v\n", err)
		os.Exit(1)
	}

	//
	// Get a list of nodes in the cluster.
	// If the user specified a nodename on the command line, then configure the
	// listOptions to filter the results such that only the specified node is
	// included.
	//
	nodeListOptions := v1.ListOptions{}
	if *nodename != "" {
		nodeSelector := fmt.Sprintf("metadata.name=%s", *nodename)
		nodeListOptions.FieldSelector = nodeSelector
	}
	nodes, err := clientset.CoreV1().Nodes().List(nodeListOptions)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting nodes: %v\n", err)
		os.Exit(1)
	}

	//
	// If nodes.Items is an empty list and the user specified a nodename on the
	// command line, then the specified node was not found in this cluster.
	//
	if *nodename != "" && len(nodes.Items) == 0 {
		fmt.Fprintf(os.Stderr, "node %s not found!\n", *nodename)
		os.Exit(1)
	}

	//
	// For each node, print the list of pods running on that node.  If verbose
	// mode is on, then also print details on the containers in each pod.
	//
	fmt.Printf("\n")
	for _, node := range nodes.Items {
		fmt.Printf("Node: %s\n", node.Name)

		selector := fmt.Sprintf("spec.nodeName=%s", node.Name)
		listOptions := v1.ListOptions {
			FieldSelector: selector,
		}

		pods, err := clientset.CoreV1().Pods("").List(listOptions)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting pods for node %s: %v\n", node.Name, err)
			os.Exit(1)
		}

		for _, pod := range pods.Items {
			fmt.Printf("  pod: %s\n", pod.Name)
			if *verbose {
				printContainers(&pod)
			}
		}
		fmt.Printf("\n")
	}

	os.Exit(0)
}