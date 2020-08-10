[![websu.io](screenshot.png "Websu.io screenshot")](https://websu.io)
# Websu - Web speed analysis

Websu helps you understand your web applications performance.
[Lighthouse](https://github.com/GoogleChrome/lighthouse) is used for running
an analysis and simulating how a website
performs in a real browser. Websu provides an API (this git repo) and
a Web UI that consumes the API. The API is more focused on people that wish
to utilize Lighthouse as a Service for example to integrate it in their
CICD pipelines or web applications.

## Features
- Run Lighthouse and get JSON results with a simple HTTP call
- Retrieve a list of previous results
- Web UI to host your own internal Lighthouse service
- Ability to compare results (TODO)
 
## FAQ
- **Why not just use Lighthouse directly?**
    - Lighthouse provides a CLI and an extension that can be installed in
      Chrome. Lighthouse doesn't provide an HTTP API or a Web UI. Websu makes
      it easier to consume Lighthouse for both standard users and web
      developers with an HTTP API and a Web UI.


## Environment variables that are expected
- `GCS_BUCKET`: the Google Cloud Storage bucket used for storing lighthouse json results. This is optional.
- `GOOGLE_APPLICATION_CREDENTIALS`: the path to the service account that will
  we used for writing to Google Cloud Storage. Only needed when `GCS_BUCKET` is set.
- `MONGO_URI`: the URI to used to connect to MongoDB. Default is `mongodb://localhost:27017`.

## License
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
