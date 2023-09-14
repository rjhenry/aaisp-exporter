# Kustomize

This directory provides a set of Kustomize manifests with which to deploy the
AAISP exporter in Kubernetes.

## Usage

### Requirements

This will _not_ handle setting any secrets, but will quite happily refer to
them. You should ensure that there is a secret in the same namespace as this is
deployed, named "aaisp-exporter", with a "username" and "password" field.
