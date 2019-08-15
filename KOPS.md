
## Bootstrap AWS encryption provider using kops during cluster creation
This guide is an extention to bootstrap instruction on README.md

#### Set the `--encryption-provider` flag with kops
To set the kubernetes `--encryption-provider` flag with kops you must add it to the kops cluster specificition.
```yaml
kind: Cluster
spec:
  encryptionConfig: true

```
Also, you need let kops know about the encryption configuration file. To do this you have to create a kops secret with a special `encryptionconfig` flag during cluster creation, in the following order:
```bash
kops create -f <path-to-cluster-spec> --state <state-store>
kops create secret encryptionconfig -f <path-to-encryption-config-file> --state <state-store> --name <cluster-name>
kops update cluster <cluster-name> --state <state-store> --yes
```
#### Permissions
Ensure master IAM role has permissions to encrypt/decrypt using the kms. You can achieve this
using additionalIAMPolicies functionality of kops.
Example:
```yaml
kind: Cluster
spec:
  additionalPolicies:
    master: |
      [
        {
          "Effect": "Allow",
          "Action": [
            "kms:Decrypt",
            "kms:Encrypt"
          ],
          "Resource": [
            <arn-of-kms-key>
          ]
        }
      ]
```
#### Use Host Network for aws-encryption-provider
As the CNI plugin is not yet available, you need to add `hostNetwork: true` to pod spec.

#### Update health port for aws-encryption-provider
When using hostNetwork, the port `8080` used by aws-encryption-provider conflicts with
kube-apiserver which also requires the same port. To fix this, add `-health-port=:8083`
to args section like the pod specification below. Also change the port in `containerPort` and `livenessProbe`
sections.

#### Run the provider at /srv/kubernetes
Mount the provider at a directory that is already mounted by default e.g. `/srv/kubernetes/socket.sock`. This is a work around mounting a custom path using kops lifecycles. So then your `encryptionConfig` file becomes:
```yaml
kind: EncryptionConfiguration
resources:
  - resources:
    - secrets
    providers:
    - kms:
        name: aws-encryption-provider
        endpoint: unix:///srv/kubernetes/socket.sock
        cachesize: 1000
        timeout: 3s
    - identity: {}
```

#### Run aws-encryption-provider as static pod
You need to have encryption provider running before kube-apiserver, and to do that you can
use [static pods](https://kubernetes.io/docs/tasks/administer-cluster/static-pod/) functionality. For kops, static pod manifests are available at `/etc/kubernetes/manifests`. You can further use kops file assets functionality to drop 
the pod specification file in that directory. 

After the above steps, the kops file asset for the pod specification should look like the following:
```yaml
kind: Cluster
spec:
  fileAssets:
  - name: aws-encryption-provider.yaml
    ## Note if not path is specified the default path it /srv/kubernetes/assets/<name>
    path: /etc/kubernetes/manifests/aws-encryption-provider.yaml
    roles:
    - Master
    content: |
      apiVersion: v1
      kind: Pod
      metadata:
        name: aws-encryption-provider
        namespace: kube-system
      spec:
        containers:
        - image: <image-of-aws-provider>
          name: aws-encryption-provider
          command:
          - /aws-encryption-provider
          - -key=<arn-of-kms-key>
          - -region=<region-of-kms-key>
          - -listen=/srv/kubernetes/socket.sock
          - -health-port=:8083
          ports:
          - containerPort: 8083
            protocol: TCP
          livenessProbe:
            httpGet:
              path: /healthz
              port: 8083
          volumeMounts:
          - mountPath: /srv/kubernetes
            name: kmsplugin
        hostNetwork: true
        volumes:
        - name: kmsplugin
          hostPath:
            path: /srv/kubernetes
            type: DirectoryOrCreate
        nodeSelector:
          dedicated: master
        tolerations:
        - key: dedicated
          operator: Equal
          value: master
          effect: NoSchedule
```
Note: The above uses labels to make sure that the pod lives on all the same nodes as the kube-apiserver. The following is the kops specification to implement node labels for the master instance group to go with the above example:
```yaml
kind: InstanceGroup
spec:
  nodeLabels:
    dedicated: master
  role: Master
```