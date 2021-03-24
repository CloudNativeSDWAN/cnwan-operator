# Set up a demo etcd installation

This guide will help you set up an example etcd cluster that you can use with CN-WAN Operator. For the scope of this demo and keep things simple, the etcd cluster will consist of only one node - or instance.

In the first part, *Create the etcd cluster*, we will install and start etcd. The second part, *Make it more secure*, is not mandatory but **highly** suggested.

**IMPORTANT NOTE**: this guide will only help you create a **demo** cluster so that you can quickly have a working example to use with the CN-WAN Operator and is not intended to be used in production. We strongly encourage you to follow more thorough guides if you want to use etcd in production.

## Create the etcd cluster

This section will guide you through installing, setting up and start a demo etcd cluster made up of only one node.

Please note that while it will create a ready-to-use and working cluster, we **strongly** encourage you to read etcd's [official documentation](https://etcd.io/docs/latest/) to learn how to make it more robust, resilient and secure in case you want to use it in production.

### Requirements

Please make sure the machine where you plan to install etcd is reachable by the CN-WAN Operator, so that you can later follow the other guides on this documentation.

Also, although this won't be much of problem, make sure that ports `2379` and `2380` are free on that machine, as those are well-known ports [assigned by IANA](https://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.xhtml?search=etcd) to etcd and the ones that will be used throughout these guides. Nonetheless, you can also specify the port on installation phase.

### Get the IP address

First, get the IP address of the machine where you want to etcd server to listen for connections from. Depending on your operating system, this could mean running either `ip a` or `ifconfig -a` or even go through the machine's operating system settings if it has GUI.

The CN-WAN Operator will use this address to connect to etcd, so make sure it is reachable.

Export the following variables, containing the IP address and the name of this cluster:

```bash
# ETCD_IP is the machine's IP address
export ETCD_IP=<IP-address>
# ETCD_NAME is the machine's name
export ETCD_NAME=<name>
```

For example, supposing the address you chose is `10.10.10.10` and the name is `demo`, the command above would be:

```bash
export ETCD_IP=10.10.10.10
export ETCD_NAME=demo
```

### Download an official release

Visit etcd's [releases page](https://github.com/etcd-io/etcd/releases) and choose the installing method most appropriate for your operating system. Follow the instructions and execute the commands to install etcd, but don't start the etcd server yet as we will need some extra parameters that are not included in the releases page and we will do this in a few minutes.

For simplicity we will assume you installed the binary version of etcd, because commands will be much shorter and brief and thus will look more readable.

Next, if you installed etcd locally follow the [next section](#start-etcd-without-docker), otherwise go to [Start etcd *with* docker](#start-etcd-with-docker).

### Start etcd *without* docker

In case you are running etcd *without* Docker and installed etcd from the releases page, start etcd as:

```bash
/tmp/etcd-download-test/etcd \
--name ${ETCD_NAME} \
--listen-peer-urls http://${ETCD_IP}:2380 \
--listen-client-urls http://${ETCD_IP}:2379 \
--initial-advertise-peer-urls http://${ETCD_IP}:2380 \
--advertise-client-urls http://${ETCD_IP}:2379
```

From now on, we will use `etcdctl` to manage the cluster, which is located at `/tmp/etcd-download-test/etcdctl`.

Your cluster is ready and we recommend you to make it [more secure](#make-it-more-secure).

Before going any further, we once again remind you that this is a **demo** cluster: if you intend to use etcd in production we encourage you to explore how to install etcd in a better way, make it more robust, etc.

### Start etcd *with* docker

Skip this part if you installed etcd *without* Docker.

Start etcd as:

```bash
docker run \
-p 2379:2379 \
-p 2380:2380 \
--mount type=bind,source=/tmp/etcd-data.tmp,destination=/etcd-data \
--name etcd-gcr-v3.4.14 \
gcr.io/etcd-development/etcd:v3.4.14 \
/usr/local/bin/etcd \
--name ${ETCD_NAME} \
--data-dir /etcd-data \
--listen-client-urls http://${ETCD_IP}:2379 \
--advertise-client-urls http://${ETCD_IP}:2379 \
--listen-peer-urls http://${ETCD_IP}:2380 \
--initial-advertise-peer-urls http://${ETCD_IP}:2380 \
--initial-cluster s1=http://${ETCD_IP}:2380 \
--initial-cluster-token tkn \
--initial-cluster-state new \
--log-level info \
--logger zap \
--log-outputs stderr
```

From now on, we will use `etcdctl` to manage the cluster, which can be done as

```bash
docker exec etcd-gcr-v<version> /bin/sh -c "/usr/local/bin/etcdctl <command>"
```

For example, as of this writing the latest version of etcd is `v3.4.14` and thus you should use `etcdctl` like this example which prints its version:

```bash
docker exec etcd-gcr-v3.4.14 /bin/sh -c "/usr/local/bin/etcdctl version"
```

Before going any further, we once again remind you that this is a **demo** cluster: if you intend to use etcd in production we encourage you to explore how to install etcd in a better way, make it more robust, etc.

## Make it more secure

While technically only being a demo, this section is not strictly required, but will be very useful for you in case you want to use this solution in production.

At this point, your demo cluster is fully operative and can already be used as it is, but we will make one step further and make it a bit more secure by adding a dedicated user with limited access to the cluster and no ability to make sensitive modifications to etcd. The CN-WAN Operator will authenticate as this user and thus will have limited visibility to the data stored in the cluster, that is only to data for the service registry.

We will then enable *authentication mode*, which will require all users to authenticate in order to perform any operations on etcd, as now any non-authenticated user can do anything.

**Important note**: this section will only add an extra layer of security, and by any means is **not** a comprehensive guide on how to make etcd secure and/or more robust, which is out of scope of this guide and for which we suggest you refer to other guides or to the official documentation.

### etcdctl

We will use `etcdctl` to manage the cluster.

### Without docker

If you installed etcd from the releases page as we covered earlier it will be located at `/tmp/etcd-download-test/etcdctl`.

To make it much easier to write, we will create an alias:

```bash
# ETCD_IP has been defined in the "Get the IP address" section
alias etcdctl="/tmp/etcd-download-test/etcdctl --endpoints http://${ETCD_IP}:2379"
```

Notice how we also included an etcd endpoint in the command with `--endpoints`: these are the addresses and ports of your etcd servers/nodes -- in our demo cluster we only have one -- and must not be confused with a *service registry endpoint* which represents an address where to contact an application of yours.

Now you can use `etcdctl` as just:

```bash
etcdctl [command]
```

If you intend to use etcd in production you may want to install etcdctl in another path or only install it on your machine: since we are only configuring a demo cluster, this is out of scope of this guide.

### With docker

You can now use etcdctl with docker as:

```bash
# ETCD_IP has been defined in the "Get the IP address" section
docker exec etcd-gcr-v3.4.14 /bin/sh -c "/usr/local/bin/etcdctl --endpoints http://${ETCD_IP}:2379"
```

although in this guide we will be using the first one for shortness: please adapt the commands for Docker if it is your case.

In production there is not much sense to use `etcdctl` inside docker so you might just want to install it as a binary, but this is out of scope for this guide so it won't be covered.

## Add root user and root role

Execute the following command to add a `root` user, which is required to enable *authentication mode*.

```bash
etcdctl user add root
```

You will be prompted for a password and to confirm that password.

Grant `root` the *root role*, which will allow it full control on the cluster (read/write/make modifications):

```bash
etcdctl user grant-role root root
```

### Enable authentication mode

Now we're going to enable authentication mode, which will require users to be authenticated in order to perform operations on etcd. Without this, having different users on the cluster is substantially useless as all data could be accessed without authentication anyway.

**Important note**: enabling authentication mode will make **all** previous connections to etcd invalid as they will now require to be authenticated.

```bash
etcdctl auth enable
```

Try to read something from etcd now:

```bash
etcdctl get /
```

and `Error: etcdserver: user name is empty` should appear. Now try to do the same operation as the root user:

```bash
etcdctl --user root:<password> get /
```

and... nothing appears. Well, this is correct: your cluster doesn't contain any data yet, so nothing is found.

## Create new user and role

Create a new user for the CN-WAN Operator:

```bash
etcdctl --user root:<password> \
user add cnwan-operator
```

You will be prompted for a password again. You can name the user whatever you want, just as long as you remember how you called it later.

Now get the list of users and you should see a new user:

```bash
etcdctl --user root:<password> \
user list
```

Create a new role:

```bash
etcdctl --user root:<password> \
role add cnwan-operator-role
```

### Grant permissions and assign the role

Now we will give the role a set of permissions and assign it the to the `cnwan-operator` user, allowing it to perform those same operations we assign to the role.

Before doing that though, please think about the [prefix](./concepts.md#prefix) that all service registry objects will include. For example, let's suppose you want the service registry to be prefixed with `/service-registry`.

To grant the `cnwan-operator-role` that we created earlier full access to all objects with `/service-registry` prefix, execute:

```bash
etcdctl --user root:<password> \
role grant-permission cnwan-operator-role --prefix=true readwrite /service-registry
```

Assign the role to the `cnwan-operator` user:

```bash
etcdctl --user root:<password> \
user grant-role cnwan-operator cnwan-operator-role
```

Now, try to perform one of the following operations as the `cnwan-operator` user:

```bash
# Get resource / as the cnwan-operator user
etcdctl --user cnwan-operator:<password> \
get /

# Get the list of users as cnwan-operator user
etcdctl --user cnwan-operator:<password> \
user list
```

Both will return `Error: etcdserver: permission denied`.

Now try to get all resources under `/service-registry`, once again as the `cnwan-operator` user:

```bash
etcdctl --user cnwan-operator:<password> \
get /service-registry
```

And you should see an empty response, meaning that no data was found under that prefix.

## Next steps

Another way to enable authentication is to use *TLS*: this is a very valid solution and much better than authenticating with username and password, though the CN-WAN Operator does not support *TLS* yet and thus it is not covered on this guide. Follow CN-WAN Operator to know when this feature will be released.

You can now [perform some operations](./interact.md) manually to get familiar with etcd or [set up CN-WAN Operator](./operator_configuration.md) to use etcd as a service registry.
