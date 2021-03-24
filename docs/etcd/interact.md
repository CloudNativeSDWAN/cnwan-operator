# Interact with the service registry

This guide will show you how you can interact with etcd and create example objects on the service registry.

## Requirements

Before going further, make sure you have a working etcd cluster: if you don't have one, you can follow the [Cluster Setup](./demo_cluster_setup.md) guide.

We will suppose that your cluster has the following:

* etcd has *at least* endpoint `10.10.10.10:2379`
* authentication mode is enabled
* service registry [prefix](../concepts.md#prefix) is `/service-registry`
* user `cnwan-operator` exists
* role `cnwan-operator-role` exists and has `readwrite` access to prefix `/service-registry`
* user `cnwan-operator` has role `cnwan-operator`

## Add resources

We will interact with etcd with `etcdctl` to add service registry objects into it.

In case you have installed etcd *without* Docker, its path will be `/tmp/etcd-download-test/etcdctl`. Otherwise it will be `docker exec etcd-gcr-v<version> /bin/sh -c "/usr/local/bin/etcdctl <command>"`. For shortness, in this document we will use the former: please adapt the commands for Docker usage in case you are running etcd *with* Docker.

Let's create an *alias*, so we can make all commands shorter - edit `<password>` with the correct value:

```bash
alias etcdctl="/tmp/etcd-download-test/etcdctl --endpoints http://10.10.10.10:2379 --user cnwan-operator:<password>"
```

Now, try to do

```bash
etcdctl get /service-registry --prefix
```

And you should see an empty response because the service registry is currently empty. `--prefix` tells etcd to get all objects that a key *starting* with the provided key.

### Create a namespace

Create the following file on your machine and call it `production.yaml`, which represents a very simple namespace:

```yaml
name: production
metadata:
    env: production
```

Now, since this is a *namespace*, its key will be `/namespaces/production`.

Remember though that we want prefix `/service-registry` to be there for all service registry objects: therefore its full key will be `/service-registry/namespaces/production`. If you haven't already, please read the [service registry keys section](./concepts.md#service-registry-keys) for more information.

Now, let's insert it on etcd:

```bash
cat production.yaml | etcdctl put /service-registry/namespaces/production
```

`OK` should be displayed.

Let's now try to retrieve it:

```bash
etcdctl get /service-registry/namespaces/production
```

and you will see the same object we just put a few minutes ago. Let's move on and create a service.

### Create a service

We will actually create two services here: *payroll* and *training*, and they both belong to `production` namespace.

*payroll* has the following definition:

```yaml
name: payroll
namespaceName: production
metadata:
    traffic-profile: standard
    version: v1.3.1
    status: stable
    hash-commit: hu2gd1c127
```

and this is *training*:

```yaml
name: training
namespaceName: production
metadata:
    traffic-profile: standard
    version: v3.4.0
    status: stable
    hash-commit: hu2gd1c127
```

It doesn't really matter what you put under `metadata` for this example, but please keep `traffic-profile` there as we will use this later.

For the same reasons we said on the previous section, the two keys will be:

* `/service-registry/namespaces/production/services/payroll` for payroll
* `/service-registry/namespaces/production/services/training` for training

Create the two service on etcd:

```bash
# Create payroll
cat payroll.yaml | etcdctl put /service-registry/namespaces/production/services/payroll

# Create training
cat training.yaml | etcdctl put /service-registry/namespaces/production/services/training
```

*or* even:

```bash
# Create payroll
cat << EOF | etcdctl put /service-registry/namespaces/production/services/payroll
name: payroll
namespaceName: production
metadata:
    traffic-profile: standard
    version: v1.3.1
    status: stable
    hash-commit: hu2gd1c127
EOF

# Create training
cat << EOF | etcdctl put /service-registry/namespaces/production/services/training
name: training
namespaceName: production
metadata:
    traffic-profile: standard
    version: v3.4.0
    status: stable
    hash-commit: n5f3yjhc9o
EOF
```

Try to read them from etcd with:

```bash
# Read payroll
etcdctl get /service-registry/namespaces/production/services/payroll

# Read training
etcdctl get /service-registry/namespaces/production/services/training
```

You can also just get `/service-registry/namespaces/production` with `--prefix`, which tells etcd to get all keys that *start* with that key:

```bash
etcdctl get /service-registry/namespaces/production/ --prefix
```

and you will see everything under `production` namespace.

## Create Endpoints

Create the following endpoints:

```bash
# Create an endpoint for payroll
cat << EOF | etcdctl put /service-registry/namespaces/production/services/payroll/endpoints/payroll-1
name: payroll-1
namespaceName: production
serviceName: payroll
address: 10.11.12.13
port: 80
metadata:
    protocol: TCP
EOF

# Create an endpoint for training
cat << EOF | etcdctl put /service-registry/namespaces/production/services/training/endpoints/training-1
name: training-1
namespaceName: production
serviceName: training
address: 10.21.22.23
port: 8080
metadata:
    protocol: TCP
EOF
```

Please take a look at previous sections for considerations about keys and how to read them from etcd.

## Watch for chages

Now that you have data there, let's try something different: open a new terminal window and put it side by side with the one you used until now.

You will probably need to set an alias again on the new terminal window. For your convenience:

```bash
alias etcdctl="/tmp/etcd-download-test/etcdctl --endpoints http://10.10.10.10:2379 --user cnwan-operator:<password>"
```

On the new terminal, execute the following command, which will *watch* for changes in etcd but only for keys that start with `/service-registry/` (`--prefix` does this):

```bash
etcdctl watch /service-registry/ --prefix
```

Now, leave that window there and switch to the previous one again. You just realized that the *training* service contains video training session: `standard` is probably not a good *traffic profile* choice there. Let's change it to `video`:

```bash
cat << EOF | etcdctl put /service-registry/namespaces/production/services/training
name: training
namespaceName: production
metadata:
    traffic-profile: video
    version: v3.4.0
    status: stable
    hash-commit: n5f3yjhc9o
EOF
```

Press enter and the other window will show you the `/service-registry/namespaces/production/services/training` key along with the new object data.

## Next steps

Congratulations: you just performed tasks that the CN-WAN Operator and [Reader](https://github.com/CloudNativeSDWAN/cnwan-reader) perform automatically for you. So why don't you take a step further and [set up the CN-WAN Operator](./operator_configuration.md) to do this for you? We'll see you there :)
