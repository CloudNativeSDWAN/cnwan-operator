// Copyright © 2021 Cisco
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

package cloudmap

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestPollOperationStatus(t *testing.T) {
	pollOperationFrequency = time.Millisecond
	cases := []struct {
		cli      cloudMapClientIface
		expError error
	}{
		{
			cli: func() cloudMapClientIface {
				h := &fakeCloudMapClient{}
				h._GetOperation = func(ctx context.Context, params *servicediscovery.GetOperationInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.GetOperationOutput, error) {
					return nil, fmt.Errorf("whatever-error")
				}
				return h
			}(),
			expError: fmt.Errorf("whatever-error"),
		},
		{
			cli: func() cloudMapClientIface {
				pollCount := 0
				h := &fakeCloudMapClient{}
				h._GetOperation = func(ctx context.Context, params *servicediscovery.GetOperationInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.GetOperationOutput, error) {
					if pollCount == 0 {
						pollCount++
						return &servicediscovery.GetOperationOutput{Operation: &types.Operation{Status: types.OperationStatusPending}}, nil
					}
					return &servicediscovery.GetOperationOutput{Operation: &types.Operation{Status: types.OperationStatusFail, ErrorMessage: aws.String("whatever-error")}}, nil
				}
				return h
			}(),
			expError: fmt.Errorf("whatever-error"),
		},
		{
			cli: func() cloudMapClientIface {
				h := &fakeCloudMapClient{}
				h._GetOperation = func(ctx context.Context, params *servicediscovery.GetOperationInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.GetOperationOutput, error) {
					return &servicediscovery.GetOperationOutput{Operation: &types.Operation{Status: types.OperationStatusSuccess}}, nil
				}
				return h
			}(),
		},
	}

	a := assert.New(t)
	for i, c := range cases {
		h := &Handler{Client: c.cli, mainCtx: context.Background(), log: ctrl.Log.WithName("test")}
		err := h.pollOperationStatus("whatever")

		if !a.Equal(c.expError, err) {
			a.FailNow("case failed", "case", i)
		}
	}
}
