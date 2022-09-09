package run

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/CloudNativeSDWAN/cnwan-operator/pkg/cluster"
	"google.golang.org/api/option"
)

const (
	defaultNamespaceName string = "cnwan-operator-system"
	namespaceEnvName     string = "CNWAN_OPERATOR_NAMESPACE"
)

type k8sResource struct {
	Type      string
	Namespace string
	Name      string
}

type fileOrK8sResource struct {
	path string
	k8s  string
}

func getFileFromPathOrK8sResource(fromPath string, k8sr *k8sResource) ([]byte, error) {
	// -- Get the file from path
	if fromPath != "" {
		log.Debug().Str("path", fromPath).
			Msg("getting file...")

		byteOpts, err := os.ReadFile(fromPath)
		if err != nil {
			return nil, fmt.Errorf(`cannot open file "%s": %w`, fromPath, err)
		}

		return byteOpts, nil
	}

	// -- Get file from configmap/secret
	ctx, canc := context.WithTimeout(context.Background(), 10*time.Second)
	defer canc()

	log.Debug().
		Str("namespace/name", path.Join(k8sr.Namespace, k8sr.Name)).
		Msg("getting resource from kubernetes...")

	var fn func(context.Context, string, string) ([][]byte, error)

	if k8sr.Type == "configmap" {
		fn = cluster.GetFilesFromConfigMap
	} else {
		fn = cluster.GetFilesFromSecret
	}

	files, err := fn(ctx, k8sr.Namespace, k8sr.Name)
	if err != nil {
		return nil, fmt.Errorf(`cannot get %s "%s": %w`,
			k8sr.Type, path.Join(k8sr.Namespace, k8sr.Name), err)
	}

	return files[0], nil
}

func retrieveCloudNetworkCfg(opts *OperatorOptions, flagOpts *fileOrK8sResource) (*cluster.NetworkConfiguration, error) {
	log.Info().Msg("retrieving network and/or subnetwork names from cloud...")
	if flagOpts.path == "" && flagOpts.k8s == "" {
		return nil,
			fmt.Errorf("cannot infer network and/or subnetwork without credentials. Please provide it via flags or file")
	}

	// -- Get the credentials
	var k8sres *k8sResource
	if flagOpts.k8s != "" {
		k8sres = &k8sResource{
			Type:      "secret",
			Namespace: getCurrentNamespace(),
			Name:      flagOpts.k8s,
		}
	}
	credentialsBytes, err := getFileFromPathOrK8sResource(flagOpts.path, k8sres)
	if err != nil {
		return nil,
			fmt.Errorf("cannot get credentials for cloud metadata: %w", err)
	}

	// -- Get data automatically
	netCfg := &cluster.NetworkConfiguration{
		NetworkName:    opts.CloudMetadata.Network,
		SubNetworkName: opts.CloudMetadata.SubNetwork,
	}

	ctx, canc := context.WithTimeout(context.Background(), 15*time.Second)
	defer canc()

	switch cluster.WhereAmIRunning() {
	case cluster.GKECluster:
		netCfg, err = cluster.GetNetworkFromGKE(ctx, option.WithCredentialsJSON(credentialsBytes))
		if err != nil {
			return nil, fmt.Errorf("cannot get network configuration from GKE: %w", err)
		}
	case cluster.EKSCluster:
		netCfg, err = cluster.GetNetworkFromEKS(ctx)
		if err != nil {
			return nil, fmt.Errorf("cannot get network configuration from EKS: %w", err)
		}
	default:
		return nil, fmt.Errorf("cannot get network configuration: unsupported cluster")
	}

	if opts.CloudMetadata.Network != autoValue {
		netCfg.NetworkName = opts.CloudMetadata.Network
	}
	if opts.CloudMetadata.SubNetwork != autoValue {
		netCfg.SubNetworkName = opts.CloudMetadata.SubNetwork
	}

	return netCfg, nil
}

func getCurrentNamespace() string {
	if nsName := os.Getenv(namespaceEnvName); nsName != "" {
		return nsName
	}

	return defaultNamespaceName
}
