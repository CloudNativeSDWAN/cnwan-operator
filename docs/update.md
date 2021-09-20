# Update

Follow the sections below to learn about the steps you need to perform before updating according to your **current** version.

If your version is not included, just follow [Simple update](#simple-update)

* [0.2.1 and below](#0.2.1-and-below)

## Simple update

Remove the operator:

```bash
./scripts/remove.sh
```

and follow installation guide again. The operator will try to perform all its work again but will stop it when it realizes that most of it was already performed on previous installation.

## 0.5.1 and below

Remove the operator:

```bash
./scripts/remove.sh
```

`monitorNamespacesByDefault` on the settings needs to be set as `true` if your previous value of `namespace.listPolicy` was `blocklist`, otherwise you can just leave it as it is.

Now, if your previous value of `namespace.listPolicy` was `allowlist` run:

```bash
for ns in $(kubectl get ns -l "operator.cnwan.io/allowed" -o jsonpath="{.items[*].metadata.name}")
do
kubectl label ns $ns operator.cnwan.io/monitor=true
kubectl label ns $ns operator.cnwan.io/allowed-
done
```

Or, if it was `blocklist`:

```bash
for ns in $(kubectl get ns -l "operator.cnwan.io/blocked" -o jsonpath="{.items[*].metadata.name}")
do
kubectl label ns $ns operator.cnwan.io/monitor=false
kubectl label ns $ns operator.cnwan.io/blocked-
done
```

<!-- TODO: write an update guide for v0.6.0 when it is going to be released:
- clone operator
- remove it
- script to replace the old labels with new one
- change the settings
 -->

## 0.2.1 and below

Clone latest `v0.2.x` version and navigate to `/scripts folder to remove it:

```bash
git clone --depth 1 --branch v0.2.1 https://github.com/CloudNativeSDWAN/cnwan-operator
cd cnwan-operator/scripts
./remove.sh
```

The *Settings* that was being used up to `v0.2.1` is deprecated and will be dropped in `v0.6.0`.
Follow the [installation guide](./install.md) carefully to re-deploy the operator with the new *Settings*.
