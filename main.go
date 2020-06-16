/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"time"

        "k8s.io/api/core/v1"
        "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"


        v1alpha1 "pkg/apis/topocontroller/v1alpha1"
	clientset "pkg/generated/clientset/versioned"
	informers "pkg/generated/informers/externalversions"

	"pkg/signals"
)

var (
	masterURL  string
	kubeconfig string
	isclient   bool
	watch      bool
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	exampleClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building example clientset: %s", err.Error())
	}

	if isclient {
		topocontroller := exampleClient.TopocontrollerV1alpha1()

		if topocontroller == nil {
			klog.Fatalf("Can't get TopocontrollerV1alpha1")
		}

		namespace := "default"

		resourceTopology := topocontroller.NodeResourceTopologies(namespace)

		if resourceTopology == nil {
		   klog.Fatalf("Can't get resource topology interface!")
		}

		resources := v1.ResourceList {
				v1.ResourceName("cpu"): *resource.NewQuantity(int64(2), resource.DecimalSI),
				v1.ResourceName("nic1"): *resource.NewQuantity(int64(3), resource.DecimalSI),
			}

		resTopo, err := resourceTopology.Create(context.TODO(), &v1alpha1.NodeResourceTopology{
			ObjectMeta: metav1.ObjectMeta {
				Name: "node-test1",
			},
			Nodes: []v1alpha1.NUMANodeResource {
				{ NUMAID: 1, Resources: resources },
				{ NUMAID: 2, Resources: resources },
			},
		}, metav1.CreateOptions{})
		if err != nil {
			klog.Fatalf("Failed to create v1alpha1.NodeResourceTopology!")
		}
		klog.Infof("resTopo: %v", resTopo)
	}

	if watch {
		kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
		exampleInformerFactory := informers.NewSharedInformerFactory(exampleClient, time.Second*30)

		controller := NewController(kubeClient, exampleClient,
			exampleInformerFactory.Topocontroller().V1alpha1().NodeResourceTopologies())

		// notice that there is no need to run Start methods in a separate goroutine. (i.e. go kubeInformerFactory.Start(stopCh)
		// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
		kubeInformerFactory.Start(stopCh)
		exampleInformerFactory.Start(stopCh)

		if err = controller.Run(2, stopCh); err != nil {
			klog.Fatalf("Error running controller: %s", err.Error())
		}
	}
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.BoolVar(&isclient, "isclient", false, "Check clientset Create")
	flag.BoolVar(&watch, "watch", false, "Watch for new CRD")
}
