package run

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"
	"time"

	"github.com/CloudNativeSDWAN/cnwan-operator/pkg/cluster"
	"google.golang.org/api/option"
)

type k8sResource struct {
	Type      string
	Namespace string
	Name      string
}

func getFileFromPathOrK8sResource(fromPath string, k8sr *k8sResource) ([]byte, error) {
	// -- Get the file from path
	if fromPath != "" {
		log.Debug().Str("path", fromPath).
			Msg("getting file...")

		byteOpts, err := ioutil.ReadFile(fromPath)
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

func retrieveCloudNetworkCfg(flagOpts *fileOrK8sResource, opts *Options) error {
	log.Info().Msg("retrieving network and/or subnetwork names from cloud...")
	if flagOpts.path == "" && flagOpts.k8s == "" {
		return fmt.Errorf("cannot infer network and/or subnetwork without credentials. Please provide it via flags or file.")
	}

	// -- Get the credentials
	var k8sres *k8sResource
	if flagOpts.k8s != "" {
		k8sres = &k8sResource{
			Type:      "secret",
			Namespace: opts.Namespace,
			Name:      flagOpts.k8s,
		}
	}
	credentialsBytes, err := getFileFromPathOrK8sResource(flagOpts.path, k8sres)
	if err != nil {
		return fmt.Errorf("cannot get credentials for cloud metadata: %w", err)
	}

	// -- Get data automatically
	platform, netwCfg, err := func() (*string, *cluster.NetworkConfiguration, error) {
		ctx, canc := context.WithTimeout(context.Background(), 15*time.Second)
		defer canc()

		switch cluster.WhereAmIRunning() {
		case cluster.GKECluster:
			platform := string(cluster.GKECluster)
			nw, err := cluster.GetNetworkFromGKE(ctx, option.WithCredentialsJSON(credentialsBytes))
			if err != nil {
				return nil, nil, fmt.Errorf("cannot get network configuration from GKE: %w", err)
			}

			return &platform, nw, nil
		case cluster.EKSCluster:
			platform := string(cluster.EKSCluster)
			nw, err := cluster.GetNetworkFromEKS(ctx)
			if err != nil {
				return nil, nil, fmt.Errorf("cannot get network configuration from EKS: %w", err)
			}

			return &platform, nw, nil
		default:
			return nil, nil, fmt.Errorf("cannot get network configuration: unsupported cluster")
		}
	}()
	if err != nil {
		return err
	}

	if platform != nil {
		opts.PersistentMetadata[platformNameMetadataKey] = *platform
	}
	if opts.CloudMetadata.Network == autoValue {
		opts.CloudMetadata.Network = netwCfg.NetworkName
		opts.PersistentMetadata[networkNameMetadataKey] = netwCfg.NetworkName
	}
	if opts.CloudMetadata.SubNetwork == autoValue {
		opts.CloudMetadata.SubNetwork = netwCfg.SubNetworkName
		opts.PersistentMetadata[subNetworkNameMetadataKey] = netwCfg.SubNetworkName
	}

	return nil
}
