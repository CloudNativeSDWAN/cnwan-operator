// Copyright © 2021 Cisco
//
// SPDX-License-Identifier: Apache-2.0
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
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
)

func fromTagsSliceToMap(tags []types.Tag) map[string]string {
	metadata := map[string]string{}

	for _, t := range tags {
		metadata[aws.ToString(t.Key)] = aws.ToString(t.Value)
	}

	return metadata
}

func fromMapToTagsSlice(metadata map[string]string) []types.Tag {
	tags := []types.Tag{}

	for k, v := range metadata {
		tags = append(tags, types.Tag{Key: aws.String(k), Value: aws.String(v)})
	}
	return tags
}

func (h *Handler) pollOperationStatus(operationID string) error {
	ticker := time.NewTicker(pollOperationFrequency)
	for {
		select {
		case <-h.mainCtx.Done():
			h.log.Info("context canceled: stopping checking for operation status")
			return nil
		case <-ticker.C:
			opCtx, opCanc := context.WithTimeout(h.mainCtx, 3*time.Second)
			op, err := h.Client.GetOperation(opCtx, &servicediscovery.GetOperationInput{OperationId: aws.String(operationID)})
			if err != nil {
				opCanc()
				return err
			}
			opCanc()

			if op.Operation.Status == types.OperationStatusPending || op.Operation.Status == types.OperationStatusSubmitted {
				h.log.Info("operation not completed yet")
				continue
			}

			if op.Operation.Status == types.OperationStatusFail {
				return fmt.Errorf(aws.ToString(op.Operation.ErrorMessage))
			}

			return nil
		}
	}
}

func (h *Handler) tagResource(arn string, tags []types.Tag) error {
	ctx, canc := context.WithTimeout(h.mainCtx, time.Minute)
	defer canc()

	// NOTE: this won't replace all tags apparently: if you have annotations
	// A, B and C on Cloud Map and want to just write A and B, I am not
	// entirely sure that C will be deleted by using this function.
	// Nonetheless, we don't register annotations if the user doesn't want
	// them, so for now this is good.
	_, err := h.Client.TagResource(ctx, &servicediscovery.TagResourceInput{
		ResourceARN: aws.String(arn),
		Tags:        tags,
	})

	return err
}
