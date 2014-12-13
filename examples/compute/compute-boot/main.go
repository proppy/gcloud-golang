// Copyright 2014 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// instance is a sample application using  the computeutil package.
package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	cloud "github.com/proppy/gcloud-golang"
	"github.com/proppy/gcloud-golang/compute/computeutil"
	"github.com/proppy/oauth2"
	"github.com/proppy/oauth2/google"
)

var (
	jsonFile   = flag.String("json", "", "A path to your JSON key file for your service account downloaded from Google Developer Console, not needed if you run it on Compute Engine instances.")
	gcloudCred = flag.Bool("gcloud", false, "If true, reuse gcloud credentials for authorization")

	projID            = flag.String("project", "", "The ID of your Google Cloud project.")
	name              = flag.String("name", "gcloud-computeutil-instance", "The name of the instance to create.")
	image             = flag.String("image", "projects/google-containers/global/images/container-vm-v20141208", "The image to use for the instance.")
	zone              = flag.String("zone", "us-central1-f", "The zone for the instance.")
	machineType       = flag.String("machine", "f1-micro", "The machine type for the instance.")
	startupScriptPath = flag.String("startup", "", "The path to the startup script for the instance")
)

func main() {
	flag.Parse()
	if (*jsonFile == "" && !*gcloudCred) || (*jsonFile != "" && *gcloudCred) || *projID == "" {
		flag.PrintDefaults()
		log.Fatalf("Please specify either gcloud or JSON credentials and a project ID.")
	}
	var metadata map[string]string
	if *startupScriptPath != "" {
		startupScript, err := ioutil.ReadFile(*startupScriptPath)
		if err != nil {
			log.Fatalf("Error reading startup script %q: %v", startupScriptPath, err)
		}
		metadata = map[string]string{
			"startup-script": string(startupScript),
		}
		log.Println(metadata)
	}
	t, err := getTransport()
	if err != nil {
		log.Fatalf("failed to create transport: %v", err)
	}
	client := &http.Client{Transport: t}
	ctx := cloud.WithZone(cloud.NewContext(*projID, client), *zone)
	var instance *computeutil.Instance
	instance, err = computeutil.GetInstance(ctx, *name)
	if err != nil { // not found
		instance, err = computeutil.NewInstance(ctx, &computeutil.Instance{
			Name:        *name,
			Image:       *image,
			MachineType: *machineType,
			Metadata:    metadata,
		})
		if err != nil {
			log.Fatalf("failed to create instance %q: %v", *name, err)
		}
	}
	log.Printf("instance %q ready: %#v", *name, instance)
	io.Copy(os.Stdout, instance.SerialPortOutput(ctx))
}

func getTransport() (*oauth2.Transport, error) {
	if *gcloudCred {
		flow, err := oauth2.New(
			google.GcloudCredentials(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create oauth2 flow from gcloud credentials")
		}
		t := flow.NewTransport()
		if err := t.Refresh(); err != nil {
			return nil, fmt.Errorf("failed to refresh token: %v", err)
		}
		return t, nil
	}
	flow, err := oauth2.New(
		google.ServiceAccountJSONKey(*jsonFile),
		oauth2.Scope(computeutil.ScopeCompute),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create oauth flow from json file %q: %v", err)
	}
	return flow.NewTransport(), nil
}
