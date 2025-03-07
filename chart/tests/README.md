# Helm Unit Tests

This directory contains unit tests for the Logstash Exporter Helm chart. The tests use the [helm-unittest](https://github.com/quintush/helm-unittest) plugin to verify that the chart templates render correctly and produce the expected Kubernetes resources.

## Test Files

- `configmap_test.yaml`: Tests for the ConfigMap template, ensuring the configuration is correctly generated for both the exporter and controller modes.
- `deployment_test.yaml`: Tests for the Deployment template, verifying container configurations, image selection based on mode, and resource settings.
- `rbac_test.yaml`: Tests for the RBAC resources (ClusterRole and ClusterRoleBinding), ensuring proper permissions for Kubernetes API access.
- `service_account_test.yaml`: Tests for the ServiceAccount resource, checking name customization and annotation support.
- `service_test.yaml`: Tests for the Service resource, validating port configuration and metadata customization.

## Running Tests

To run the tests locally, you'll need to have the helm-unittest plugin installed:

```bash
helm plugin install https://github.com/quintush/helm-unittest
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

When adding new features to the Helm chart, you should add corresponding tests to verify the behavior. Tests should follow the structure:

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

For more details on available assertions and test structure, see the [helm-unittest documentation](https://github.com/quintush/helm-unittest/blob/master/DOCUMENT.md).