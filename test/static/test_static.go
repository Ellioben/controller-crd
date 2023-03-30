package main

import (
	"encoding/json"
	"os"

	appsv1 "k8s.io/api/apps/v1"
)

func main() {
	ds := &appsv1.DaemonSet{}
	ds.Name = "example"
	// edit deployment spec

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "    ")
	enc.Encode(ds)
}
