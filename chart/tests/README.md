# Helm Unit Tests

This directory contains unit tests for the Logstash Exporter Helm chart. The tests use the [helm-unittest](https://github.com/helm-unittest/helm-unittest) plugin to verify that the chart templates render correctly and produce the expected Kubernetes resources.

## Test Files

- `configmap_test.yaml`: Tests for the ConfigMap template, comparing the full generated `config.yml` in exporter, controller and custom config modes.
- `deployment_test.yaml`: Tests for the Deployment template, covering container and image selection for both modes, probes, environment variables, scheduling, security contexts, pull secrets and service account usage.
- `naming_test.yaml`: Tests for the resource naming rules shared by all templates (release name, `fullnameOverride` and the 63 character limit).
- `rbac_test.yaml`: Tests for the RBAC resources (ClusterRole and ClusterRoleBinding), checking rules and the service account binding.
- `service_account_test.yaml`: Tests for the ServiceAccount resource, checking when it is created, its labels and annotations.
- `service_monitor_test.yaml`: Tests for the optional Prometheus Operator ServiceMonitor, checking labels, annotations, selector and scrape endpoint.
- `service_test.yaml`: Tests for the Service resource, validating port, type, selector, labels and annotations.

## Running Tests

To run the tests locally, you'll need to have the helm-unittest plugin installed:

```bash
helm plugin install https://github.com/helm-unittest/helm-unittest --verify=false
```

Then you can run the tests with:

```bash
helm unittest ../
```

Or from the project root:

```bash
make helm-test
```

## Adding New Tests

When adding new features to the Helm chart, you should add corresponding tests to verify the behavior.
Prefer asserting the exact rendered value with `equal` over `isKind` alone, so that a test fails when a value is not rendered. Tests should follow the structure:

```yaml
suite: [Test Suite Name]
templates:
  - [template-file.yaml]
release:
  name: [release-name]
  namespace: [namespace]

tests:
  - it: [test description]
    set:
      [value-to-set]: [value]
    asserts:
      - [assertion-type]:
          [assertion-parameters]
```

For more details on available assertions and test structure, see the [helm-unittest documentation](https://github.com/helm-unittest/helm-unittest/blob/main/DOCUMENT.md).