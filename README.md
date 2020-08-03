# Websu - Web App performance and uptime

Speedster helps you understand whether your web applications performs well.

## Environment variables that are expected
`GCS_BUCKET`: the Google Cloud Storage bucket used for storing lighthouse json results
`GOOGLE_APPLICATION_CREDENTIALS`: the path to the service account that will
we used for writing to Google Cloud Storage.
`MONGO_URI`: the URI to used to connect to MongoDB. Default is `mongodb://localhost:27017`.
