# Concepts

## Table of contents

* [Service registry on etcd](#service-registry-on-etcd)
* [Service registry objects](#service-registry-objects)
* [Usage with CN-WAN Operator](#usage-with-cn-wan-operator)
* [Keys](#keys)
* [Values](#values)
* [Path-like keys and hierarchy](#path-like-keys-and-hierarchy)
* [Prefix](#prefix)
* [Service registry keys](#service-registry-keys)

## Service registry on etcd

*etcd* is a distributed and reliable Key-Value storage and works as a database where you can store your data in a simple and convenient way used in production by many companies, included *Kubernetes* for service discovery and cluster state.

If you are already using etcd for your own purposes or are exploring on doing so, you may want to store *Service Registry* objects there, as well.

## Service registry objects

The CN-WAN Operator will store *Namespace*s, *Service*s and *Endpoint*s on etcd. Please take a look at the [general service registry documentation](../service_registry.md) to learn more about them.

## Usage with CN-WAN Operator

The *CN-WAN Operator* reacts to changes that happen in your cluster and reflects those changes to the service registry, so that you always have the latest information.

To start up with CN-WAN Operator and etcd you can start with these links:

* [Set up an example etcd cluster](./demo_cluster_setup.md)
* [Perform simple operations on etcd](./interact.md)
* [Quick start with CN-WAN Operator and etcd](./quickstart.md)
* [Configure CN-WAN Operator with etcd](./operator_configuration.md)

Finally, we strongly recommend you to learn more about the [CN-WAN Operator](../concepts.md) and [how to configure it](../configuration.md).

## Keys

Keys are used as an "index" to retrieve the actual object. For example, an object called `object-1` can have a key called `object-1`.

Although this is a very valid and fine key, the best practice is to use file system [path-like keys](#path-like-keys-and-hierarchy): that is, keys that resemble a path.

Take a look at [Service registry keys](#service-registry-keys) for a thourough example.

## Values

Values are the object that you want to store with that key. When you try to retrieve it from etcd by querying its key, etcd will try its best to print it as a string, but sometimes, especially when you are storing a complex objects, the result may be unintelligible: that is fine and means that probably it is intended for a software to *unmarshal* it into an object defined by its code.

CN-WAN Operator works with objects defined in the [general service registry documentation](../service_registry.md) and you can read them even with [manual operations](./interact.md).

## Path-like keys and hierarchy

Being a *flat* key-value storage, etcd has no concept of hierarchy, so it is not really the same as a *NoSQL* database, but more similar to a *Map* or a *Dictionary*.

Hierarchy is thus enforced with the use of *absolute paths* in the key, just as it happens on your computer. The goal is to be as precise as possible in order to avoid confusion and potential overrides:

* `/environments/testing/applications/nginx/settings/host`
* `/authentication/rbac/roles/dev-role/policies/create-objects`
* `/service-registry/namespaces/production/services/payroll/endpoints/payroll-8080`

As you can see from the examples above, there is a clear hierarchy structure where objects are divided into object types or categories, until you get the "final" object.

## Prefix

A *prefix* is just a regular key in etcd. If you want to insert new objects and they all fall under the same object type, i.e. `/environments`, you can use that as a prefix -- or, *base path*. This will allow you to have a nice separation or scope for your objects and will prevent you from writing typos or long keys. For example, with respect to the previous section:

* if you are inserting application objects, you may want to specify `/environments/testing/applications` as prefix
* if you are inserting rbac roles: `/authentication/rbac/roles`
* if you are inserting service registry objects: `/service-registry`

Focusing on this last example, the CN-WAN Operator will put all objects under `/service-registry` if you don't provide another one. As we said, the prefix is just a regular key and therefore can have a value, though CN-WAN Operator will neither create it nor put values in there, but you are free to do that if you want: i.e. you can put the name of the team that is in charge of managing your *Kubernetes* cluster just to make a very simple example.

If you pass an empty string as prefix to the CN-WAN Operator or just pass `/`, then objects will only have `/` as base path. Be careful with what you set as prefix as this may potentially overwrite existing data.

## Service registry keys

Let's now define *keys* for service registry objects.

Consider the graph below: on the left you will see the actual hierarchy and how each object is organized. On the right, the corresponding key on etcd as created by the CN-WAN Operator.

```bash
service-registry                    | /service-registry
├── production                      | /service-registry/namespaces/production
│   ├── payroll                     | /service-registry/namespaces/production/services/payroll
│   │   ├── payroll-80              | /service-registry/namespaces/production/services/payroll/endpoints/payroll-80
│   │   └── payroll-8080            | /service-registry/namespaces/production/services/payroll/endpoints/payroll-8080
│   └── training                    | /service-registry/namespaces/production/services/training
│       └── tcp-8989                | /service-registry/namespaces/production/services/training/endpoints/tcp-8989
└── testing                         | /service-registry/namespaces/testing
    └── payroll                     | /service-registry/namespaces/testing/services/payroll
        └── payroll-80              | /service-registry/namespaces/testing/services/payroll/endpoints/payroll-80
```

On top level we have our prefix, or base path: `/service-registry`. Once again, the CN-WAN Operator will not actually create this key and value, and you can specify whatever you want here: in the graph above it is included only to make the hierarchy more evident.

After that, you can see that each object "inherits" the full key of its parent object and includes its object type in plural form. This will make it organized and prevent overrides when two objects have the same name but different object type.

Notice how the service `payroll` is contained in both `production` and `testing`: the key format on the right leaves no room for confusion nor overrides.
