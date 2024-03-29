[![Go Report Card](https://goreportcard.com/badge/github.com/Luzifer/preserve)](https://goreportcard.com/report/github.com/Luzifer/preserve)
![](https://badges.fyi/github/license/Luzifer/preserve)
![](https://badges.fyi/github/downloads/Luzifer/preserve)
![](https://badges.fyi/github/latest-release/Luzifer/preserve)
![](https://knut.in/project-status/preserve)

# Luzifer / preserve

`preserve` is a little HTTP server to preserve the presence of URLs.

Ever relied on an HTTP resource to be available and it vanished? Happened too often to me so I wrote a little tool to prevent URLs from vanishing: `preserve`.

## Usage

After you've started `preserve` it will by default listen on port 3000 and you can start using it by prefixing the URL of the resource:

Lets say you want to ensure the image `https://example.com/image.png` does not vanish:

- `http://localhost:3000/https://example.com/image.png` will fetch the resource once and then deliver it from the local cache
- `http://localhost:3000/latest/https://example.com/image.png` will fetch the resource with every request until it gets unavailable and then serve it from local cache

This also works with parameters:

`http://localhost:3000/https://pbs.twimg.com/media/somemediaid?format=jpg&name=4096x4096`

If you do have some service (like Discord) screwing up these URLs you can apply base64 URL-Encoding to them (do NOT omit the padding):

`http://localhost:3000/b64:aHR0cHM6Ly9wYnMudHdpbWcuY29tL21lZGlhL3NvbWVtZWRpYWlkP2Zvcm1hdD1qcGcmbmFtZT00MDk2eDQwOTY=`

### Select Storage Provider

**Local files**

```console
preserve \
  --listen=:3000 \
  --storage-provider=local \
  --storage-dir=/var/lib/preserve
```

**Google Cloud Storage**

```console
preserve \
  --listen=:3000 \
  --storage-provider=gcs \
  --bucket-uri=gs://mybucket/prefix
```
