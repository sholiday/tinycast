# tinycast
Automatically compress podcasts to tiny file sizes for bandwidth constrained
connections like cellular or satellite.

## Use Case

Sometimes I'm in locations where Internet connectivity is weak or expensive, but
I'd still like to listen to the latest episode of my favorite podcasts. Podcasts
can be pretty large files to download (50-100 MiB), but if you are willing to
reduce the quality they can be much smaller to download (2-10 MiB).

Instead of downloading the episode to my server in a fast connection, manually
compressing the file and copying it over, I wanted a way to download the files
in a regular podcast app, right on my phone.

## Conversion

This service takes a podcast feed and changes the links to podcasts downloads to
point to the service. When a podcast file is requested, the service transcodes
the audio file on the fly (without storing it locally first) and streams the
much smaller file to the client.

## Screenshots

![Example of search results](/doc/screenshots/search.png)

![Example of a podcast feed in Apple Podcasts](/doc/screenshots/apple-podcast.png)

## Deployment

### Docker

An example `Dockerfile`.

```Dockerfile
docker run -d \
  --name=tinycast \
  -e PORT=8082 \
  -e BASE_URL="http://example.com:8082/" \
  -p 8082:8082 \
  --restart unless-stopped \
  sholiday/tinycast
```

### Docker Compose

An example `docker-compose.yml`.

```yaml
---
version: "3"
services:
  tinycast:
    image: sholiday/tinycast:latest
    restart: unless-stopped
    environment:
      - BASE_URL=https://tinycast.example.com/
      - API_KEY=AUniqueStringForTheDeployment
```
