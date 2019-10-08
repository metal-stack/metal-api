#!/bin/sh

/kubefwd svc -n metal-control-plane -c /metal-lab/.kubeconfig &
echo "Letting kubefwd starting up..."
sleep 10

metalctl image apply -f /masterdata/images.yaml
metalctl size apply -f /masterdata/sizes.yaml
metalctl partition apply -f /masterdata/partitions.yaml

metal-api \
  --db-addr rethinkdb-rethinkdb-proxy:28015 \
  --hmac-admin-lifetime 0 \
  --nsqd-tcp-addr metal-control-plane-nsqd:4150 \
  --nsqd-rest-endpoint metal-control-plane-nsqd:4152 \
  --ipam-db-addr metal-control-plane-postgres \
  --ipam-db-password password \
  --ipam-db-user ipam \
  --ipam-db-name ipam
