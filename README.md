# AWS Encryption Provider

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
aws kms create-alias --alias-name alias/K8sKMSKey --target-key-id $KEY_ID
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
    - -key=arn:aws:kms:us-west-2:111122223333:key/1234abcd-12ab-34cd-56ef-1234567890ab
    - -region=us-west-2
    - -listen=/var/run/kmsplugin/socket.sock
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
provider by setting the `--encryption-provider` flag and with the path to
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
