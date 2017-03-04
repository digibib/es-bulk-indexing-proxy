# ES-bulk-indexing-proxy

A proxy intended to sit between ls.ext/services and Elasticsearch, to intercept indexing requests, and queue theese up for bulk insertion. Any other requests (search queries etc) are proxied directly.

## Why?

The bulk API makes it possible to perform many index/delete operations in a single API call. This can greatly increase the indexing speed.

## Usage

The recommended usage is to configure it to collect and dispatch batches of 1MB, or dispatch immediately if there have been no new documents queued up in the last 10 seconds, to ensure that we never risk waiting too long before resources are indexed.