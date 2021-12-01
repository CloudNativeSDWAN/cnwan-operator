# Configure CN-WAN Operator with AWS Cloud Map

## Settings format

The included directory `artifacts/settings` contains a `settings.yaml` for you to modify with the appropriate values.

For your convenience, here is how the settings for the CN-WAN Operator looks like:

```yaml
watchNamespacesByDefault: false
serviceAnnotations: []
serviceRegistry:
  etcd:
    prefix: <prefix>
    authentication: <your-authentication-type>
    endpoints:
    - host: <host-1>
      port: <port-1>
    - host: <host-2>
      port: <port-2>
  gcpServiceDirectory:
    defaultRegion: <region>
    projectID: <project>
  awsCloudMap:
    defaultRegion: <region>
cloudMetadata:
  network: auto
  subNetwork: auto
```

We will only cover Cloud Map settings here, so you can go ahead and remove some settings:

```yaml
watchNamespacesByDefault: false
serviceAnnotations: []
serviceRegistry:
  awsCloudMap:
    defaultRegion: <region>
```

`namespace` and `service` settings are covered in the [main documentation](../configuration.md). Let's now only focus on `serviceRegistry` options.

## Cloud Map settings

### Default region

This is the [region](https://aws.amazon.com/about-aws/global-infrastructure/regions_az/) where you want the CN-WAN Operator to put objects into. You should choose a region as close as possible to your cluster or the end user of Cloud Map.

## Full example

### Example 1

In this example, you are telling the CN-WAN Operator:

* to use `us-west-2` as default region

Here is the settings example - we omit `namespace` and `service` settings for brevity:

```yaml
namespace: ...
service: ...
  awsCloudMap:
    defaultRegion: us-west-2
```
