# Configure CN-WAN Operator with Google Service Directory

This short guide is focused on configuring the CN-WAN Operator to use and configure Service Directory.

## Settings format

The included directory `deploy/settings` contains a `settings.yaml` for you to modify with the appropriate values.

For your convenience, here is how the settings for the CN-WAN Operator looks like:

```yaml
namespace:
  listPolicy: allowlist
service:
  annotations: []
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
```

We will only cover Service Directory settings here, so you can go ahead and remove the whole `etcd` settings:

```yaml
namespace:
  listPolicy: allowlist
service:
  annotations: []
serviceRegistry:
  gcpServiceDirectory:
    defaultRegion: <region>
    projectID: <project>
```

`namespace` and `service` settings are covered in the [main documentation](../configuration.md). Let's now only focus on `serviceRegistry` options.

## Service Directory settings

### Default region

This is the [region](https://cloud.google.com/compute/docs/regions-zones) where you want the CN-WAN Operator to put objects into. You should choose a region as close as possible to your cluster or the end user of Service Directory.

### Project ID

This is the *ID* of the Google project where you want to use Service Directory. It is **not** the project's *name*.

You can find this on you Google console.

## Full example

### Example 1

In this example, you are telling the CN-WAN Operator:

* to use `us-west1` as default region
* to use `project-example-1234` as the project ID.

Here is the settings example - we omit `namespace` and `service` settings for brevity:

```yaml
namespace: ...
service: ...
serviceRegistry:
  gcpServiceDirectory:
    defaultRegion: us-west1
    projectID: project-example-1234
```

## Upgrade from v0.2.0

If you were already using CN-WAN Operator *before* `v0.3.0` your settings should look like this:

```yaml
gcloud:
  serviceDirectory:
    region: <region>
    project: <project>
namespace: ...
service: ...
```

Before upgrading to `v0.3.0` please change the settings yaml as you see in [example 1](#example-1).
