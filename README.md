# AWS Encryption Provider

[![GoDoc](https://godoc.org/sigs.k8s.io/aws-encryption-provider?status.svg)](https://godoc.org/sigs.k8s.io/aws-encryption-provider)
[![sig-aws-encryption-provider/verify](https://testgrid.k8s.io/q/summary/sig-aws-encryption-provider/verify/tests_status?style=svg)](https://testgrid.k8s.io/sig-aws-encryption-provider#verify)
[![sig-aws-encryption-provider/unit-test](https://testgrid.k8s.io/q/summary/sig-aws-encryption-provider/unit-test/tests_status?style=svg)](https://testgrid.k8s.io/sig-aws-encryption-provider#unit-test)

This repository is an implementation of the kube-apiserver [encryption provider](https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/), backed by AWS KMS.

## Use with Kubernetes

### Assumptions

The following guide makes several assumptions:

* You have an AWS account and permission to manage KMS keys
* You have management access to a Kubernetes API server
* You have already read the Kubernetes documentation page [Encrypting Secret Data at Rest](https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data/)
* You have already read the Kubernetes documentation page [Using a KMS provider for data encryption](https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/)
* The AWS KMS encryption provider will need AWS credentials configured in order to call KMS APIs. You can read more about providing credentials by reading the [AWS SDK documentation](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials) on configuring your application.

### Setup

First you'll need to create a KMS master key. For more details you can read the [KMS documentation on creating a key](https://docs.aws.amazon.com/kms/latest/developerguide/create-keys.html). Check the [KMS pricing page](https://aws.amazon.com/kms/pricing/) for up-to-date pricing information.

```bash
KEY_ID=$(aws kms create-key --query KeyMetadata.KeyId --output text)
aws kms describe-key --key-id $KEY_ID
{
    "KeyMetadata": {
        "Origin": "AWS_KMS",
        "KeyId": "1234abcd-12ab-34cd-56ef-1234567890ab",
        "Description": "",
        "KeyManager": "CUSTOMER",
        "Enabled": true,
        "KeyUsage": "ENCRYPT_DECRYPT",
        "KeyState": "Enabled",
        "CreationDate": 1502910355.475,
        "Arn": "arn:aws:kms:us-west-2:111122223333:key/1234abcd-12ab-34cd-56ef-1234567890ab",
        "AWSAccountId": "111122223333"
    }
}
```

Key aliases can be used but it is not recommended. An alias can be updated to a new key, which would break how this encryption provider works. As a result all secrets encrypted before the alias update will become unreadable.

### Deploy the aws-encryption-provider plugin

While there are numerous ways you could deploy the aws-encryption-provider
plugin, the simplest way for most installations would be a static pod on the
same node as each Kubernetes API server. Below is an example pod spec, and you
will need to replace the image, key ARN, and region to fit your requirements.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: aws-encryption-provider
  namespace: kube-system
spec:
  containers:
  - image: 111122223333.dkr.ecr.us-west-2.amazonaws.com/aws-encryption-provider:v0.0.1
    name: aws-encryption-provider
    command:
    - /aws-encryption-provider
    - --key=arn:aws:kms:us-west-2:111122223333:key/1234abcd-12ab-34cd-56ef-1234567890ab
    - --region=us-west-2
    - --listen=/var/run/kmsplugin/socket.sock
    ports:
    - containerPort: 8080
      protocol: TCP
    livenessProbe:
      httpGet:
        path: /healthz
        port: 8080
    volumeMounts:
    - mountPath: /var/run/kmsplugin
      name: var-run-kmsplugin
  volumes:
  - name: var-run-kmsplugin
    hostPath:
      path: /var/run/kmsplugin
      type: DirectoryOrCreate
```

Once you have deployed the encryption provider on all the same nodes as your API
servers, you will need to update the kube-apiserver to use the encryption
provider by setting the `--encryption-provider-config` flag and with the path to
your encryption configuration file. Below is an example:

```yaml
apiVersion: apiserver.config.k8s.io/v1
kind: EncryptionConfiguration
resources:
  - resources:
    - secrets
    providers:
    - kms:
        name: aws-encryption-provider
        endpoint: unix:///var/run/kmsplugin/socket.sock
        cachesize: 1000
        timeout: 3s
    - identity: {}
```

Don't forget, you'll need to mount the directory containing the unix socket that
the KMS server is listening on into the kube-apiserver.

### Bootstrap during cluster creation (kops)
To use encryption provider during cluster creation, you need to ensure that its running
before starting kube-apiserver. For that you need to perform the following high level steps.

Note: These steps have been verified with [kops](https://github.com/kubernetes/kops) but
it should be similar to any other cluster bootstrapping tool.

For exact kops instructions see `KOPS.md`.

#### Run aws-encryption-provider as static pod
You need to have encryption provider running before kube-apiserver, and to do that you can
use [static pods](https://kubernetes.io/docs/tasks/administer-cluster/static-pod/) functionality. For kops, static pod manifests are available at `/etc/kubernetes/manifests`. You can further use kops file assets functionality to drop
the pod spec file in that directory.

#### Use Host Network for aws-encryption-provider
As the CNI plugin is not yet available, you need to add `hostNetwork: true` to pod spec.

#### Update health port for aws-encryption-provider
When using hostNetwork, the port `8080` used by aws-encryption-provider conflicts with
kube-apiserver which also requires the same port. To fix this, add `-health-port=:8083`
to args section of pod spec above. Also change the port in `containerPort` and `livenessProbe`
sections.

#### Add /var/run/kmsplugin hostMount to api server spec
Use kops lifecycle hook to run a script/container that can update the kube-apiserver
manifest (available at /etc/kubernetes/manifests) to add `/var/run/kmsplugin` as hostMount.

#### Permissions
Ensure master IAM role has permissions to encrypt/decrypt using the kms. You can achieve this
using additionalIAMPolicies functionality of kops.

After above changes, the modified pod-spec would look like:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: aws-encryption-provider
  namespace: kube-system
spec:
  containers:
  - image: 111122223333.dkr.ecr.us-west-2.amazonaws.com/aws-encryption-provider:v0.0.1
    name: aws-encryption-provider
    command:
    - /aws-encryption-provider
    - --key=arn:aws:kms:us-west-2:111122223333:key/1234abcd-12ab-34cd-56ef-1234567890ab
    - --region=us-west-2
    - --listen=/var/run/kmsplugin/socket.sock
    - --health-port=:8083
    ports:
    - containerPort: 8083
      protocol: TCP
    livenessProbe:
      httpGet:
        path: /healthz
        port: 8083
    volumeMounts:
    - mountPath: /var/run/kmsplugin
      name: var-run-kmsplugin
  hostNetwork: true
  volumes:
  - name: var-run-kmsplugin
    hostPath:
      path: /var/run/kmsplugin
      type: DirectoryOrCreate
```

### Check that the provider plugin is working
- First we create a secret: `kubectl create secret generic secret1 -n default --from-literal=mykey=mydata`
- Then we exec into the etcd-server: `kubectl exec -it -n kube-system $(kubectl get pods -n kube-system | grep etcd-manager-main | awk '{print $1}') bash`
- `cd /opt/etcd-v3.3.10-linux-amd64/`
- Then check the contents of our secret in etcd store by running the following:
```
ETCDCTL_API=3 etcdctl \
    --key /rootfs/etc/kubernetes/pki/kube-apiserver/etcd-client.key \
    --cert  /rootfs/etc/kubernetes/pki/kube-apiserver/etcd-client.crt \
    --cacert /rootfs/etc/kubernetes/pki/kube-apiserver/etcd-ca.crt  \
    --endpoints "https://etcd-a.internal.${CLUSTER}:4001" get /registry/secrets/default/secret1
```
&nbsp;&nbsp;&nbsp;-- output should be something like:
```
0m`�He.0�cryption-provider:�1x��%�B���#JP��J���*ȝ���΂@\n�96�^��ۦ�~0| *�H��
                    `q�*�J�.P��;&~��o#�O�8m��->8L��0�C3���A7�����~���f�V�ܬ���X��_��`�H#�D��z)+�81��qW��y��`�q��}1<LF, ��N��p����i*�aC#E�߸�s������s��l�?�a
�AźR������.��8H�4�O
```

### Rotation

If you have configured your KMS master key (CMK) to have rotation enabled, AWS will
update the CMK's backing encryption key every year. (You can read more about
automatic key rotation at [the KMS documentation
page](https://docs.aws.amazon.com/kms/latest/developerguide/rotate-keys.html))
If you are using the aws-encryption-provider with an existing master key, but
want to update your cluster to use a new KMS master key, you can by roughly
following the below procedure. Be sure to read the Kubernetes documentation on
[rotating a decryption key](https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data/#rotating-a-decryption-key),
as all of those steps apply to this process.

#### Option 1 - Use single encryption provider

Update the encryption provider for each API server to set a comma-separated list of
keys for the `key` field, a comma-separated list of unix sockets for the `listen` field,
and a comma-separated list of kms versions (colon-separated) for the `kms-versions`
field. These lists must be the same size. The key of each index in the `key` list
will be associated with the unix socket at the same index of the `listen` list.
Below is an example of the updated `command` field in the encryption provider pod
spec.

```yaml
    command:
    - /aws-encryption-provider
    - --key=arn:aws:kms:us-west-2:111122223333:key/1234abcd-12ab-34cd-56ef-1234567890ab,arn:aws:kms:us-west-2:111122223333:key/4321abcd-12ab-34cd-56ef-1234567890ba
    - --listen=/var/run/kmsplugin/socket.sock,/var/run/kmsplugin/socket2.sock
    - --kms-versions=v1:v2,v1:v2
    - --region=us-west-2
    - --health-port=:8083
```

Below is an axample encryption configuration file using the new key.

```yaml
apiVersion: apiserver.config.k8s.io/v1
kind: EncryptionConfiguration
resources:
  - resources:
    - secrets
    providers:
    # using old key
    - kms:
        name: aws-encryption-provider
        endpoint: unix:///var/run/kmsplugin/socket.sock
        cachesize: 1000
        timeout: 3s
    # using new key
    - kms:
        apiVersion: v2
        name: aws-encryption-provider-2
        endpoint: unix:///var/run/kmsplugin/socket2.sock
    - identity: {}
```


#### Option 2 - Use two encryption providers

You will need to run two encryption providers on each API server using different
keys, and you must configure them to each use a different value for the `name`
field and each provider must listen on a different unix socket. Below is an
example encryption configuration file for all API servers prior to using the new
key.

```yaml
apiVersion: apiserver.config.k8s.io/v1
kind: EncryptionConfiguration
resources:
  - resources:
    - secrets
    providers:
    # provider using old key
    - kms:
        name: aws-encryption-provider
        endpoint: unix:///var/run/kmsplugin/socket.sock
        cachesize: 1000
        timeout: 3s
    # provider using new Key
    - kms:
        name: aws-encryption-provider-2
        endpoint: unix:///var/run/kmsplugin/socket2.sock
        cachesize: 1000
        timeout: 3s
    - identity: {}
```

After all API servers have been restarted and are able to decrypt using the
new key, you can switch the order of the providers with the new key at the
beginning of the list and the old key below it. After all secrets have been
re-encrypted with the new key, you can remove the encryption provider using the
old key from the list.

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack](http://slack.k8s.io/)
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-sig-aws)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).
