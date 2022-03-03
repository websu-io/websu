#!/usr/bin/env python3

import subprocess
import argparse
import json
import requests

parser = argparse.ArgumentParser()
parser.add_argument("action", help="the action to take. Possible values: deploy, update-frontend.",
        choices=["deploy", "update-frontend"])
parser.add_argument('--project-id', dest="project_id", help='The GCP project ID', required=True)
parser.add_argument('--image', dest="image", help='Location of docker image to use', required=True)
args = parser.parse_args()
image = args.image

regions_standard = [
        {"cloudrun_name": "lighthouse-server", "name": "us-central1", "display_name": "Iowa, US"},
        {"cloudrun_name": "lighthouse-server-asia-east1", "name": "asia-east1", "display_name": "Taiwan"},
        {"cloudrun_name": "lighthouse-server-asia-northeast1", "name": "asia-northeast1", "display_name": "Tokyo, Japan"},
        {"cloudrun_name": "lighthouse-server-asia-northeast2", "name": "asia-northeast2", "display_name": "Osaka, Japan"},
        {"cloudrun_name": "lighthouse-server-europe-north1", "name": "europe-north1", "display_name": "Finland"},
        {"cloudrun_name": "lighthouse-server-europe-west1", "name": "europe-west1", "display_name": "Belgium"},
        {"cloudrun_name": "lighthouse-server-europe-west4", "name": "europe-west4", "display_name": "Netherlands"},
        {"cloudrun_name": "lighthouse-server-us-east1", "name": "us-east1", "display_name": "South Carolina, US"},
        {"cloudrun_name": "lighthouse-server-us-east4", "name": "us-east4", "display_name": "Virginia, US"},
        {"cloudrun_name": "lighthouse-server-us-west1", "name": "us-west1", "display_name": "Oregon, US"},
]

regions_premium = [
        {"cloudrun_name": "lighthouse-server-australia-southeast1",
            "name": "australia-southeast1", "display_name": "Sydney, Australia"},
        {"cloudrun_name": "lighthouse-server-asia-east2", "name": "asia-east2", "display_name": "Hong Kong"},
        {"cloudrun_name": "lighthouse-server-asia-northeast3", "name": "asia-northeast3", "display_name": "Seoul, South Korea"},
        {"cloudrun_name": "lighthouse-server-asia-southeast1", "name": "asia-southeast1", "display_name": "Singapore"},
        {"cloudrun_name": "lighthouse-server-asia-southeast2", "name": "asia-southeast2", "display_name": "Jakarta, Indonesia"},
        {"cloudrun_name": "lighthouse-server-asia-south1", "name": "asia-south1", "display_name": "Mumbai, India"},
        {"cloudrun_name": "lighthouse-server-europe-central2", "name": "europe-central2", "display_name": "Warsaw, Poland"},
        {"cloudrun_name": "lighthouse-server-europe-west2", "name": "europe-west2", "display_name": "London, UK"},
        {"cloudrun_name": "lighthouse-server-europe-west3", "name": "europe-west3", "display_name": "Frankfurt, DE"},
        {"cloudrun_name": "lighthouse-server-europe-west6", "name": "europe-west6", "display_name": "Zurich, CH"},
        {"cloudrun_name": "lighthouse-server-northamerica-northeast1",
            "name": "northamerica-northeast1", "display_name": "Montreal, CA"},
        {"cloudrun_name": "lighthouse-server-southamerica-east1",
            "name": "southamerica-east1", "display_name": "Sao Paulo, Brazil"},
        {"cloudrun_name": "lighthouse-server-us-west2", "name": "us-west2", "display_name": "Los Angeles, USA"},
        {"cloudrun_name": "lighthouse-server-us-west3", "name": "us-west3", "display_name": "Las Vegas, USA"},
        {"cloudrun_name": "lighthouse-server-us-west4", "name": "us-west4", "display_name": "Salt Lake City, USA"},
]

regions_standard = [dict(item, secure=True, premium=False) for item in regions_standard]
regions_premium = [dict(item, secure=True, premium=True, order=90) for item in regions_premium]

if args.action == "deploy":
    for region in (regions_standard + regions_premium):
        cmd = """
    gcloud beta run deploy {cloudrun_name} \
      --image {image} \
      --memory 1024Mi --platform managed --port 50051 --timeout 120s --concurrency 1 --max-instances=20 \
      --region {name} --set-env-vars="USE_DOCKER=false" --allow-unauthenticated \
      --execution-environment=gen2""".format(
            project_id=args.project_id, image=image, **region
        )
        print("Going to run:\n", cmd)
        result = subprocess.run(cmd, capture_output=True, shell=True, check=True)
        print(result)
        cmd = "gcloud run services update-traffic {cloudrun_name} --project {project_id} \
                --to-latest --platform managed --region {name}".format(
            project_id=args.project_id, **region
        )
        result = subprocess.run(cmd, capture_output=True, shell=True, check=True)
        print(result)
## Websu API Location example:
#    {
#        "address": "lighthouse-server-australia-southeast1-2ur3kq6dkq-ts.a.run.app:443",
#        "created_at": "2020-12-27T19:41:35.745Z",
#        "display_name": "Sydney, AU",
#        "id": "5fe8e36f5b2ee6831d99e220",
#        "name": "australia-southeast1",
#        "order": 40,
#        "premium": true,
#        "secure": true
#    }#



if args.action == "update-frontend":
    cmd = "gcloud run services list --platform managed --format=json | jq '[.[]|{name: .metadata.name, url: .status.url}]'"
    cloudrun_locations_jsonstr = subprocess.check_output(cmd, shell=True, text=True)
    cloudrun_locations = json.loads(cloudrun_locations_jsonstr)
    cloudrun_locations = {loc['name']: loc for loc in cloudrun_locations}
    locations_resp = requests.get("https://api.websu.io/locations").json()
    current_locations = {loc['name']: loc for loc in locations_resp}
    for region in (regions_standard + regions_premium):
        if region['name'] not in current_locations:
            body = dict(region)
            body['address'] = cloudrun_locations[region['cloudrun_name']]['url']
            body['address'] = body['address'][8:] # strip https://
            body['address'] = body['address'] + ':443'
            del body['cloudrun_name']
            requests.post("https://api.websu.io/locations", json=body)
        else:
            # get original location retrieved from REST API
            original = current_locations[region['name']]
            # merge with the defined regions_standard and regions_premium dict
            original.update(region)
            # remove fields that are not part of REST API
            del original['cloudrun_name']
            requests.put("https://api.websu.io/locations/%s" % original["id"], json=original)
