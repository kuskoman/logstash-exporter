#! /usr/bin/env python3

import os
import yaml
import json

def read_versions(file_path):
    with open(file_path, 'r') as file:
        return [line.strip() for line in file.readlines() if line.strip()]

def generate_compose(logstash_versions):
    services = {}
    base_prometheus_port = 19198
    port_mapping = {}

    for idx, version in enumerate(logstash_versions):
        version_name = version.replace('.', '_')
        prometheus_port = base_prometheus_port + idx
        port_mapping[version] = prometheus_port

        services[f'logstash_{version_name}'] = {
            'image': f'docker.elastic.co/logstash/logstash:{version}',
            'restart': 'unless-stopped',
            'volumes': [
                './.docker/logstash-matrix.conf:/usr/share/logstash/pipeline/logstash.conf:ro',
                './.docker/logstash.yml:/usr/share/logstash/config/logstash.yml:ro'
            ],
            'depends_on': ['elasticsearch'],
            'healthcheck': {
                'test': ["CMD", "bash", "-c", 'curl -Ss localhost:9600 | grep -o \'"status":"[a-z]*"\' | awk -F\':\' \'{print $2}\' | grep -q "green" || exit 1'],
                'interval': '30s',
                'timeout': '10s',
                'retries': 8
            }
        }

        services[f'exporter_{version_name}'] = {
            'build': {
                'context': '.',
                'dockerfile': 'Dockerfile'
            },
            'restart': 'unless-stopped',
            'environment': {
                f'LOGSTASH_URL': f'http://logstash_{version_name}:9600'
            },
        }

        # Prometheus configuration
        services[f'prometheus_{version_name}'] = {
            'image': 'prom/prometheus:v2.44.0',
            'restart': 'unless-stopped',
            'volumes': [
                './.docker/prometheus.yaml:/etc/prometheus/prometheus.yml:ro'
            ],
            'environment': {
                'PROMETHEUS_PORT': prometheus_port,
            },
            'ports': [
                f'{prometheus_port}:9090'
            ],
            'healthcheck': {
                'test': ["CMD", "sh", "-c", "wget --no-verbose --tries=1 --spider http://localhost:9090 || exit 1"],
                'interval': '30s',
                'timeout': '10s',
                'retries': 8
            }
        }

    services['elasticsearch'] = {
        'image': 'elasticsearch:8.8.0',
        'restart': 'unless-stopped',
        'environment': {
            'discovery.type': 'single-node',
            'xpack.security.enabled': 'false',
            'xpack.security.http.ssl.enabled': 'false',
            'xpack.security.transport.ssl.enabled': 'false',
            'ES_JAVA_OPTS': '-Xms1g -Xmx1g',
        },
        'healthcheck': {
            'test': ["CMD", "curl", "-f", "http://localhost:9200/_cluster/health?wait_for_status=yellow&timeout=30s"],
            'interval': '30s',
            'timeout': '10s',
            'retries': 8
        }
    }

    compose = {
        'version': '3.8',
        'services': services
    }

    script_dir = os.path.dirname(os.path.abspath(__file__))
    file_path = os.path.join(script_dir, '..', '..', 'docker-compose.compatibility.yml')

    with open(file_path, 'w') as file:
        yaml.dump(compose, file)

    print(json.dumps(port_mapping))

versions_file_path = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'versions.txt')
versions = read_versions(versions_file_path)
generate_compose(versions)
