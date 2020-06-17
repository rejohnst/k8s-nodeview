package main

import (
	"flag"
	"fmt"
	"os"

	apiV1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const VERSION="2020-06-16"

type kubeClient struct {
	kcClientset		*kubernetes.Clientset
	kcVerbose		bool
}

func printContainers(pod *apiV1.Pod) {
	for _, container := range pod.Spec.Containers {
		fmt.Printf("    container: %-30s image: %s\n", container.Name, container.Image)
	}
}

func printNode(node *apiV1.Node) {
	fmt.Printf("%-20s %s\n", "OS Image:", node.Status.NodeInfo.OSImage)
	fmt.Printf("%-20s %s\n", "Kernel Version:", node.Status.NodeInfo.KernelVersion)
	fmt.Printf("%-20s %s\n", "CRI Version:", node.Status.NodeInfo.ContainerRuntimeVersion)
	fmt.Printf("%-20s %s\n", "Kubelet Version:", node.Status.NodeInfo.KubeletVersion)
	fmt.Printf("%-20s %s\n", "IP Address:", node.Status.Addresses[0].Address)

}

func listNodes(client *kubeClient, nodename *string) {
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
	nodes, err := client.kcClientset.CoreV1().Nodes().List(nodeListOptions)
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
		listOptions := v1.ListOptions{
			FieldSelector: selector,
		}

		pods, err := client.kcClientset.CoreV1().Pods("").List(listOptions)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting pods for node %s: %v\n", node.Name, err)
			os.Exit(1)
		}

		for _, pod := range pods.Items {
			fmt.Printf("  pod: %s\n", pod.Name)
			if client.kcVerbose {
				printContainers(&pod)
			}
		}
		fmt.Printf("\n")
	}
}

func findPod(client *kubeClient, podname *string) {
	pods, err := client.kcClientset.CoreV1().Pods("").List(v1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting pods: %v\n", err)
		os.Exit(1)
	}

	for _, pod := range pods.Items {
		if pod.Name == *podname {
			if client.kcVerbose {
				nodeListOptions := v1.ListOptions{}
				nodeSelector := fmt.Sprintf("metadata.name=%s", pod.Spec.NodeName)
				nodeListOptions.FieldSelector = nodeSelector

				nodes, err := client.kcClientset.CoreV1().Nodes().List(nodeListOptions)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error getting nodes: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("%-20s %s\n", "Node Name:", pod.Spec.NodeName)
				printNode(&nodes.Items[0])
			} else {
				fmt.Printf("%s\n", pod.Spec.NodeName)
			}
			return
		}
	}
	fmt.Printf("couldn't find pod %s\n", *podname)
}

func main() {
	var client kubeClient
	var kubeconfig, cmd, nodename, podname *string
	var verbose, version *bool

	// Parse the command line arguments
	kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	cmd = flag.String("command", "", "<list|findpod>")
	nodename = flag.String("nodename", "", "name of node to print info for (default=all nodes)")
	podname = flag.String("podname", "", "show info for node which hosts specified pod")
	verbose = flag.Bool("verbose", false, "enable verbose mode")
	version = flag.Bool("version", false, "print program version and exit")
	flag.Parse()

	if *version {
		fmt.Printf("%s\n", VERSION)
		os.Exit(0)
	}

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
	client.kcClientset = clientset
	client.kcVerbose = *verbose

	switch *cmd {
	case "":
		fmt.Fprintf(os.Stderr, "no command specified\n")
	case "list":
		listNodes(&client, nodename)
	case "findpod":
		if *podname == "" {
			fmt.Fprintf(os.Stderr, "podname not specified\n")
			os.Exit(2)
		}
		findPod(&client, podname)
	default:
		fmt.Fprintf(os.Stderr, "Invalid command: %s\n", *cmd)
		os.Exit(2)
	}

	os.Exit(0)
}
