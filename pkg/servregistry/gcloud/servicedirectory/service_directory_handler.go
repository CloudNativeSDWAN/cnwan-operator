// Copyright © 2020 Cisco
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// All rights reserved.

package servicedirectory

import (
	"context"
	"errors"
	"io/ioutil"
	"sync"
	"time"

	sd "cloud.google.com/go/servicedirectory/apiv1beta1"
	"github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	"github.com/go-logr/logr"
	"google.golang.org/api/option"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type servDir struct {
	project     string
	region      string
	client      *sd.RegistrationClient
	log         logr.Logger
	context     context.Context
	resMetadata map[string]string
	timeout     time.Duration
	lock        sync.Mutex
}

// NewHandler creates a handler for service directory. The handler will set up
// the library with gcloud project name, the region of
// service directory where services will registered and the path to the
// credentials file - or service account.
// Additionally, it takes a timeout value that represents the amount of time
// (in seconds) an http call can stay active before being terminated.
//
// It returns an instance of ServiceRegistry, or nil and an error in case
// something went wrong
func NewHandler(ctx context.Context, project, region, credsPath string, timeout int) (servregistry.ServiceRegistry, error) {
	// -- Init
	s := &servDir{
		context: ctx,
	}

	// -- Validations
	if timeout <= 0 {
		timeout = 30
	}
	s.timeout = time.Duration(timeout) * time.Second

	if len(project) == 0 {
		return nil, errors.New("project not provided")
	}
	s.project = project

	if len(region) == 0 {
		return nil, errors.New("region not provided")
	}
	s.region = region
	s.log = zap.New(zap.UseDevMode(true)).WithName("Service Directory").WithValues("project", project, "region", region)

	// -- Load the credentials
	jsonBytes, err := ioutil.ReadFile(credsPath)
	if err != nil {
		return nil, err
	}

	c, err := sd.NewRegistrationClient(s.context, option.WithCredentialsJSON(jsonBytes))
	if err != nil {
		return nil, err
	}
	s.client = c

	return s, nil
}
