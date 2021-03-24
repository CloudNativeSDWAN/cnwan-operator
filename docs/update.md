# Update

Follow the sections below to learn about the steps you need to perform before updating according to your **current** version.

If your version is not included, just follow [Simple update](#simple-update)

* [0.2.1 and below](#0.2.1-and-below)

## Simple update

Export the version you want to use:

```bash
export IMG=cnwan/cnwan-operator:v0.3.0
```

If you intend to use your own build you will have to modify the value of `IMG` above accordingly.

Run:

```bash
kubectl set image deployment/cnwan-operator-controller-manager -n cnwan-operator-system manager=$IMG --record
```

and you should see

```bash
deployment.apps/cnwan-operator-controller-manager image updated
```

## 0.2.1 and below

Clone latest `v0.2.x` version and navigate to `/scripts folder to remove it:

```bash
git clone --depth 1 --branch v0.2.1 https://github.com/CloudNativeSDWAN/cnwan-operator
cd cnwan-operator/scripts
./remove.sh
```

The *Settings* that was being used up to `v0.2.1` is deprecated and will be dropped in `v0.6.0`.
Follow [Basic installation](./basic_installation.md) carefully to re-deploy the operator with the new *Settings*.
