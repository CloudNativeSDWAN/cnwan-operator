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

package cluster

import (
	"context"
	"fmt"
	"time"

	gcpmetadata "cloud.google.com/go/compute/metadata"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	awssess "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	gccontainer "google.golang.org/api/container/v1"
	gcoption "google.golang.org/api/option"
)

type ClusterManager string

const (
	GKECluster     ClusterManager = "GKE"
	EKSCluster     ClusterManager = "EKS"
	UnknownCluster ClusterManager = "UNKNOWN"

	gkeClusterNameAttr string = "cluster-name"
	eksInstanceIDAttr  string = "instance-id"
)

type NetworkConfiguration struct {
	NetworkName    string
	SubNetworkName string
}

var (
	iAmIn ClusterManager
)

func init() {
	iAmIn = WhereAmIRunning()
}

func WhereAmIRunning() ClusterManager {
	if iAmIn != "" {
		return iAmIn
	}

	if amIInGKE() {
		return GKECluster
	}

	if amIInEKS() {
		return EKSCluster
	}

	return UnknownCluster
}

func amIInEKS() bool {
	sess := awssess.Must(awssess.NewSession())

	ec2m := ec2metadata.New(sess)
	return ec2m.AvailableWithContext(context.Background())
}

func amIInGKE() bool {
	return gcpmetadata.OnGCE()
}

func GetNetworkFromGKE(ctx context.Context, opts ...gcoption.ClientOption) (*NetworkConfiguration, error) {
	if iAmIn != GKECluster {
		return nil, fmt.Errorf("not running in GKE or no permissions to get metadata from GKE")
	}

	projectID, err := gcpmetadata.ProjectID()
	if err != nil {
		return nil, err
	}

	zone, err := gcpmetadata.Zone()
	if err != nil {
		return nil, err
	}

	clusterName, err := gcpmetadata.InstanceAttributeValue(gkeClusterNameAttr)
	if err != nil {
		return nil, err
	}

	clctx, canc := context.WithTimeout(ctx, time.Minute)
	defer canc()

	clientopts := opts
	if len(clientopts) == 0 {
		clientopts = []gcoption.ClientOption{gcoption.WithScopes(gccontainer.CloudPlatformScope)}
	}

	cli, err := gccontainer.NewService(clctx, clientopts...)
	if err != nil {
		return nil, err
	}

	cluster, err := cli.Projects.Zones.Clusters.Get(projectID, zone, clusterName).Do()
	if err != nil {
		return nil, err
	}

	return &NetworkConfiguration{cluster.Network, cluster.Subnetwork}, nil
}

func GetNetworkFromEKS(ctx context.Context, cfgs ...*aws.Config) (*NetworkConfiguration, error) {
	if iAmIn != EKSCluster {
		return nil, fmt.Errorf("not running in EKS or no permissions to get metadata from EKS")
	}

	sess := awssess.Must(awssess.NewSession())
	var metcli *ec2metadata.EC2Metadata

	if len(cfgs) > 0 {
		metcli = ec2metadata.New(sess)
	} else {
		metcli = ec2metadata.New(sess, cfgs...)
	}

	instanceID, err := metcli.GetMetadataWithContext(ctx, "instance-id")
	if err != nil {
		return nil, err
	}

	region, err := metcli.RegionWithContext(ctx)
	if err != nil {
		return nil, err
	}

	ec2cli := ec2.New(sess, aws.NewConfig().WithRegion(region))
	out, err := ec2cli.DescribeInstancesWithContext(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{instanceID}),
	})
	if err != nil {
		return nil, err
	}

	if len(out.Reservations) == 0 || (len(out.Reservations) > 0 && len(out.Reservations[0].Instances) == 0) {
		return nil, fmt.Errorf("could not find currently running instance")
	}

	inst := out.Reservations[0].Instances[0]
	return &NetworkConfiguration{aws.StringValue(inst.VpcId), aws.StringValue(inst.SubnetId)}, nil
}
