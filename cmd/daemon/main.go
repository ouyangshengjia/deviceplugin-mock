/*
Copyright 2025 The Volcano Authors.

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
	"flag"
	"os"

	"k8s.io/klog/v2"

	_ "volcano.sh/deviceplugin-mock/pkg/daemon/controller"
	_ "volcano.sh/deviceplugin-mock/pkg/daemon/customized"
	"volcano.sh/deviceplugin-mock/pkg/daemon/framework"
	_ "volcano.sh/deviceplugin-mock/pkg/daemon/podmonitor"
)

func main() {
	args := initFlags()
	klog.InitFlags(nil)
	flag.Parse()
	flag.VisitAll(func(f *flag.Flag) {
		klog.V(1).Infof("FLAG: --%s=%q", f.Name, f.Value)
	})

	klog.V(1).Info("device plugin mock daemon starting")

	if err := framework.Initialize(args); err != nil {
		klog.ErrorS(err, "failed to initialize framework")
		exit(1)
	}

	framework.Run()
	exit(0)
}

func initFlags() *framework.Args {
	var config framework.Args
	flag.StringVar(&config.CustomizedService, "customized", "", "Enable selected customized service")
	flag.IntVar(&config.KubeClientQPS, "kube-cli-qps", 10, "Kube client QPS")
	flag.IntVar(&config.KubeClientBurst, "kube-cli-burst", 10, "Kube client burst")
	return &config
}

func exit(code int) {
	klog.V(1).InfoS("device plugin mock daemon stopped", "code", code)
	klog.Flush()
	os.Exit(code)
}
