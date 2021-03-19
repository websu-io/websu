#!/usr/bin/env python3

import subprocess
import argparse

parser = argparse.ArgumentParser()
parser.add_argument('--project-id', dest="project_id", help='The GCP project ID', required=True)
args = parser.parse_args()

regions = [
        {"name": "lighthouse-server", "region": "us-central1"},
        {"name": "lighthouse-server-asia-east1", "region": "asia-east1"},
        {"name": "lighthouse-server-australia-southeast1", "region": "australia-southeast1"},
        {"name": "lighthouse-server-europe-north1", "region": "europe-north1"},
        {"name": "lighthouse-server-europe-west4", "region": "europe-west4"},
        {"name": "lighthouse-server-us-east1", "region": "us-east1"},
]

for region in regions:
    cmd = """
gcloud run deploy {name} \
  --image us-central1-docker.pkg.dev/{project_id}/websu/lighthouse-server:latest \
  --memory 1024Mi --platform managed --port 50051 --timeout 60s --concurrency 1 --max-instances=10 \
  --region {region} --set-env-vars="USE_DOCKER=false" --allow-unauthenticated""".format(
        project_id=args.project_id, **region
    )
    result = subprocess.run(cmd, capture_output=True, shell=True, check=True)
    print(result)
