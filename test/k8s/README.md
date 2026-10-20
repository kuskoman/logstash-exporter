# Kubernetes e2e tests

End-to-end tests of the Kubernetes controller in a real cluster. They run in the
[Kubernetes e2e workflow](../../.github/workflows/k8s-e2e.yml) every night and can be started
manually. They take a long time, so they do not run for pull requests. Run them before releases.

The workflow builds the controller image, creates a kind cluster and installs the chart in
controller mode, watching the namespaces `e2e-a` and `e2e-b` (pods and services). The tests then
create workloads and check the controller's `/metrics` through the Kubernetes API:

- pods started after the controller, scaling up and down, deleted pods being replaced
- pods without the annotation and pods in the unwatched namespace `e2e-c` are ignored
- credentials from the annotations, and wrong credentials reported as down
- removing the annotation stops scraping, annotated services are discovered
- a real Logstash 9.5.5

Most Logstash instances are mocks: busybox `httpd` serving `fixtures/node_info.json` and
`fixtures/node_stats.json`, optionally behind basic authentication.

The tests use the `k8se2e` build tag, so `go test ./...` skips them. To run them against another
cluster with the controller installed as above:

```bash
KUBECONFIG=~/.kube/config go test -tags k8se2e -v -timeout 30m ./test/k8s
```

`E2E_CONTROLLER_NAMESPACE`, `E2E_CONTROLLER_SERVICE`, `E2E_LOGSTASH_IMAGE` and `E2E_MOCK_IMAGE`
override the defaults.
