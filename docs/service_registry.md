# Service registry

A service registry is basically a database of services: each registered service provides information about its instances, addresses, ports and other data. Often times, *metadata* can be registered as well, which provide additional data about the service.

Therefore, a client can log in to the service registry and *discover* registered services to know how to connect to them and get data about them. Sounds familiar? That's because it is very similar to how DNS works, but the service registry pattern is a [key concept of microservices](https://auth0.com/blog/an-introduction-to-microservices-part-3-the-service-registry/).

**Note** that this document defines how a service registry *should* be for the CN-WAN Operator to support it and, therefore, may also be used as a *guideline* for designing one on your backend; but it does not mean that *all* service registries already existing are implemented as you see here. Some of them may only be *partially* similar to this and can, thus, may not be supported by the CN-WAN Operator in future or too different and thus not supported.

Of course, this does not apply to backends as they are actually oblivious of the data you put there: in case demand is there, we may provide support for them as well in future.

## Metadata

Some service registries allow for *metadata* to be published along with the "object" that you are publishing. These are things that may not really have a meaning for the object itself but more to you, your team, your application or your use case in general. This is just list of `key: value` pairs and we're confident that examples in this document will clarify the concept more.

Examples:

* `hash-commit: asd043qv`, `branch: master` to define repository data
* `protocol: RTP`, `authentication: enabled` to define more information on how to connect to the application
* `team: best-team-in-company`, `contact: bu-manager@company.com` to define information about the team and how to get help about the application

Please keep in mind that *some* service registries may provide *partial* support for metadata, i.e. a maximum number of values or allow them only for some objects and not all.

## Which one to choose?

Which one you choose depends on different factors, including:

* restrictions on metadata if you plan to have many
* your budget
* visibility

and so on.

As we said on the main documentation, the CN-WAN Operator currently supports *Google Service Directory* and *etcd*.

## Objects

Let's now cover the objects that the CN-WAN Operator will work with. Keep in mind that some products, i.e. Google Service Directory, may work with objects that have the same names but different formats. Nonetheless, the CN-WAN Operator will provide you with data that always look like the ones included in this guide. Even more, some service registries may not even provide some of those objects: in case you want to use those, the CN-WAN Operator will try to abstract them as best as it can while still using the service registry's own format.

We will give examples in a *YAML* format since it is the one that CN-WAN adopts as default, along with a formal description of the objects with their types so that - as long as you respect field names, hierarchies and types - you can marshal and unmarshal them in your application as well.

### Namespace

A namespace, exactly like Kubernetes' definition, is like a virtual cluster or group where you contain applications/services that have similar use or share the same purpose.

Here are some example use cases for namespaces:

* an environment, i.e. `production` or `dev`
* a business unit or team, i.e. `hr` or `my-software-team`

A namespace will look like this in a *YAML* format:

```yaml
name: team
metadata:
    env: production
    manager: John Smith
```

This is a more formal description:

| Field       | Type        | Description
| ----------- | ----------- | -----------
| name        | string      | the name of the namespace
| metadata    | map (dictionary) | A list of key -> value pairs that provide more information about this namespace. Look at the example. Keys and values are both strings.

### Service

As the name suggests, a service is something that provides users/softwares a resource or that performs some operations with a final result. For simplicity, think of a service as an *application*.

It cannot exist by itself but only when it is part of a namespace. For example, two teams may have the same service with the same name but that does different things, even if slightly.

Here are some example services:

* `payroll` or `user-profile`
* `mysql` or `redis`

Note that a service can also have the same name as its parent namespace, i.e. when you don't plan on creating others.

A service looks like this in a *YAML* format:

```yaml
name: payroll
namespaceName: production
metadata:
    traffic-profile: standard
    version: v1.2.1
    maintainers: software-team
    contact: software-team@company.com
```

This is a more formal description:

| Field       | Type        | Description
| ----------- | ----------- | -----------
| name        | string      | the name of the service
| namespaceName | string      | the name of the namespace this service belongs to
| metadata    | map (dictionary) | A list of key -> value pairs that provide more information about this service. Look at the example. Keys and values are both strings.

### Endpoint

This is the actual "place" where you can reach a service/application.

It cannot exist by itself, as it obviously only has a meaning within a service.

Here are some example endpoints:

* `payroll-tcp` or `payroll-8080`
* `user-profile-internal` or `user-profile-vpn`

Note that an endpoint can also have the same name as its parent service, i.e. when you don't plan on creating others.

An endpoint looks like this in a *YAML* format:

```yaml
name: payroll-internal
serviceName: payroll
namespaceName: production
address: 10.11.12.13
port: 9876
metadata:
    protocol: UDP
    weight: 0.25
```

This is a more formal description:

| Field       | Type        | Description
| ----------- | ----------- | -----------
| name        | string      | the name of the endpoint
| serviceName | string      | the name of the service that will be reached with this endpoint
| namespaceName | string | the name of the namespace that contains the parent service (and therefore the endpoint as well)
| address | string | the IP address of the endpoint
| port | 32 bit integer | the port of the endpoint
| metadata    | map (dictionary) | A list of key -> value pairs that provide more information about this endpoint. Look at the example. Keys and values are both strings.
