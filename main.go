package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/devfile/api/pkg/apis/workspaces/v1alpha2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	maxContainers       = 10
	iterations          = 10
	containerImage      = "quay.io/eclipse/che-nodejs10-ubi:nightly"
	containerMemRequest = "32Mi"
	dwName              = "timing-test"
	dwNamespace         = "timing-test"
)

var dwNamespacedName = types.NamespacedName{
	Name:      dwName,
	Namespace: dwNamespace,
}

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha2.AddToScheme(scheme))

}

func getContainerComponent(idx int) *v1alpha2.ContainerComponent {
	container := &v1alpha2.ContainerComponent{}
	container.Image = containerImage
	container.MemoryLimit = containerMemRequest
	container.Endpoints = []v1alpha2.Endpoint{
		{
			Name:       fmt.Sprintf("container-%d", idx),
			Protocol:   "http",
			TargetPort: 8080 + idx,
		},
	}
	return container
}

func getDevWorkspace(numContainers int) *v1alpha2.DevWorkspace {
	dw := &v1alpha2.DevWorkspace{}
	dw.Name = dwName
	dw.Namespace = dwNamespace
	dw.Spec.Started = true
	theiaComponent := v1alpha2.Component{}
	theiaComponent.Name = "theia-ide"
	theiaComponent.Plugin = &v1alpha2.PluginComponent{}
	theiaComponent.Plugin.Id = "eclipse/che-theia/latest"
	machineExecComponent := v1alpha2.Component{}
	machineExecComponent.Name = "machine-exec"
	machineExecComponent.Plugin = &v1alpha2.PluginComponent{}
	machineExecComponent.Plugin.Id = "eclipse/che-machine-exec-plugin/latest"
	dw.Spec.Template.Components = []v1alpha2.Component{theiaComponent, machineExecComponent}
	for i := 0; i < numContainers; i++ {
		component := v1alpha2.Component{}
		component.Name = fmt.Sprintf("component-%d", i)
		component.Container = getContainerComponent(i)
		dw.Spec.Template.Components = append(dw.Spec.Template.Components, component)
	}
	return dw
}

func createDevWorkspace(c client.Client, dw *v1alpha2.DevWorkspace) *v1alpha2.DevWorkspace {
	err := c.Create(context.TODO(), dw)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	var clusterDW v1alpha2.DevWorkspace
WAIT_RUNNING:
	for {
		err := c.Get(context.TODO(), dwNamespacedName, &clusterDW)
		if err != nil {
			fmt.Printf("Error getting devworkspace: %s\n", err.Error())
			continue
		}
		switch clusterDW.Status.Phase {
		case v1alpha2.WorkspaceStatusStarting:
			fmt.Println("Workspace still starting")
			time.Sleep(1 * time.Second)
			continue
		case v1alpha2.WorkspaceStatusRunning:
			break WAIT_RUNNING
		case "":
			fmt.Println("Workspace phase empty")
			time.Sleep(1 * time.Second)
			continue
		default:
			fmt.Printf("Workspace phase is %q; this is unexpected\n", dw.Status.Phase)
			os.Exit(1)
		}
	}
	return &clusterDW
}

func writeTimingData(f *os.File, dw *v1alpha2.DevWorkspace, numContainers int) {
	annot := dw.Annotations
	delete(annot, "kubectl.kubernetes.io/last-applied-configuration")
	annot["numContainers"] = fmt.Sprintf("%d", numContainers)
	bytes, err := json.MarshalIndent(annot, "", "  ")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if _, err := fmt.Fprintf(f, "%s\n", string(bytes)); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func deleteDevWorkspace(c client.Client, dw *v1alpha2.DevWorkspace) {
	err := c.Delete(context.TODO(), dw)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	f, err := os.OpenFile("startup.log",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()
	c := getClient()
	for numContainers := 1; numContainers <= maxContainers; numContainers++ {
		for iter := 0; iter < iterations; iter++ {
			fmt.Printf("Containers %d; Iteration %d\n", numContainers, iter)
			dw := createDevWorkspace(c, getDevWorkspace(numContainers))
			writeTimingData(f, dw, numContainers)
			deleteDevWorkspace(c, dw)
		}
	}
}

func getClient() client.Client {
	cfg, err := config.GetConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return c
}
